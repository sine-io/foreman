package manageragent

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

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
	Projects                     ports.ProjectRepository
	Modules                      ports.ModuleRepository
	Tasks                        ports.TaskRepository
	Runs                         ports.RunRepository
	Approvals                    ports.ApprovalRepository
	ApproveApprovalHandler       *command.ApproveApprovalHandler
	RejectApprovalHandler        *command.RejectApprovalHandler
	RetryApprovalHandler         *command.RetryApprovalDispatchHandler
	CreateProject                *command.CreateProjectHandler
	CreateModule                 *command.CreateModuleHandler
	CreateTask                   *command.CreateTaskHandler
	DispatchTask                 *command.DispatchTaskHandler
	RetryTask                    *command.RetryTaskHandler
	CancelTask                   *command.CancelTaskHandler
	ReprioritizeTask             *command.ReprioritizeTaskHandler
	QueryTaskStatus              *query.TaskStatusQuery
	QueryRunWorkbench            *query.RunWorkbenchQuery
	QueryArtifactWorkbench       *query.ArtifactWorkbenchQuery
	QueryTaskWorkbench           *query.TaskWorkbenchQuery
	QueryModuleBoard             *query.ModuleBoardQuery
	QueryTaskBoard               *query.TaskBoardQuery
	QueryApprovalWorkbenchQueue  *query.ApprovalWorkbenchQueueQuery
	QueryApprovalWorkbenchDetail *query.ApprovalWorkbenchDetailQuery
	Defaults                     Defaults
}

type Service struct {
	Projects                     ports.ProjectRepository
	Modules                      ports.ModuleRepository
	Tasks                        ports.TaskRepository
	Runs                         ports.RunRepository
	Approvals                    ports.ApprovalRepository
	ApproveApprovalHandler       *command.ApproveApprovalHandler
	RejectApprovalHandler        *command.RejectApprovalHandler
	RetryApprovalHandler         *command.RetryApprovalDispatchHandler
	CreateProject                *command.CreateProjectHandler
	CreateModule                 *command.CreateModuleHandler
	CreateTask                   *command.CreateTaskHandler
	DispatchTask                 *command.DispatchTaskHandler
	RetryTask                    *command.RetryTaskHandler
	CancelTask                   *command.CancelTaskHandler
	ReprioritizeTask             *command.ReprioritizeTaskHandler
	QueryTaskStatus              *query.TaskStatusQuery
	QueryRunWorkbench            *query.RunWorkbenchQuery
	QueryArtifactWorkbench       *query.ArtifactWorkbenchQuery
	QueryTaskWorkbench           *query.TaskWorkbenchQuery
	QueryModuleBoard             *query.ModuleBoardQuery
	QueryTaskBoard               *query.TaskBoardQuery
	QueryApprovalWorkbenchQueue  *query.ApprovalWorkbenchQueueQuery
	QueryApprovalWorkbenchDetail *query.ApprovalWorkbenchDetailQuery
	Defaults                     Defaults
}

