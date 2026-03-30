package query

import (
	"fmt"

	"github.com/sine-io/foreman/internal/ports"
)

type TaskWorkbenchArtifact struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Path    string `json:"path"`
	Summary string `json:"summary"`
}

type TaskWorkbenchAction struct {
	ActionID       string `json:"action_id"`
	Enabled        bool   `json:"enabled"`
	DisabledReason string `json:"disabled_reason,omitempty"`
	CurrentValue   any    `json:"current_value,omitempty"`
}

type TaskWorkbenchView struct {
	TaskID               string                  `json:"task_id"`
	ProjectID            string                  `json:"project_id"`
	ModuleID             string                  `json:"module_id"`
	Summary              string                  `json:"summary"`
	TaskState            string                  `json:"task_state"`
	Priority             int                     `json:"priority"`
	WriteScope           string                  `json:"write_scope"`
	TaskType             string                  `json:"task_type"`
	Acceptance           string                  `json:"acceptance"`
	LatestRunID          string                  `json:"latest_run_id,omitempty"`
	LatestRunState       string                  `json:"latest_run_state,omitempty"`
	LatestRunSummary     string                  `json:"latest_run_summary,omitempty"`
	LatestApprovalID     string                  `json:"latest_approval_id,omitempty"`
	LatestApprovalState  string                  `json:"latest_approval_state,omitempty"`
	LatestApprovalReason string                  `json:"latest_approval_reason,omitempty"`
	ApprovalWorkbenchURL string                  `json:"approval_workbench_url"`
	RunDetailURL         string                  `json:"run_detail_url,omitempty"`
	Artifacts            []TaskWorkbenchArtifact `json:"artifacts"`
	AvailableActions     []TaskWorkbenchAction   `json:"available_actions"`
	DisabledReasons      map[string]string       `json:"disabled_reasons"`
}

type TaskWorkbenchRepository interface {
	GetTaskWorkbench(taskID string) (ports.TaskWorkbenchRow, error)
}

type TaskWorkbenchQuery struct {
	Repo TaskWorkbenchRepository
}

func NewTaskWorkbenchQuery(repo TaskWorkbenchRepository) *TaskWorkbenchQuery {
	return &TaskWorkbenchQuery{Repo: repo}
}

func (q *TaskWorkbenchQuery) Execute(projectID, taskID string) (TaskWorkbenchView, error) {
	row, err := q.Repo.GetTaskWorkbench(taskID)
	if err != nil {
		return TaskWorkbenchView{}, err
	}
	if row.ProjectID != projectID {
		return TaskWorkbenchView{}, fmt.Errorf("task %s does not belong to project %s", taskID, projectID)
	}

	view := TaskWorkbenchView{
		TaskID:               row.TaskID,
		ProjectID:            row.ProjectID,
		ModuleID:             row.ModuleID,
		Summary:              row.Summary,
		TaskState:            row.TaskState,
		Priority:             row.Priority,
		WriteScope:           row.WriteScope,
		TaskType:             row.TaskType,
		Acceptance:           row.Acceptance,
		LatestRunID:          row.LatestRunID,
		LatestRunState:       row.LatestRunState,
		LatestRunSummary:     row.LatestRunSummary,
		LatestApprovalID:     row.LatestApprovalID,
		LatestApprovalState:  row.LatestApprovalState,
		LatestApprovalReason: row.LatestApprovalReason,
		ApprovalWorkbenchURL: approvalWorkbenchURL(row.ProjectID, row.LatestApprovalID),
		RunDetailURL:         runDetailURL(row.LatestRunID),
		Artifacts:            make([]TaskWorkbenchArtifact, 0, len(row.Artifacts)),
		DisabledReasons:      map[string]string{},
	}

	for _, artifact := range row.Artifacts {
		view.Artifacts = append(view.Artifacts, TaskWorkbenchArtifact{
			ID:      artifact.ID,
			Kind:    artifact.Kind,
			Path:    artifact.Path,
			Summary: artifact.Summary,
		})
	}

	view.AvailableActions = taskWorkbenchActions(row.TaskState, row.LatestRunID, row.LatestApprovalID, row.Priority)
	for _, action := range view.AvailableActions {
		if !action.Enabled && action.DisabledReason != "" {
			view.DisabledReasons[action.ActionID] = action.DisabledReason
		}
	}

	return view, nil
}

