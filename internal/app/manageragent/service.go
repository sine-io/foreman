package manageragent

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/sine-io/foreman/internal/app/command"
	"github.com/sine-io/foreman/internal/app/query"
)

type Defaults struct {
	ProjectID string
	ModuleID  string
	RepoRoot  string
	TaskType  string
	Priority  int
}

type Dependencies struct {
	CreateProject    *command.CreateProjectHandler
	CreateModule     *command.CreateModuleHandler
	CreateTask       *command.CreateTaskHandler
	DispatchTask     *command.DispatchTaskHandler
	QueryModuleBoard *query.ModuleBoardQuery
	QueryTaskBoard   *query.TaskBoardQuery
	Defaults         Defaults
}

type Service struct {
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
	_ = ctx

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
		moduleID, err := s.resolveModuleID(req.ModuleID)
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

		return s.dispatchResponse(taskDTO.ID, projectID, moduleID, req.Summary, taskDTO.Summary)
	case "dispatch_task":
		return s.dispatchResponse(req.TaskID, req.ProjectID, req.ModuleID, req.Summary, req.Summary)
	default:
		return Response{}, fmt.Errorf("unsupported manager-agent action: %s", req.Kind)
	}
}

func (s *Service) TaskStatus(ctx context.Context, taskID string) (TaskStatusView, error) {
	_ = ctx

	projectID, err := s.resolveProjectID("")
	if err != nil {
		return TaskStatusView{}, err
	}

	board, err := s.QueryTaskBoard.Execute(projectID)
	if err != nil {
		return TaskStatusView{}, err
	}

	for _, cards := range board.Columns {
		for _, card := range cards {
			if card.ID != taskID {
				continue
			}

			return TaskStatusView{
				TaskID:          card.ID,
				ModuleID:        card.ModuleID,
				Summary:         card.Summary,
				State:           card.State,
				Priority:        card.Priority,
				PendingApproval: card.PendingApproval,
			}, nil
		}
	}

	return TaskStatusView{}, sql.ErrNoRows
}

func (s *Service) BoardSnapshot(ctx context.Context, projectID string) (BoardSnapshotView, error) {
	_ = ctx

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

func (s *Service) dispatchResponse(taskID, projectID, moduleID, requestedAction, fallbackSummary string) (Response, error) {
	result, err := s.DispatchTask.Handle(command.DispatchTaskCommand{
		TaskID:          taskID,
		RequestedAction: requestedAction,
	})
	if err != nil {
		return Response{}, err
	}

	if result.TaskState == "waiting_approval" {
		summary := fallbackSummary
		if result.ApprovalID != "" && s.DispatchTask.Approvals != nil {
			approval, err := s.DispatchTask.Approvals.Get(result.ApprovalID)
			if err == nil && approval.Reason != "" {
				summary = approval.Reason
			}
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
		Kind:      "completion",
		ProjectID: projectID,
		ModuleID:  moduleID,
		TaskID:    taskID,
		Summary:   fallbackSummary,
	}, nil
}

func (s *Service) resolveProjectID(projectID string) (string, error) {
	if projectID != "" {
		return projectID, nil
	}
	if s.Defaults.ProjectID != "" {
		return s.Defaults.ProjectID, nil
	}
	return "", fmt.Errorf("project_id is required")
}

func (s *Service) resolveModuleID(moduleID string) (string, error) {
	if moduleID != "" {
		return moduleID, nil
	}
	if s.Defaults.ModuleID != "" {
		return s.Defaults.ModuleID, nil
	}
	return "", fmt.Errorf("module_id is required")
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