func NewService(deps Dependencies) *Service {
	return &Service{
		Projects:                     deps.Projects,
		Modules:                      deps.Modules,
		Tasks:                        deps.Tasks,
		Runs:                         deps.Runs,
		Approvals:                    deps.Approvals,
		ApproveApprovalHandler:       deps.ApproveApprovalHandler,
		RejectApprovalHandler:        deps.RejectApprovalHandler,
		RetryApprovalHandler:         deps.RetryApprovalHandler,
		CreateProject:                deps.CreateProject,
		CreateModule:                 deps.CreateModule,
		CreateTask:                   deps.CreateTask,
		DispatchTask:                 deps.DispatchTask,
		RetryTask:                    deps.RetryTask,
		CancelTask:                   deps.CancelTask,
		ReprioritizeTask:             deps.ReprioritizeTask,
		QueryTaskStatus:              deps.QueryTaskStatus,
		QueryRunWorkbench:            deps.QueryRunWorkbench,
		QueryArtifactWorkbench:       deps.QueryArtifactWorkbench,
		QueryTaskWorkbench:           deps.QueryTaskWorkbench,
		QueryModuleBoard:             deps.QueryModuleBoard,
		QueryTaskBoard:               deps.QueryTaskBoard,
		QueryApprovalWorkbenchQueue:  deps.QueryApprovalWorkbenchQueue,
		QueryApprovalWorkbenchDetail: deps.QueryApprovalWorkbenchDetail,
		Defaults:                     deps.Defaults,
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

	return s.QueryTaskStatus.Execute(resolvedProjectID, taskID)
}

func (s *Service) RunWorkbench(ctx context.Context, runID string) (RunWorkbenchView, error) {
	if err := ctxErr(ctx); err != nil {
		return RunWorkbenchView{}, err
	}

	return s.QueryRunWorkbench.Execute(runID)
}

func (s *Service) ArtifactWorkbench(ctx context.Context, artifactID string) (ArtifactWorkbenchView, error) {
	if err := ctxErr(ctx); err != nil {
		return ArtifactWorkbenchView{}, err
	}

	view, err := s.QueryArtifactWorkbench.Execute(artifactID)
	if err != nil {
		return ArtifactWorkbenchView{}, normalizeArtifactWorkbenchError(artifactID, err)
	}

	return view, nil
}

func (s *Service) TaskWorkbench(ctx context.Context, projectID, taskID string) (TaskWorkbenchView, error) {
	if err := ctxErr(ctx); err != nil {
		return TaskWorkbenchView{}, err
	}

	resolvedProjectID, err := s.resolveProjectID(projectID)
	if err != nil {
		return TaskWorkbenchView{}, err
	}

	view, err := s.QueryTaskWorkbench.Execute(resolvedProjectID, taskID)
	if err != nil {
		return TaskWorkbenchView{}, normalizeTaskActionError(taskID, resolvedProjectID, err)
	}
	return view, nil
}

func (s *Service) DispatchTaskWorkbench(ctx context.Context, projectID, taskID string) (TaskWorkbenchActionResponse, error) {
	view, err := s.taskWorkbenchForAction(ctx, projectID, taskID)
	if err != nil {
		return TaskWorkbenchActionResponse{}, err
	}
	if action := taskWorkbenchAction(view.AvailableActions, "dispatch"); !action.Enabled {
		return TaskWorkbenchActionResponse{}, fmt.Errorf("%w: dispatch unavailable: %s", ErrTaskActionConflict, action.DisabledReason)
	}

	result, err := s.DispatchTask.Handle(command.DispatchTaskCommand{
		TaskID:          taskID,
		RequestedAction: view.Summary,
	})
	if err != nil {
		return TaskWorkbenchActionResponse{}, err
	}
	_ = result
	return s.taskWorkbenchActionResponse(ctx, projectID, taskID, "")
}

func (s *Service) RetryTaskWorkbench(ctx context.Context, projectID, taskID string) (TaskWorkbenchActionResponse, error) {
	view, err := s.taskWorkbenchForAction(ctx, projectID, taskID)
	if err != nil {
		return TaskWorkbenchActionResponse{}, err
	}
	if action := taskWorkbenchAction(view.AvailableActions, "retry"); !action.Enabled {
		return TaskWorkbenchActionResponse{}, fmt.Errorf("%w: retry unavailable: %s", ErrTaskActionConflict, action.DisabledReason)
	}

	if err := s.RetryTask.Handle(command.RetryTaskCommand{TaskID: taskID}); err != nil {
		return TaskWorkbenchActionResponse{}, err
	}
	return s.taskWorkbenchActionResponse(ctx, projectID, taskID, "")
}

func (s *Service) CancelTaskWorkbench(ctx context.Context, projectID, taskID string) (TaskWorkbenchActionResponse, error) {
	view, err := s.taskWorkbenchForAction(ctx, projectID, taskID)
	if err != nil {
		return TaskWorkbenchActionResponse{}, err
	}
	if view.LatestRunState == "running" {
		return TaskWorkbenchActionResponse{}, fmt.Errorf("%w: cancel unavailable: Run in progress", ErrTaskActionConflict)
	}
	if action := taskWorkbenchAction(view.AvailableActions, "cancel"); !action.Enabled {
		return TaskWorkbenchActionResponse{}, fmt.Errorf("%w: cancel unavailable: %s", ErrTaskActionConflict, action.DisabledReason)
	}

	if err := s.CancelTask.Handle(command.CancelTaskCommand{TaskID: taskID}); err != nil {
		return TaskWorkbenchActionResponse{}, err
	}
	return s.taskWorkbenchActionResponse(ctx, projectID, taskID, "")
}

func (s *Service) ReprioritizeTaskWorkbench(ctx context.Context, projectID, taskID string, priority int) (TaskWorkbenchActionResponse, error) {
	view, err := s.taskWorkbenchForAction(ctx, projectID, taskID)
	if err != nil {
		return TaskWorkbenchActionResponse{}, err
	}
	if action := taskWorkbenchAction(view.AvailableActions, "reprioritize"); !action.Enabled {
		return TaskWorkbenchActionResponse{}, fmt.Errorf("%w: reprioritize unavailable: %s", ErrTaskActionConflict, action.DisabledReason)
	}

	if err := s.ReprioritizeTask.Handle(command.ReprioritizeTaskCommand{TaskID: taskID, Priority: priority}); err != nil {
		return TaskWorkbenchActionResponse{}, err
	}
	return s.taskWorkbenchActionResponse(ctx, projectID, taskID, fmt.Sprintf("priority updated to %d", priority))
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

func (s *Service) ApprovalWorkbenchQueue(ctx context.Context, projectID string) (ApprovalWorkbenchQueueView, error) {
	if err := ctxErr(ctx); err != nil {
		return ApprovalWorkbenchQueueView{}, err
	}

	resolvedProjectID, err := s.resolveProjectID(projectID)
	if err != nil {
		return ApprovalWorkbenchQueueView{}, err
	}

	return s.QueryApprovalWorkbenchQueue.Execute(resolvedProjectID)
}

func (s *Service) ApprovalWorkbenchDetail(ctx context.Context, approvalID string) (ApprovalWorkbenchDetailView, error) {
	if err := ctxErr(ctx); err != nil {
		return ApprovalWorkbenchDetailView{}, err
	}

	view, err := s.QueryApprovalWorkbenchDetail.Execute(approvalID)
	if err != nil {
		return ApprovalWorkbenchDetailView{}, normalizeApprovalActionError(err)
	}

	return view, nil
}

func (s *Service) ApproveApproval(ctx context.Context, approvalID string) (ApprovalWorkbenchActionResponse, error) {
	if err := ctxErr(ctx); err != nil {
		return ApprovalWorkbenchActionResponse{}, err
	}

	result, err := s.ApproveApprovalHandler.Handle(command.ApproveApprovalCommand{ApprovalID: approvalID})
	if err != nil {
		return ApprovalWorkbenchActionResponse{}, normalizeApprovalActionError(err)
	}

	return approvalActionResponse(result), nil
}

func (s *Service) RejectApproval(ctx context.Context, approvalID, rejectionReason string) (ApprovalWorkbenchActionResponse, error) {
	if err := ctxErr(ctx); err != nil {
		return ApprovalWorkbenchActionResponse{}, err
	}

	result, err := s.RejectApprovalHandler.Handle(command.RejectApprovalCommand{
		ApprovalID: approvalID,
		Reason:     rejectionReason,
	})
	if err != nil {
		return ApprovalWorkbenchActionResponse{}, normalizeApprovalActionError(err)
	}

	return approvalActionResponse(result), nil
}

func (s *Service) RetryApprovalDispatch(ctx context.Context, approvalID string) (ApprovalWorkbenchActionResponse, error) {
	if err := ctxErr(ctx); err != nil {
		return ApprovalWorkbenchActionResponse{}, err
	}

	result, err := s.RetryApprovalHandler.Handle(command.RetryApprovalDispatchCommand{ApprovalID: approvalID})
	if err != nil {
		return ApprovalWorkbenchActionResponse{}, normalizeApprovalActionError(err)
	}

	return approvalActionResponse(result), nil
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

func approvalActionResponse(result command.ApprovalActionResult) ApprovalWorkbenchActionResponse {
	return ApprovalWorkbenchActionResponse{
		ApprovalID:      result.ApprovalID,
		ApprovalState:   result.ApprovalStatus,
		RejectionReason: result.RejectionReason,
		TaskID:          result.TaskID,
		TaskState:       result.TaskState,
		RunID:           result.RunID,
		RunState:        result.RunState,
	}
}

func normalizeApprovalActionError(err error) error {
	switch {
	case errors.Is(err, command.ErrApprovalActionNotFound):
		return err
	case errors.Is(err, command.ErrApprovalActionConflict):
		return err
	case errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("%w: %v", command.ErrApprovalActionNotFound, err)
	default:
		return err
	}
}

func normalizeTaskActionError(taskID, projectID string, err error) error {
	switch {
	case errors.Is(err, ErrTaskActionNotFound):
		return err
	case errors.Is(err, ErrTaskActionConflict):
		return err
	case errors.Is(err, command.ErrTaskActionConflict):
		return ErrTaskActionConflict
	case errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("%w: task %s not found in project %s", ErrTaskActionNotFound, taskID, projectID)
	case strings.Contains(err.Error(), "does not belong to project"):
		return fmt.Errorf("%w: task %s not found in project %s", ErrTaskActionNotFound, taskID, projectID)
	default:
		return err
	}
}

func normalizeArtifactWorkbenchError(artifactID string, err error) error {
	switch {
	case errors.Is(err, ErrArtifactWorkbenchNotFound):
		return err
	case errors.Is(err, ErrArtifactWorkbenchConflict):
		return err
	case errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("%w: artifact %s not found", ErrArtifactWorkbenchNotFound, artifactID)
	case errors.Is(err, ports.ErrArtifactRunLinkageConflict):
		return fmt.Errorf("%w: %v", ErrArtifactWorkbenchConflict, err)
	default:
		return err
	}
}

func taskWorkbenchAction(actions []TaskWorkbenchAction, actionID string) TaskWorkbenchAction {
	for _, action := range actions {
		if action.ActionID == actionID {
			return action
		}
	}
	return TaskWorkbenchAction{}
}

func (s *Service) taskWorkbenchForAction(ctx context.Context, projectID, taskID string) (TaskWorkbenchView, error) {
	if err := ctxErr(ctx); err != nil {
		return TaskWorkbenchView{}, err
	}

	view, err := s.TaskWorkbench(ctx, projectID, taskID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return TaskWorkbenchView{}, fmt.Errorf("%w: %v", ErrTaskActionNotFound, err)
		}
		if strings.Contains(err.Error(), "does not belong to project") {
			return TaskWorkbenchView{}, fmt.Errorf("%w: task %s not found in project %s", ErrTaskActionNotFound, taskID, projectID)
		}
		return TaskWorkbenchView{}, err
	}
	return view, nil
}

func (s *Service) taskWorkbenchActionResponse(ctx context.Context, projectID, taskID, message string) (TaskWorkbenchActionResponse, error) {
	view, err := s.TaskWorkbench(ctx, projectID, taskID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return TaskWorkbenchActionResponse{}, fmt.Errorf("%w: %v", ErrTaskActionNotFound, err)
		}
		return TaskWorkbenchActionResponse{}, err
	}
	return TaskWorkbenchActionResponse{
		TaskID:              view.TaskID,
		TaskState:           view.TaskState,
		LatestRunID:         view.LatestRunID,
		LatestRunState:      view.LatestRunState,
		LatestApprovalID:    view.LatestApprovalID,
		LatestApprovalState: view.LatestApprovalState,
		RefreshRequired:     true,
		Message:             message,
	}, nil
}

func ctxErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}