func approvalWorkbenchURL(projectID, approvalID string) string {
	url := "/board/approvals/workbench?project_id=" + projectID
	if approvalID != "" {
		url += "&approval_id=" + approvalID
	}
	return url
}

func runDetailURL(runID string) string {
	if runID == "" {
		return ""
	}
	return "/board/runs/" + runID
}

func taskWorkbenchActions(taskState, latestRunID, latestApprovalID string, priority int) []TaskWorkbenchAction {
	return []TaskWorkbenchAction{
		dispatchAction(taskState),
		cancelAction(taskState),
		reprioritizeAction(taskState, priority),
		retryAction(taskState),
		openLatestRunAction(latestRunID),
		openApprovalWorkbenchAction(latestApprovalID),
	}
}

func dispatchAction(taskState string) TaskWorkbenchAction {
	switch taskState {
	case "ready", "leased":
		return TaskWorkbenchAction{ActionID: "dispatch", Enabled: true}
	case "waiting_approval":
		return TaskWorkbenchAction{ActionID: "dispatch", DisabledReason: "Waiting approval"}
	case "approved_pending_dispatch":
		return TaskWorkbenchAction{ActionID: "dispatch", DisabledReason: "Use approval workbench retry-dispatch"}
	case "running":
		return TaskWorkbenchAction{ActionID: "dispatch", DisabledReason: "Already running"}
	case "failed":
		return TaskWorkbenchAction{ActionID: "dispatch", DisabledReason: "Use retry for failed tasks"}
	case "completed":
		return TaskWorkbenchAction{ActionID: "dispatch", DisabledReason: "Already completed"}
	case "canceled":
		return TaskWorkbenchAction{ActionID: "dispatch", DisabledReason: "Task canceled"}
	default:
		return TaskWorkbenchAction{ActionID: "dispatch", DisabledReason: "Task not dispatchable"}
	}
}

func cancelAction(taskState string) TaskWorkbenchAction {
	switch taskState {
	case "completed":
		return TaskWorkbenchAction{ActionID: "cancel", DisabledReason: "Already completed"}
	case "canceled":
		return TaskWorkbenchAction{ActionID: "cancel", DisabledReason: "Task canceled"}
	default:
		return TaskWorkbenchAction{ActionID: "cancel", Enabled: true}
	}
}

func reprioritizeAction(taskState string, priority int) TaskWorkbenchAction {
	switch taskState {
	case "completed":
		return TaskWorkbenchAction{ActionID: "reprioritize", DisabledReason: "Already completed", CurrentValue: priority}
	case "canceled":
		return TaskWorkbenchAction{ActionID: "reprioritize", DisabledReason: "Task canceled", CurrentValue: priority}
	default:
		return TaskWorkbenchAction{ActionID: "reprioritize", Enabled: true, CurrentValue: priority}
	}
}

func retryAction(taskState string) TaskWorkbenchAction {
	switch taskState {
	case "failed":
		return TaskWorkbenchAction{ActionID: "retry", Enabled: true}
	case "canceled":
		return TaskWorkbenchAction{ActionID: "retry", DisabledReason: "Task canceled"}
	default:
		return TaskWorkbenchAction{ActionID: "retry", DisabledReason: "Task not failed"}
	}
}

func openLatestRunAction(latestRunID string) TaskWorkbenchAction {
	if latestRunID == "" {
		return TaskWorkbenchAction{ActionID: "open_latest_run", DisabledReason: "No latest run"}
	}
	return TaskWorkbenchAction{ActionID: "open_latest_run", Enabled: true}
}

func openApprovalWorkbenchAction(latestApprovalID string) TaskWorkbenchAction {
	if latestApprovalID == "" {
		return TaskWorkbenchAction{ActionID: "open_approval_workbench", DisabledReason: "No approval history"}
	}
	return TaskWorkbenchAction{ActionID: "open_approval_workbench", Enabled: true}
}
