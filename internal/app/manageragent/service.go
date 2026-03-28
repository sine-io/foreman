package manageragent

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/sine-io/foreman/internal/app/command"
	"github.com/sine-io/foreman/internal/app/query"
	"github.com/sine-io/foreman/internal/ports"
)

type Defaults struct {
	ProjectID string
	ModuleID  string
	RepoRoot  string
	TaskType  string
	Priority  int
}

type Dependencies struct {
	Projects         ports.ProjectRepository
	Modules          ports.ModuleRepository
	Tasks            ports.TaskRepository
	Runs             ports.RunRepository
	Approvals        ports.ApprovalRepository
	CreateProject    *command.CreateProjectHandler
	CreateModule     *command.CreateModuleHandler
	CreateTask       *command.CreateTaskHandler
	DispatchTask     *command.DispatchTaskHandler
	QueryModuleBoard *query.ModuleBoardQuery
	QueryTaskBoard   *query.TaskBoardQuery
	Defaults         Defaults
}

type Service struct {
	Projects         ports.ProjectRepository
	Modules          ports.ModuleRepository
	Tasks            ports.TaskRepository
	Runs             ports.RunRepository
	Approvals        ports.ApprovalRepository
	CreateProject    *command.CreateProjectHandler
	CreateModule     *command.CreateModuleHandler
	CreateTask       *command.CreateTaskHandler
	DispatchTask     *command.DispatchTaskHandler
	QueryModuleBoard *query.ModuleBoardQuery
	QueryTaskBoard   *query.TaskBoardQuery
	Defaults         Defaults
}

func NewService(deps Dependencies) *Service {
	return &Service{
		Projects:         deps.Projects,
		Modules:          deps.Modules,
		Tasks:            deps.Tasks,
		Runs:             deps.Runs,
		Approvals:        deps.Approvals,
		CreateProject:    deps.CreateProject,
		CreateModule:     deps.CreateModule,
		CreateTask:       deps.CreateTask,
		DispatchTask:     deps.DispatchTask,
		QueryModuleBoard: deps.QueryModuleBoard,
		QueryTaskBoard:   deps.QueryTaskBoard,
		Defaults:         deps.Defaults,
	}
}

func (s *Service) Handle(ctx context.Context, req Request) (Response, error) {
	if err := ctxErr(ctx); err != nil {
		return Response{}, err
	}

	switch req.Kind {
	case "create_project":
		record, err := s.CreateProject.Handle(command.CreateProjectCommand{
			ID:       req.ProjectID,
			Name:     firstNonEmpty(req.Name, req.Summary),
			RepoRoot: firstNonEmpty(req.RepoRoot, s.Defaults.RepoRoot),
		})
		if err != nil {
			return Response{}, err
		}
		if err := ctxErr(ctx); err != nil {
			return Response{}, err
		}

		return Response{
			Kind:      "project_created",
			ProjectID: record.ID,
			Summary:   record.Name,
		}, nil
	case "create_module":
		projectID, err := s.resolveProjectID(req.ProjectID)
		if err != nil {
			return Response{}, err
		}

		record, err := s.CreateModule.Handle(command.CreateModuleCommand{
			ID:          req.ModuleID,
			ProjectID:   projectID,
			Name:        firstNonEmpty(req.Name, req.Summary),
			Description: req.Description,
		})
		if err != nil {
			return Response{}, err
		}
		if err := ctxErr(ctx); err != nil {
			return Response{}, err
		}

		return Response{
			Kind:      "module_created",
			ProjectID: record.ProjectID,
			ModuleID:  record.ID,
			Summary:   record.Name,
		}, nil
	case "create_task":
		projectID, err := s.resolveProjectID(req.ProjectID)
		if err != nil {
			return Response{}, err
		}
		moduleID, err := s.resolveModuleID(req.ModuleID, projectID)
		if err != nil {
			return Response{}, err
		}

		taskDTO, err := s.CreateTask.Handle(command.CreateTaskCommand{
			ModuleID:   moduleID,
			Title:      req.Summary,
			TaskType:   firstNonEmpty(req.TaskType, s.Defaults.TaskType, "write"),
			WriteScope: firstNonEmpty(req.WriteScope, "repo:"+projectID),
			Acceptance: firstNonEmpty(req.Acceptance, req.Summary),
			Priority:   firstPositive(req.Priority, s.Defaults.Priority, 10),
		})
		if err != nil {
			return Response{}, err
		}
		if err := ctxErr(ctx); err != nil {
			return Response{}, err
		}

		return s.dispatchResponse(taskDTO.ID, projectID, moduleID)
	case "dispatch_task":
		projectID, err := s.resolveProjectID(req.ProjectID)
		if err != nil {
			return Response{}, err
		}

		taskRecord, err := s.Tasks.Get(req.TaskID)
		if err != nil {
			return Response{}, err
		}

		moduleRecord, err := s.Modules.Get(taskRecord.ModuleID)
		if err != nil {
			return Response{}, err
		}
		if moduleRecord.ProjectID != projectID {
			return Response{}, fmt.Errorf("task %s does not belong to project %s", req.TaskID, projectID)
		}

		return s.dispatchResponse(req.TaskID, projectID, moduleRecord.ID)
	default:
		return Response{}, fmt.Errorf("unsupported manager-agent action: %s", req.Kind)
	}
}

