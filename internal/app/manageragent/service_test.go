package manageragent

import (
	"context"
	"database/sql"
	"testing"

	"github.com/sine-io/foreman/internal/app/command"
	"github.com/sine-io/foreman/internal/app/query"
	"github.com/sine-io/foreman/internal/domain/approval"
	modulepkg "github.com/sine-io/foreman/internal/domain/module"
	domainpolicy "github.com/sine-io/foreman/internal/domain/policy"
	projectpkg "github.com/sine-io/foreman/internal/domain/project"
	"github.com/sine-io/foreman/internal/domain/task"
	"github.com/sine-io/foreman/internal/ports"
	"github.com/stretchr/testify/require"
)

func TestHandleCreateProjectReturnsCreatedProject(t *testing.T) {
	harness := newHarness()
	svc := harness.newService()

	out, err := svc.Handle(context.Background(), Request{
		Kind:      "create_project",
		SessionID: "mgr-1",
		ProjectID: "project-1",
		Name:      "Alpha",
		RepoRoot:  "/tmp/alpha",
	})
	require.NoError(t, err)
	require.Equal(t, "project_created", out.Kind)
	require.Equal(t, "project-1", out.ProjectID)

	saved, err := harness.projects.Get("project-1")
	require.NoError(t, err)
	require.Equal(t, "Alpha", saved.Name)
	require.Equal(t, "/tmp/alpha", saved.RepoRoot)
}

func TestHandleReturnsContextErrorBeforeStartingSideEffects(t *testing.T) {
	harness := newHarness()
	svc := harness.newService()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := svc.Handle(ctx, Request{
		Kind:      "create_project",
		SessionID: "mgr-1",
		ProjectID: "project-1",
		Name:      "Alpha",
		RepoRoot:  "/tmp/alpha",
	})
	require.ErrorIs(t, err, context.Canceled)

	_, err = harness.projects.Get("project-1")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestHandleCreateModuleReturnsCreatedModule(t *testing.T) {
	harness := newHarness()
	svc := harness.newService()
	harness.mustCreateProject(t, "project-1")

	out, err := svc.Handle(context.Background(), Request{
		Kind:        "create_module",
		SessionID:   "mgr-1",
		ProjectID:   "project-1",
		ModuleID:    "module-2",
		Name:        "Inbox",
		Description: "OpenClaw ingress module",
	})
	require.NoError(t, err)
	require.Equal(t, "module_created", out.Kind)
	require.Equal(t, "project-1", out.ProjectID)
	require.Equal(t, "module-2", out.ModuleID)

	saved, err := harness.modules.Get("module-2")
	require.NoError(t, err)
	require.Equal(t, "project-1", saved.ProjectID)
	require.Equal(t, "Inbox", saved.Name)
}

func TestHandleCreateTaskReturnsCompletionWhenDispatchFinishes(t *testing.T) {
	harness := newHarness()
	svc := harness.newService()
	harness.mustCreateProject(t, "project-1")
	harness.mustCreateModule(t, "module-1", "project-1")

	out, err := svc.Handle(context.Background(), Request{
		Kind:      "create_task",
		SessionID: "mgr-1",
		ProjectID: "project-1",
		ModuleID:  "module-1",
		Summary:   "Summarize the module status",
	})
	require.NoError(t, err)
	require.Equal(t, "completion", out.Kind)
	require.NotEmpty(t, out.TaskID)

	saved, err := harness.tasks.Get(out.TaskID)
	require.NoError(t, err)
	require.Equal(t, "module-1", saved.ModuleID)
	require.Equal(t, "repo:project-1", saved.WriteScope)
}

func TestHandleCreateTaskReturnsApprovalNeededWhenPolicyRequiresApproval(t *testing.T) {
	harness := newHarness()
	harness.mustCreateProject(t, "project-1")
	harness.mustCreateModule(t, "module-1", "project-1")
	harness.policyDecision = domainpolicy.Decision{
		RequiresApproval: true,
		Reason:           "git push origin main requires approval",
	}
	svc := harness.newService()

	out, err := svc.Handle(context.Background(), Request{
		Kind:      "create_task",
		SessionID: "mgr-2",
		ProjectID: "project-1",
		ModuleID:  "module-1",
		Summary:   "git push origin main",
	})
	require.NoError(t, err)
	require.Equal(t, "approval_needed", out.Kind)
	require.NotEmpty(t, out.Summary)
}

func TestHandleDispatchTaskReturnsCompletionForExistingTask(t *testing.T) {
	harness := newHarness()
	harness.mustCreateProject(t, "project-1")
	harness.mustCreateModule(t, "module-1", "project-1")
	require.NoError(t, harness.tasks.Save(task.NewTask("task-1", "module-1", task.TaskTypeWrite, "Dispatch me", "repo:project-1")))
	svc := harness.newService()

	out, err := svc.Handle(context.Background(), Request{
		Kind:    "dispatch_task",
		TaskID:  "task-1",
		Summary: "Dispatch me",
	})
	require.NoError(t, err)
	require.Equal(t, "completion", out.Kind)
	require.Equal(t, "task-1", out.TaskID)
}

func TestHandleDispatchTaskReturnsApprovalNeededForStoredRiskyTask(t *testing.T) {
	harness := newHarness()
	harness.mustCreateProject(t, "project-1")
	harness.mustCreateModule(t, "module-1", "project-1")
	riskyTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "git push origin main", "repo:project-1")
	require.NoError(t, harness.tasks.Save(riskyTask))
	harness.policyDecision = domainpolicy.Decision{
		RequiresApproval: true,
		Reason:           "git push origin main requires approval",
	}
	svc := harness.newService()

	out, err := svc.Handle(context.Background(), Request{
		Kind:      "dispatch_task",
		ProjectID: "project-1",
		TaskID:    "task-1",
		Summary:   "totally safe summary",
	})
	require.NoError(t, err)
	require.Equal(t, "approval_needed", out.Kind)
	require.Equal(t, "git push origin main requires approval", out.Summary)
}

