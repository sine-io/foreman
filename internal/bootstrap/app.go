package bootstrap

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	stdhttp "net/http"
	"os"
	"time"

	manageragent "github.com/sine-io/foreman/internal/adapters/gateway/manageragent"
	"github.com/sine-io/foreman/internal/adapters/gateway/openclaw"
	httpadapter "github.com/sine-io/foreman/internal/adapters/http"
	"github.com/sine-io/foreman/internal/adapters/runner/codex"
	"github.com/sine-io/foreman/internal/app/command"
	"github.com/sine-io/foreman/internal/app/query"
	modulepkg "github.com/sine-io/foreman/internal/domain/module"
	domainpolicy "github.com/sine-io/foreman/internal/domain/policy"
	projectpkg "github.com/sine-io/foreman/internal/domain/project"
	"github.com/sine-io/foreman/internal/infrastructure/store/sqlite"
)

type App interface {
	Serve(context.Context) error
	CreateProject(command.CreateProjectCommand) (projectpkg.Project, error)
	CreateModule(command.CreateModuleCommand) (modulepkg.Module, error)
	CreateTask(command.CreateTaskCommand) (command.TaskDTO, error)
	ApprovalQueue(projectID string) (query.ApprovalQueueView, error)
	ApproveTask(command.ApproveTaskCommand) (string, error)
	RetryTask(command.RetryTaskCommand) (string, error)
	CancelTask(command.CancelTaskCommand) (string, error)
	ReprioritizeTask(command.ReprioritizeTaskCommand) (string, error)
}

type app struct {
	Config Config

	db            *sql.DB
	repoRoot      string
	router        stdhttp.Handler
	openclaw      *openclaw.Handler
	projects      *sqlite.ProjectRepository
	modules       *sqlite.ModuleRepository
	tasks         *sqlite.TaskRepository
	runs          *sqlite.RunRepository
	artifacts     *sqlite.ArtifactRepository
	approvals     *sqlite.ApprovalRepository
	leases        *sqlite.LeaseRepository
	board         *sqlite.BoardQueryRepository
	createProject *command.CreateProjectHandler
	createModule  *command.CreateModuleHandler
	createTask    *command.CreateTaskHandler
	approveTask   *command.ApproveTaskHandler
	retryTask     *command.RetryTaskHandler
	cancelTask    *command.CancelTaskHandler
	reprioritize  *command.ReprioritizeTaskHandler
	dispatchTask  *command.DispatchTaskHandler
}

func BuildApp(cfg Config) (App, error) {
	if err := PrepareRuntime(cfg); err != nil {
		return nil, err
	}

	db, err := sqlite.Open(cfg.DBPath)
	if err != nil {
		return nil, err
	}

	repoRoot, err := os.Getwd()
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	projects := sqlite.NewProjectRepository(db)
	modules := sqlite.NewModuleRepository(db)
	tasks := sqlite.NewTaskRepository(db)
	runs := sqlite.NewRunRepository(db)
	artifacts := sqlite.NewArtifactRepository(db)
	approvals := sqlite.NewApprovalRepository(db)
	leases := sqlite.NewLeaseRepository(db)
	board := sqlite.NewBoardQueryRepository(db)

	instance := &app{
		Config:        cfg,
		db:            db,
		repoRoot:      repoRoot,
		projects:      projects,
		modules:       modules,
		tasks:         tasks,
		runs:          runs,
		artifacts:     artifacts,
		approvals:     approvals,
		leases:        leases,
		board:         board,
		createProject: command.NewCreateProjectHandler(projects),
		createModule:  command.NewCreateModuleHandler(modules),
		createTask:    command.NewCreateTaskHandler(tasks),
		approveTask:   command.NewApproveTaskHandler(approvals, tasks),
		retryTask:     command.NewRetryTaskHandler(tasks),
		cancelTask:    command.NewCancelTaskHandler(tasks),
		reprioritize:  command.NewReprioritizeTaskHandler(tasks),
	}
	instance.dispatchTask = command.NewDispatchTaskHandler(
		tasks,
		leases,
		strictPolicy{},
		codex.NewCodexAdapter(nil, repoRoot, cfg.ArtifactRoot),
		approvals,
		runs,
		artifacts,
	)
	instance.openclaw = openclaw.NewHandler(instance, nil)
	instance.router = httpadapter.NewRouter(instance)

	return instance, nil
}

func (a *app) Serve(ctx context.Context) error {
	server := &stdhttp.Server{
		Addr:    a.Config.HTTPAddr,
		Handler: a.router,
	}

	defer func() {
		_ = a.db.Close()
	}()

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	err := server.ListenAndServe()
	if errors.Is(err, stdhttp.ErrServerClosed) {
		return nil
	}

	return err
}

func (a *app) ModuleBoard(projectID string) (query.ModuleBoardView, error) {
	return query.NewModuleBoardQuery(a.board).Execute(projectID)
}

