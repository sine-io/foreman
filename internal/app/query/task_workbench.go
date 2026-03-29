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
	ID             string `json:"id"`
	Enabled        bool   `json:"enabled"`
	DisabledReason string `json:"disabled_reason,omitempty"`
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

	view.AvailableActions = taskWorkbenchActions(row.TaskState, row.LatestRunID, row.LatestApprovalID)
	for _, action := range view.AvailableActions {
		if !action.Enabled && action.DisabledReason != "" {
			view.DisabledReasons[action.ID] = action.DisabledReason
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

func taskWorkbenchActions(taskState, latestRunID, latestApprovalID string) []TaskWorkbenchAction {
	return []TaskWorkbenchAction{
		dispatchAction(taskState),
		cancelAction(taskState),
		reprioritizeAction(taskState),
		retryAction(taskState),
		openLatestRunAction(latestRunID),
		openApprovalWorkbenchAction(latestApprovalID),
	}
}

func dispatchAction(taskState string) TaskWorkbenchAction {
	switch taskState {
	case "ready", "leased":
		return TaskWorkbenchAction{ID: "dispatch", Enabled: true}
	case "waiting_approval":
		return TaskWorkbenchAction{ID: "dispatch", DisabledReason: "Waiting approval"}
	case "approved_pending_dispatch":
		return TaskWorkbenchAction{ID: "dispatch", DisabledReason: "Use approval workbench retry-dispatch"}
	case "running":
		return TaskWorkbenchAction{ID: "dispatch", DisabledReason: "Already running"}
	case "failed":
		return TaskWorkbenchAction{ID: "dispatch", DisabledReason: "Use retry for failed tasks"}
	case "completed":
		return TaskWorkbenchAction{ID: "dispatch", DisabledReason: "Already completed"}
	case "canceled":
		return TaskWorkbenchAction{ID: "dispatch", DisabledReason: "Task canceled"}
	default:
		return TaskWorkbenchAction{ID: "dispatch", DisabledReason: "Task not dispatchable"}
	}
}

func cancelAction(taskState string) TaskWorkbenchAction {
	switch taskState {
	case "completed":
		return TaskWorkbenchAction{ID: "cancel", DisabledReason: "Already completed"}
	case "canceled":
		return TaskWorkbenchAction{ID: "cancel", DisabledReason: "Task canceled"}
	default:
		return TaskWorkbenchAction{ID: "cancel", Enabled: true}
	}
}

func reprioritizeAction(taskState string) TaskWorkbenchAction {
	switch taskState {
	case "completed":
		return TaskWorkbenchAction{ID: "reprioritize", DisabledReason: "Already completed"}
	case "canceled":
		return TaskWorkbenchAction{ID: "reprioritize", DisabledReason: "Task canceled"}
	default:
		return TaskWorkbenchAction{ID: "reprioritize", Enabled: true}
	}
}

func retryAction(taskState string) TaskWorkbenchAction {
	switch taskState {
	case "failed":
		return TaskWorkbenchAction{ID: "retry", Enabled: true}
	case "canceled":
		return TaskWorkbenchAction{ID: "retry", DisabledReason: "Task canceled"}
	default:
		return TaskWorkbenchAction{ID: "retry", DisabledReason: "Task not failed"}
	}
}

func openLatestRunAction(latestRunID string) TaskWorkbenchAction {
	if latestRunID == "" {
		return TaskWorkbenchAction{ID: "open_latest_run", DisabledReason: "No latest run"}
	}
	return TaskWorkbenchAction{ID: "open_latest_run", Enabled: true}
}

func openApprovalWorkbenchAction(latestApprovalID string) TaskWorkbenchAction {
	if latestApprovalID == "" {
		return TaskWorkbenchAction{ID: "open_approval_workbench", DisabledReason: "No approval history"}
	}
	return TaskWorkbenchAction{ID: "open_approval_workbench", Enabled: true}
}