func TestHandleDispatchTaskRunsRiskyTaskAfterApproval(t *testing.T) {
	harness := newHarness()
	harness.mustCreateProject(t, "project-1")
	harness.mustCreateModule(t, "module-1", "project-1")
	harness.policyDecision = domainpolicy.Decision{
		RequiresApproval: true,
		Reason:           "git push origin main requires approval",
	}

	riskyTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "git push origin main", "repo:project-1")
	riskyTask.State = task.TaskStateLeased
	require.NoError(t, harness.tasks.Save(riskyTask))

	approved := approval.New("approval-1", "task-1", "git push origin main requires approval")
	approved.Status = approval.StatusApproved
	require.NoError(t, harness.approvals.Save(approved))

	svc := harness.newService()

	out, err := svc.Handle(context.Background(), Request{
		Kind:      "dispatch_task",
		ProjectID: "project-1",
		TaskID:    "task-1",
	})
	require.NoError(t, err)
	require.Equal(t, "completion", out.Kind)

	run, err := harness.runs.FindByTask("task-1")
	require.NoError(t, err)
	require.Equal(t, "completed", run.State)

	view, err := svc.TaskStatus(context.Background(), "project-1", "task-1")
	require.NoError(t, err)
	require.Equal(t, "completed", view.State)
	require.Equal(t, "approved", view.ApprovalState)
	require.False(t, view.PendingApproval)
}

func TestHandleDispatchTaskReturnsInProgressWhenRunHasNotCompleted(t *testing.T) {
	harness := newHarness()
	harness.mustCreateProject(t, "project-1")
	harness.mustCreateModule(t, "module-1", "project-1")
	require.NoError(t, harness.tasks.Save(task.NewTask("task-1", "module-1", task.TaskTypeWrite, "Dispatch me", "repo:project-1")))
	harness.runnerState = "running"
	svc := harness.newService()

	out, err := svc.Handle(context.Background(), Request{
		Kind:      "dispatch_task",
		ProjectID: "project-1",
		TaskID:    "task-1",
	})
	require.NoError(t, err)
	require.Equal(t, "in_progress", out.Kind)
}