func (a *app) CreateProject(cmd command.CreateProjectCommand) (projectpkg.Project, error) {
	if cmd.RepoRoot == "" {
		cmd.RepoRoot = a.repoRoot
	}

	return a.createProject.Handle(cmd)
}

func (a *app) CreateModule(cmd command.CreateModuleCommand) (modulepkg.Module, error) {
	return a.createModule.Handle(cmd)
}

func (a *app) CreateTask(cmd command.CreateTaskCommand) (command.TaskDTO, error) {
	return a.createTask.Handle(cmd)
}

func (a *app) TaskBoard(projectID string) (query.TaskBoardView, error) {
	return query.NewTaskBoardQuery(a.board).Execute(projectID)
}

func (a *app) ApprovalQueue(projectID string) (query.ApprovalQueueView, error) {
	return query.NewApprovalQueueQuery(a.board).Execute(projectID)
}

func (a *app) RunDetail(runID string) (query.RunDetailView, error) {
	return query.NewRunDetailQuery(a.board).Execute(runID)
}

func (a *app) ApproveTask(cmd command.ApproveTaskCommand) (string, error) {
	if err := a.approveTask.Handle(cmd); err != nil {
		return "", err
	}

	return a.taskState(cmd.TaskID)
}

func (a *app) RetryTask(cmd command.RetryTaskCommand) (string, error) {
	if err := a.retryTask.Handle(cmd); err != nil {
		return "", err
	}

	return a.taskState(cmd.TaskID)
}

func (a *app) CancelTask(cmd command.CancelTaskCommand) (string, error) {
	if err := a.cancelTask.Handle(cmd); err != nil {
		return "", err
	}

	return a.taskState(cmd.TaskID)
}

func (a *app) ReprioritizeTask(cmd command.ReprioritizeTaskCommand) (string, error) {
	if err := a.reprioritize.Handle(cmd); err != nil {
		return "", err
	}

	return a.taskState(cmd.TaskID)
}

func (a *app) OpenClawCommand(ctx context.Context, env openclaw.Envelope) (openclaw.Response, error) {
	return a.openclaw.Handle(ctx, env)
}

func (a *app) Dispatch(ctx context.Context, cmd manageragent.Command) (manageragent.Result, error) {
	_ = ctx

	if err := a.ensureDefaultProject(); err != nil {
		return manageragent.Result{}, err
	}
	if err := a.ensureDefaultModule(); err != nil {
		return manageragent.Result{}, err
	}

	switch cmd.Kind {
	case "create_task":
		taskDTO, err := a.createTask.Handle(command.CreateTaskCommand{
			ModuleID:   defaultModuleID,
			Title:      cmd.Summary,
			TaskType:   "write",
			WriteScope: "repo:" + defaultProjectID,
			Acceptance: cmd.Summary,
			Priority:   10,
		})
		if err != nil {
			return manageragent.Result{}, err
		}

		result, err := a.dispatchTask.Handle(command.DispatchTaskCommand{
			TaskID:          taskDTO.ID,
			RequestedAction: cmd.Summary,
		})
		if err != nil {
			return manageragent.Result{}, err
		}

		if result.TaskState == "waiting_approval" {
			approvalRecord, err := a.approvals.Get(result.ApprovalID)
			if err != nil {
				return manageragent.Result{}, err
			}

			return manageragent.Result{
				Kind:    "approval_needed",
				TaskID:  taskDTO.ID,
				Summary: approvalRecord.Reason,
			}, nil
		}

		return manageragent.Result{
			Kind:    "completion",
			TaskID:  taskDTO.ID,
			Summary: taskDTO.Summary,
		}, nil
	default:
		return manageragent.Result{}, fmt.Errorf("unsupported openclaw action: %s", cmd.Kind)
	}
}

func (a *app) ensureDefaultProject() error {
	_, err := a.projects.Get(defaultProjectID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	_, err = a.createProject.Handle(command.CreateProjectCommand{
		ID:       defaultProjectID,
		Name:     "Demo Project",
		RepoRoot: a.repoRoot,
	})
	return err
}

func (a *app) ensureDefaultModule() error {
	_, err := a.modules.Get(defaultModuleID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	record, err := a.createModule.Handle(command.CreateModuleCommand{
		ID:          defaultModuleID,
		ProjectID:   defaultProjectID,
		Name:        "Inbox",
		Description: "OpenClaw ingress module",
	})
	if err != nil {
		return err
	}

	record.State = modulepkg.BoardStateActive
	return a.modules.Save(record)
}

func (a *app) taskState(taskID string) (string, error) {
	record, err := a.tasks.Get(taskID)
	if err != nil {
		return "", err
	}

	return string(record.State), nil
}

type strictPolicy struct{}

func (strictPolicy) Evaluate(action string) domainpolicy.Decision {
	return domainpolicy.EvaluateStrictAction(action)
}

const (
	defaultProjectID = "demo"
	defaultModuleID  = "module-default"
)