func (s *Service) TaskStatus(ctx context.Context, projectID, taskID string) (TaskStatusView, error) {
	if err := ctxErr(ctx); err != nil {
		return TaskStatusView{}, err
	}

	resolvedProjectID, err := s.resolveProjectID(projectID)
	if err != nil {
		return TaskStatusView{}, err
	}

	taskRecord, err := s.Tasks.Get(taskID)
	if err != nil {
		return TaskStatusView{}, err
	}
	moduleRecord, err := s.Modules.Get(taskRecord.ModuleID)
	if err != nil {
		return TaskStatusView{}, err
	}
	if moduleRecord.ProjectID != resolvedProjectID {
		return TaskStatusView{}, fmt.Errorf("task %s does not belong to project %s", taskID, resolvedProjectID)
	}

	view := TaskStatusView{
		TaskID:    taskRecord.ID,
		ProjectID: moduleRecord.ProjectID,
		ModuleID:  taskRecord.ModuleID,
		Summary:   taskRecord.Summary,
		State:     string(taskRecord.State),
		Priority:  taskRecord.Priority,
	}

	runRecord, err := s.Runs.FindByTask(taskID)
	if err == nil {
		view.RunID = runRecord.ID
		view.RunState = runRecord.State
	} else if err != nil && err != sql.ErrNoRows {
		return TaskStatusView{}, err
	}

	approvalRecord, err := s.Approvals.FindPendingByTask(taskID)
	if err == sql.ErrNoRows {
		approvalRecord, err = s.Approvals.FindLatestByTask(taskID)
	}
	if err == nil {
		view.ApprovalID = approvalRecord.ID
		view.ApprovalReason = approvalRecord.Reason
		view.ApprovalState = string(approvalRecord.Status)
		view.PendingApproval = approvalRecord.Status == "pending"
	} else if err != nil && err != sql.ErrNoRows {
		return TaskStatusView{}, err
	}

	return view, nil
}

func (s *Service) BoardSnapshot(ctx context.Context, projectID string) (BoardSnapshotView, error) {
	if err := ctxErr(ctx); err != nil {
		return BoardSnapshotView{}, err
	}

	resolvedProjectID, err := s.resolveProjectID(projectID)
	if err != nil {
		return BoardSnapshotView{}, err
	}

	moduleBoard, err := s.QueryModuleBoard.Execute(resolvedProjectID)
	if err != nil {
		return BoardSnapshotView{}, err
	}

	taskBoard, err := s.QueryTaskBoard.Execute(resolvedProjectID)
	if err != nil {
		return BoardSnapshotView{}, err
	}

	view := BoardSnapshotView{
		ProjectID: resolvedProjectID,
		Modules:   map[string][]ModuleSnapshot{},
		Tasks:     map[string][]TaskSnapshot{},
	}

	for column, cards := range moduleBoard.Columns {
		for _, card := range cards {
			view.Modules[column] = append(view.Modules[column], ModuleSnapshot{
				ModuleID: card.ID,
				Name:     card.Name,
				State:    card.State,
			})
		}
	}

	for column, cards := range taskBoard.Columns {
		for _, card := range cards {
			view.Tasks[column] = append(view.Tasks[column], TaskSnapshot{
				TaskID:          card.ID,
				ModuleID:        card.ModuleID,
				Summary:         card.Summary,
				State:           card.State,
				Priority:        card.Priority,
				PendingApproval: card.PendingApproval,
			})
		}
	}

	return view, nil
}

func (s *Service) dispatchResponse(taskID, projectID, moduleID string) (Response, error) {
	taskRecord, err := s.Tasks.Get(taskID)
	if err != nil {
		return Response{}, err
	}

	result, err := s.DispatchTask.Handle(command.DispatchTaskCommand{
		TaskID:          taskID,
		RequestedAction: taskRecord.Summary,
	})
	if err != nil {
		return Response{}, err
	}

	if result.TaskState == "waiting_approval" {
		summary := taskRecord.Summary
		if result.ApprovalReason != "" {
			summary = result.ApprovalReason
		}

		return Response{
			Kind:      "approval_needed",
			ProjectID: projectID,
			ModuleID:  moduleID,
			TaskID:    taskID,
			Summary:   summary,
		}, nil
	}

	return Response{
		Kind:      responseKind(result),
		ProjectID: projectID,
		ModuleID:  moduleID,
		TaskID:    taskID,
		Summary:   taskRecord.Summary,
	}, nil
}

func (s *Service) resolveProjectID(projectID string) (string, error) {
	resolved := projectID
	if resolved == "" {
		resolved = s.Defaults.ProjectID
	}
	if resolved == "" {
		return "", fmt.Errorf("project_id is required")
	}

	if _, err := s.Projects.Get(resolved); err != nil {
		return "", err
	}

	return resolved, nil
}

func (s *Service) resolveModuleID(moduleID, projectID string) (string, error) {
	resolved := moduleID
	if resolved == "" {
		resolved = s.Defaults.ModuleID
	}
	if resolved == "" {
		return "", fmt.Errorf("module_id is required")
	}

	record, err := s.Modules.Get(resolved)
	if err != nil {
		return "", err
	}
	if record.ProjectID != projectID {
		return "", fmt.Errorf("module %s does not belong to project %s", resolved, projectID)
	}

	return resolved, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func responseKind(result command.DispatchTaskResult) string {
	if result.TaskState == "waiting_approval" {
		return "approval_needed"
	}
	if result.RunState == "completed" || result.TaskState == "completed" {
		return "completion"
	}
	return "in_progress"
}

func ctxErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}