func TestHandleCreateModuleReturnsErrorWhenProjectMissing(t *testing.T) {
	harness := newHarness()
	svc := harness.newService()

	_, err := svc.Handle(context.Background(), Request{
		Kind:      "create_module",
		SessionID: "mgr-1",
		ProjectID: "project-missing",
		ModuleID:  "module-1",
		Name:      "Inbox",
	})
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestHandleCreateTaskReturnsErrorWhenModuleMissing(t *testing.T) {
	harness := newHarness()
	svc := harness.newService()
	harness.mustCreateProject(t, "project-1")

	_, err := svc.Handle(context.Background(), Request{
		Kind:      "create_task",
		SessionID: "mgr-1",
		ProjectID: "project-1",
		ModuleID:  "module-missing",
		Summary:   "Summarize the module status",
	})
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestHandleCreateTaskReturnsErrorWhenProjectMissing(t *testing.T) {
	harness := newHarness()
	svc := harness.newService()

	_, err := svc.Handle(context.Background(), Request{
		Kind:      "create_task",
		SessionID: "mgr-1",
		ProjectID: "project-missing",
		ModuleID:  "module-1",
		Summary:   "Summarize the module status",
	})
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestHandleCreateTaskReturnsErrorWhenModuleBelongsToDifferentProject(t *testing.T) {
	harness := newHarness()
	svc := harness.newService()
	harness.mustCreateProject(t, "project-1")
	harness.mustCreateProject(t, "project-2")
	harness.mustCreateModule(t, "module-2", "project-2")

	_, err := svc.Handle(context.Background(), Request{
		Kind:      "create_task",
		SessionID: "mgr-1",
		ProjectID: "project-1",
		ModuleID:  "module-2",
		Summary:   "Summarize the module status",
	})
	require.EqualError(t, err, "module module-2 does not belong to project project-1")
}

func TestTaskStatusUsesRequestedProjectBoard(t *testing.T) {
	harness := newHarness()
	harness.mustCreateProject(t, "project-1")
	harness.mustCreateProject(t, "project-2")
	harness.mustCreateModule(t, "module-1", "project-1")
	harness.mustCreateModule(t, "module-2", "project-2")
	harness.policyDecision = domainpolicy.Decision{
		RequiresApproval: true,
		Reason:           "git push origin main requires approval",
	}
	svc := harness.newService()

	out, err := svc.Handle(context.Background(), Request{
		Kind:      "create_task",
		SessionID: "mgr-2",
		ProjectID: "project-2",
		ModuleID:  "module-2",
		Summary:   "git push origin main",
	})
	require.NoError(t, err)
	require.Equal(t, "approval_needed", out.Kind)

	view, err := svc.TaskStatus(context.Background(), "project-2", out.TaskID)
	require.NoError(t, err)
	require.Equal(t, out.TaskID, view.TaskID)
	require.Equal(t, "project-2", view.ProjectID)
	require.Equal(t, "waiting_approval", view.State)
	require.True(t, view.PendingApproval)
	require.NotEmpty(t, view.ApprovalID)
	require.Equal(t, "git push origin main requires approval", view.ApprovalReason)
	require.Equal(t, "pending", view.ApprovalState)
}

func TestBoardSnapshotReturnsModuleAndTaskColumnsFromPersistedState(t *testing.T) {
	harness := newHarness()
	svc := harness.newService()
	harness.mustCreateProject(t, "project-1")
	harness.mustCreateModule(t, "module-1", "project-1")

	module, err := harness.modules.Get("module-1")
	require.NoError(t, err)
	module.State = modulepkg.BoardStateActive
	require.NoError(t, harness.modules.Save(module))

	out, err := svc.Handle(context.Background(), Request{
		Kind:      "create_task",
		SessionID: "mgr-1",
		ProjectID: "project-1",
		ModuleID:  "module-1",
		Summary:   "Bootstrap board",
	})
	require.NoError(t, err)
	require.Equal(t, "completion", out.Kind)

	status, err := svc.TaskStatus(context.Background(), "project-1", out.TaskID)
	require.NoError(t, err)
	require.Equal(t, "completed", status.State)
	require.Equal(t, "run-1", status.RunID)
	require.Equal(t, "completed", status.RunState)

	view, err := svc.BoardSnapshot(context.Background(), "project-1")
	require.NoError(t, err)
	require.Equal(t, "project-1", view.ProjectID)
	require.Len(t, view.Modules["Implementing"], 1)
	require.Len(t, view.Tasks["Done"], 1)
	require.Equal(t, out.TaskID, view.Tasks["Done"][0].TaskID)
}

func TestTaskStatusKeepsLatestApprovalAfterApprovalDecision(t *testing.T) {
	harness := newHarness()
	harness.mustCreateProject(t, "project-1")
	harness.mustCreateModule(t, "module-1", "project-1")
	harness.policyDecision = domainpolicy.Decision{
		RequiresApproval: true,
		Reason:           "git push origin main requires approval",
	}
	svc := harness.newService()

	out, err := svc.Handle(context.Background(), Request{
		Kind:      "create_task",
		SessionID: "mgr-2",
		ProjectID: "project-1",
		ModuleID:  "module-1",
		Summary:   "git push origin main",
	})
	require.NoError(t, err)
	require.Equal(t, "approval_needed", out.Kind)

	approvalRecord, err := harness.approvals.FindPendingByTask(out.TaskID)
	require.NoError(t, err)
	approvalRecord.Status = approval.StatusApproved
	require.NoError(t, harness.approvals.Save(approvalRecord))

	view, err := svc.TaskStatus(context.Background(), "project-1", out.TaskID)
	require.NoError(t, err)
	require.Equal(t, approvalRecord.ID, view.ApprovalID)
	require.Equal(t, "git push origin main requires approval", view.ApprovalReason)
	require.Equal(t, "approved", view.ApprovalState)
	require.False(t, view.PendingApproval)
}

func TestTaskWorkbenchUsesRequestedProjectBoard(t *testing.T) {
	harness := newHarness()
	harness.mustCreateProject(t, "project-1")
	harness.mustCreateProject(t, "project-2")
	harness.mustCreateModule(t, "module-1", "project-1")
	harness.mustCreateModule(t, "module-2", "project-2")
	harness.policyDecision = domainpolicy.Decision{
		RequiresApproval: true,
		Reason:           "git push origin main requires approval",
	}
	svc := harness.newService()

	out, err := svc.Handle(context.Background(), Request{
		Kind:      "create_task",
		SessionID: "mgr-2",
		ProjectID: "project-2",
		ModuleID:  "module-2",
		Summary:   "git push origin main",
	})
	require.NoError(t, err)
	require.Equal(t, "approval_needed", out.Kind)

	view, err := svc.TaskWorkbench(context.Background(), "project-2", out.TaskID)
	require.NoError(t, err)
	require.Equal(t, out.TaskID, view.TaskID)
	require.Equal(t, "project-2", view.ProjectID)
	require.Equal(t, "module-2", view.ModuleID)
	require.Equal(t, "waiting_approval", view.TaskState)
	require.Equal(t, "repo:project-2", view.WriteScope)
	require.Equal(t, "write", view.TaskType)
	require.Equal(t, "git push origin main", view.Acceptance)
	require.NotEmpty(t, view.LatestApprovalID)
	require.Equal(t, "pending", view.LatestApprovalState)
	require.Equal(t, "/board/approvals/workbench?project_id=project-2&approval_id="+view.LatestApprovalID, view.ApprovalWorkbenchURL)
	require.False(t, managerTaskWorkbenchAction(view.AvailableActions, "dispatch").Enabled)
	require.Equal(t, "Waiting approval", managerTaskWorkbenchAction(view.AvailableActions, "dispatch").DisabledReason)
}

func TestTaskWorkbenchRejectsCrossProjectTask(t *testing.T) {
	harness := newHarness()
	harness.mustCreateProject(t, "project-1")
	harness.mustCreateProject(t, "project-2")
	harness.mustCreateModule(t, "module-1", "project-1")
	harness.mustCreateModule(t, "module-2", "project-2")
	svc := harness.newService()

	out, err := svc.Handle(context.Background(), Request{
		Kind:      "create_task",
		SessionID: "mgr-1",
		ProjectID: "project-2",
		ModuleID:  "module-2",
		Summary:   "Inspect repo state",
	})
	require.NoError(t, err)

	_, err = svc.TaskWorkbench(context.Background(), "project-1", out.TaskID)
	require.EqualError(t, err, "task "+out.TaskID+" does not belong to project project-1")
}

func TestTaskWorkbenchActionRespectsProjectScope(t *testing.T) {
	harness := newHarness()
	harness.mustCreateProject(t, "project-1")
	harness.mustCreateProject(t, "project-2")
	harness.mustCreateModule(t, "module-1", "project-1")
	harness.mustCreateModule(t, "module-2", "project-2")
	svc := harness.newService()

	out, err := svc.Handle(context.Background(), Request{
		Kind:      "create_task",
		SessionID: "mgr-1",
		ProjectID: "project-2",
		ModuleID:  "module-2",
		Summary:   "Inspect repo state",
	})
	require.NoError(t, err)

	_, err = svc.DispatchTaskWorkbench(context.Background(), "project-1", out.TaskID)
	require.ErrorIs(t, err, ErrTaskActionNotFound)
}

func TestTaskWorkbenchActionReturnsConflictForDisabledAction(t *testing.T) {
	harness := newHarness()
	harness.mustCreateProject(t, "project-1")
	harness.mustCreateModule(t, "module-1", "project-1")
	harness.policyDecision = domainpolicy.Decision{
		RequiresApproval: true,
		Reason:           "git push origin main requires approval",
	}
	svc := harness.newService()

	out, err := svc.Handle(context.Background(), Request{
		Kind:      "create_task",
		SessionID: "mgr-2",
		ProjectID: "project-1",
		ModuleID:  "module-1",
		Summary:   "git push origin main",
	})
	require.NoError(t, err)
	require.Equal(t, "approval_needed", out.Kind)

	_, err = svc.DispatchTaskWorkbench(context.Background(), "project-1", out.TaskID)
	require.ErrorIs(t, err, ErrTaskActionConflict)
}

func TestTaskWorkbenchActionReturnsCompactRefreshResult(t *testing.T) {
	harness := newHarness()
	harness.mustCreateProject(t, "project-1")
	harness.mustCreateModule(t, "module-1", "project-1")
	svc := harness.newService()

	record := task.NewTask("task-ready", "module-1", task.TaskTypeWrite, "Inspect repo state", "repo:project-1")
	require.NoError(t, harness.tasks.Save(record))

	resp, err := svc.ReprioritizeTaskWorkbench(context.Background(), "project-1", "task-ready", 42)
	require.NoError(t, err)
	require.Equal(t, "task-ready", resp.TaskID)
	require.Equal(t, "ready", resp.TaskState)
	require.True(t, resp.RefreshRequired)
	require.Equal(t, "priority updated to 42", resp.Message)
}

func TestTaskWorkbenchDispatchResponseIncludesLatestRunID(t *testing.T) {
	harness := newHarness()
	harness.mustCreateProject(t, "project-1")
	harness.mustCreateModule(t, "module-1", "project-1")
	svc := harness.newService()

	record := task.NewTask("task-dispatch", "module-1", task.TaskTypeWrite, "Inspect repo state", "repo:project-1")
	require.NoError(t, harness.tasks.Save(record))

	resp, err := svc.DispatchTaskWorkbench(context.Background(), "project-1", "task-dispatch")
	require.NoError(t, err)
	require.Equal(t, "task-dispatch", resp.TaskID)
	require.Equal(t, "completed", resp.TaskState)
	require.Equal(t, "completed", resp.LatestRunState)
	require.NotEmpty(t, resp.LatestRunID)
	require.True(t, resp.RefreshRequired)
}

type serviceHarness struct {
	projects       *fakeProjectRepo
	modules        *fakeModuleRepo
	tasks          *fakeTaskRepo
	approvals      *fakeApprovalRepo
	runs           *fakeRunRepo
	board          *fakeBoardQueryRepo
	policyDecision domainpolicy.Decision
	runnerState    string
}

func newHarness() *serviceHarness {
	harness := &serviceHarness{
		projects:  newFakeProjectRepo(),
		modules:   newFakeModuleRepo(),
		tasks:     newFakeTaskRepo(),
		approvals: &fakeApprovalRepo{byTaskID: map[string]approval.Approval{}},
		runs:      &fakeRunRepo{},
	}
	harness.board = &fakeBoardQueryRepo{
		modules:   harness.modules,
		tasks:     harness.tasks,
		approvals: harness.approvals,
		runs:      harness.runs,
	}

	return harness
}

func (h *serviceHarness) newService() *Service {
	artifacts := &fakeArtifactRepo{}
	tx := managerAgentTransactor{
		tasks:     h.tasks,
		runs:      h.runs,
		approvals: h.approvals,
		artifacts: artifacts,
	}
	return NewService(Dependencies{
		Projects:  h.projects,
		Modules:   h.modules,
		Tasks:     h.tasks,
		Runs:      h.runs,
		Approvals: h.approvals,
		ApproveApprovalHandler: command.NewApproveApprovalHandler(tx, h.approvals, h.tasks, command.NewDispatchTaskHandler(
			tx,
			h.tasks,
			&fakeLeaseRepo{},
			fakePolicy{decision: h.policyDecision},
			&fakeRunner{state: firstNonEmpty(h.runnerState, "completed")},
			h.approvals,
			h.runs,
			artifacts,
		)),
		RejectApprovalHandler: command.NewRejectApprovalHandler(tx, h.approvals, h.tasks),
		RetryApprovalHandler: command.NewRetryApprovalDispatchHandler(
			h.approvals,
			h.tasks,
			command.NewDispatchTaskHandler(
				tx,
				h.tasks,
				&fakeLeaseRepo{},
				fakePolicy{decision: h.policyDecision},
				&fakeRunner{state: firstNonEmpty(h.runnerState, "completed")},
				h.approvals,
				h.runs,
				artifacts,
			),
		),
		CreateProject: command.NewCreateProjectHandler(h.projects),
		CreateModule:  command.NewCreateModuleHandler(h.projects, h.modules),
		CreateTask:    command.NewCreateTaskHandler(h.modules, h.tasks),
		DispatchTask: command.NewDispatchTaskHandler(
			tx,
			h.tasks,
			&fakeLeaseRepo{},
			fakePolicy{decision: h.policyDecision},
			&fakeRunner{state: firstNonEmpty(h.runnerState, "completed")},
			h.approvals,
			h.runs,
			artifacts,
		),
		RetryTask:                    command.NewRetryTaskHandler(h.tasks),
		CancelTask:                   command.NewCancelTaskHandler(h.tasks),
		ReprioritizeTask:             command.NewReprioritizeTaskHandler(h.tasks),
		QueryTaskStatus:              query.NewTaskStatusQueryFromRepositories(h.tasks, h.modules, h.runs, h.approvals),
		QueryTaskWorkbench:           query.NewTaskWorkbenchQuery(h.board),
		QueryModuleBoard:             query.NewModuleBoardQuery(h.board),
		QueryTaskBoard:               query.NewTaskBoardQuery(h.board),
		QueryApprovalWorkbenchQueue:  query.NewApprovalWorkbenchQueueQuery(h.board),
		QueryApprovalWorkbenchDetail: query.NewApprovalWorkbenchDetailQuery(h.board),
		Defaults: Defaults{
			ProjectID: "project-1",
			ModuleID:  "module-1",
		},
	})
}

func managerTaskWorkbenchAction(actions []TaskWorkbenchAction, id string) TaskWorkbenchAction {
	for _, action := range actions {
		if action.ActionID == id {
			return action
		}
	}
	return TaskWorkbenchAction{}
}

type managerAgentTransactor struct {
	tasks     ports.TaskRepository
	runs      ports.RunRepository
	approvals ports.ApprovalRepository
	artifacts ports.ArtifactRepository
}

func (t managerAgentTransactor) WithinTransaction(ctx context.Context, fn func(context.Context, ports.TransactionRepositories) error) error {
	return fn(ctx, ports.TransactionRepositories{
		Tasks:     t.tasks,
		Runs:      t.runs,
		Approvals: t.approvals,
		Artifacts: t.artifacts,
	})
}

func (h *serviceHarness) mustCreateProject(t *testing.T, projectID string) {
	t.Helper()

	_, err := command.NewCreateProjectHandler(h.projects).Handle(command.CreateProjectCommand{
		ID:       projectID,
		Name:     projectID,
		RepoRoot: "/tmp/" + projectID,
	})
	require.NoError(t, err)
}

func (h *serviceHarness) mustCreateModule(t *testing.T, moduleID, projectID string) {
	t.Helper()

	_, err := command.NewCreateModuleHandler(h.projects, h.modules).Handle(command.CreateModuleCommand{
		ID:          moduleID,
		ProjectID:   projectID,
		Name:        moduleID,
		Description: moduleID + " description",
	})
	require.NoError(t, err)
}

type fakeProjectRepo struct {
	byID map[string]projectpkg.Project
}

func newFakeProjectRepo() *fakeProjectRepo {
	return &fakeProjectRepo{byID: map[string]projectpkg.Project{}}
}

func (f *fakeProjectRepo) Save(value projectpkg.Project) error {
	f.byID[value.ID] = value
	return nil
}

func (f *fakeProjectRepo) Get(id string) (projectpkg.Project, error) {
	value, ok := f.byID[id]
	if !ok {
		return projectpkg.Project{}, sql.ErrNoRows
	}
	return value, nil
}

type fakeModuleRepo struct {
	byID map[string]modulepkg.Module
}

func newFakeModuleRepo() *fakeModuleRepo {
	return &fakeModuleRepo{byID: map[string]modulepkg.Module{}}
}

func (f *fakeModuleRepo) Save(value modulepkg.Module) error {
	f.byID[value.ID] = value
	return nil
}

func (f *fakeModuleRepo) Get(id string) (modulepkg.Module, error) {
	value, ok := f.byID[id]
	if !ok {
		return modulepkg.Module{}, sql.ErrNoRows
	}
	return value, nil
}

type fakeTaskRepo struct {
	byID map[string]task.Task
}

func newFakeTaskRepo() *fakeTaskRepo {
	return &fakeTaskRepo{
		byID: map[string]task.Task{},
	}
}

func (f *fakeTaskRepo) Save(value task.Task) error {
	f.byID[value.ID] = value
	return nil
}

func (f *fakeTaskRepo) Get(id string) (task.Task, error) {
	value, ok := f.byID[id]
	if !ok {
		return task.Task{}, sql.ErrNoRows
	}

	return value, nil
}

type fakeApprovalRepo struct {
	byTaskID map[string]approval.Approval
	saved    approval.Approval
}

func (f *fakeApprovalRepo) Save(value approval.Approval) error {
	f.saved = value
	f.byTaskID[value.TaskID] = value
	return nil
}

func (f *fakeApprovalRepo) Get(id string) (approval.Approval, error) {
	if f.saved.ID == id {
		return f.saved, nil
	}

	for _, value := range f.byTaskID {
		if value.ID == id {
			return value, nil
		}
	}

	return approval.Approval{}, sql.ErrNoRows
}

func (f *fakeApprovalRepo) FindPendingByTask(taskID string) (approval.Approval, error) {
	value, ok := f.byTaskID[taskID]
	if !ok {
		return approval.Approval{}, sql.ErrNoRows
	}

	if value.Status != approval.StatusPending {
		return approval.Approval{}, sql.ErrNoRows
	}

	return value, nil
}

func (f *fakeApprovalRepo) FindLatestByTask(taskID string) (approval.Approval, error) {
	value, ok := f.byTaskID[taskID]
	if !ok {
		return approval.Approval{}, sql.ErrNoRows
	}

	return value, nil
}

type fakeLeaseRepo struct{}

func (f *fakeLeaseRepo) Acquire(taskID, scopeKey string) error {
	return nil
}

func (f *fakeLeaseRepo) Release(taskID, scopeKey string) error {
	return nil
}

type fakePolicy struct {
	decision domainpolicy.Decision
}

func (f fakePolicy) Evaluate(action string) domainpolicy.Decision {
	return f.decision
}

type fakeRunner struct {
	state string
}

func (f fakeRunner) Dispatch(req ports.RunRequest) (ports.Run, error) {
	return ports.Run{
		ID:                   "run-1",
		TaskID:               req.TaskID,
		RunnerKind:           "codex",
		State:                f.state,
		AssistantSummaryPath: "artifacts/tasks/task-1/assistant_summary.txt",
	}, nil
}

func (f fakeRunner) Observe(runID string) (ports.Run, error) {
	return ports.Run{}, nil
}

func (f fakeRunner) Stop(runID string) error {
	return nil
}

type fakeRunRepo struct {
	saved ports.Run
}

func (f *fakeRunRepo) Save(run ports.Run) error {
	f.saved = run
	return nil
}

func (f *fakeRunRepo) Get(id string) (ports.Run, error) {
	if f.saved.ID != id {
		return ports.Run{}, sql.ErrNoRows
	}

	return f.saved, nil
}

func (f *fakeRunRepo) FindByTask(taskID string) (ports.Run, error) {
	if f.saved.TaskID != taskID {
		return ports.Run{}, sql.ErrNoRows
	}

	return f.saved, nil
}

type fakeArtifactRepo struct{}

func (f *fakeArtifactRepo) Create(taskID, kind, path string) (string, error) {
	return "artifact-1", nil
}

func (f *fakeArtifactRepo) Get(id string) (ports.ArtifactRecord, error) {
	if id == "" {
		return ports.ArtifactRecord{}, sql.ErrNoRows
	}

	return ports.ArtifactRecord{ID: id}, nil
}

type fakeBoardQueryRepo struct {
	modules   *fakeModuleRepo
	tasks     *fakeTaskRepo
	approvals *fakeApprovalRepo
	runs      *fakeRunRepo
}

func (f *fakeBoardQueryRepo) ListModules(projectID string) ([]ports.ModuleBoardRow, error) {
	rows := make([]ports.ModuleBoardRow, 0, len(f.modules.byID))
	for _, value := range f.modules.byID {
		if value.ProjectID != projectID {
			continue
		}

		rows = append(rows, ports.ModuleBoardRow{
			ModuleID:   value.ID,
			Name:       value.Name,
			BoardState: string(value.State),
		})
	}

	return rows, nil
}

func (f *fakeBoardQueryRepo) ListTasks(projectID string) ([]ports.TaskBoardRow, error) {
	rows := make([]ports.TaskBoardRow, 0, len(f.tasks.byID))
	for _, value := range f.tasks.byID {
		module, err := f.modules.Get(value.ModuleID)
		if err != nil || module.ProjectID != projectID {
			continue
		}

		_, approvalErr := f.approvals.FindPendingByTask(value.ID)
		rows = append(rows, ports.TaskBoardRow{
			TaskID:          value.ID,
			ModuleID:        value.ModuleID,
			Summary:         value.Summary,
			State:           string(value.State),
			Priority:        value.Priority,
			PendingApproval: approvalErr == nil,
		})
	}

	return rows, nil
}

func (f *fakeBoardQueryRepo) GetRunDetail(runID string) (ports.RunDetailRecord, error) {
	return ports.RunDetailRecord{}, sql.ErrNoRows
}

func (f *fakeBoardQueryRepo) ListApprovals(projectID string) ([]ports.ApprovalQueueRow, error) {
	return nil, nil
}

func (f *fakeBoardQueryRepo) ListApprovalWorkbenchQueue(projectID string) ([]ports.ApprovalWorkbenchQueueRow, error) {
	rows := make([]ports.ApprovalWorkbenchQueueRow, 0, len(f.approvals.byTaskID))
	for _, value := range f.approvals.byTaskID {
		if value.Status != approval.StatusPending {
			continue
		}

		taskValue, err := f.tasks.Get(value.TaskID)
		if err != nil {
			continue
		}
		module, err := f.modules.Get(taskValue.ModuleID)
		if err != nil || module.ProjectID != projectID {
			continue
		}

		rows = append(rows, ports.ApprovalWorkbenchQueueRow{
			ApprovalID: value.ID,
			TaskID:     value.TaskID,
			Summary:    taskValue.Summary,
			RiskLevel:  string(value.RiskLevel),
			Priority:   taskValue.Priority,
			CreatedAt:  value.CreatedAt,
		})
	}

	return rows, nil
}

func (f *fakeBoardQueryRepo) GetApprovalWorkbenchDetail(approvalID string) (ports.ApprovalWorkbenchDetailRow, error) {
	for _, approvalValue := range f.approvals.byTaskID {
		if approvalValue.ID != approvalID {
			continue
		}
		taskValue, err := f.tasks.Get(approvalValue.TaskID)
		if err != nil {
			return ports.ApprovalWorkbenchDetailRow{}, err
		}
		runValue, _ := f.runs.FindByTask(approvalValue.TaskID)
		return ports.ApprovalWorkbenchDetailRow{
			ApprovalID:      approvalValue.ID,
			TaskID:          approvalValue.TaskID,
			Summary:         taskValue.Summary,
			Reason:          approvalValue.Reason,
			ApprovalState:   string(approvalValue.Status),
			RiskLevel:       string(approvalValue.RiskLevel),
			PolicyRule:      approvalValue.PolicyRule,
			RejectionReason: approvalValue.RejectionReason,
			Priority:        taskValue.Priority,
			CreatedAt:       approvalValue.CreatedAt,
			TaskState:       string(taskValue.State),
			RunID:           runValue.ID,
			RunState:        runValue.State,
		}, nil
	}

	return ports.ApprovalWorkbenchDetailRow{}, sql.ErrNoRows
}

func (f *fakeBoardQueryRepo) GetTaskWorkbench(taskID string) (ports.TaskWorkbenchRow, error) {
	taskValue, err := f.tasks.Get(taskID)
	if err != nil {
		return ports.TaskWorkbenchRow{}, err
	}

	moduleValue, err := f.modules.Get(taskValue.ModuleID)
	if err != nil {
		return ports.TaskWorkbenchRow{}, err
	}

	runValue, runErr := f.runs.FindByTask(taskID)
	if runErr != nil && runErr != sql.ErrNoRows {
		return ports.TaskWorkbenchRow{}, runErr
	}

	approvalValue, approvalErr := f.approvals.FindLatestByTask(taskID)
	if approvalErr != nil && approvalErr != sql.ErrNoRows {
		return ports.TaskWorkbenchRow{}, approvalErr
	}

	row := ports.TaskWorkbenchRow{
		TaskID:         taskValue.ID,
		ProjectID:      moduleValue.ProjectID,
		ModuleID:       taskValue.ModuleID,
		Summary:        taskValue.Summary,
		TaskState:      string(taskValue.State),
		Priority:       taskValue.Priority,
		WriteScope:     taskValue.WriteScope,
		TaskType:       string(taskValue.Type),
		Acceptance:     taskValue.Acceptance,
		LatestRunID:    runValue.ID,
		LatestRunState: runValue.State,
	}
	if approvalErr == nil {
		row.LatestApprovalID = approvalValue.ID
		row.LatestApprovalState = string(approvalValue.Status)
		row.LatestApprovalReason = approvalValue.Reason
	}

	return row, nil
}
