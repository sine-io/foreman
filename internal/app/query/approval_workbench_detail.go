package query

import "github.com/sine-io/foreman/internal/ports"

type ApprovalWorkbenchArtifact struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Path    string `json:"path"`
	Summary string `json:"summary"`
}

type ApprovalWorkbenchDetailView struct {
	ApprovalID              string                      `json:"approval_id"`
	TaskID                  string                      `json:"task_id"`
	Summary                 string                      `json:"summary"`
	Reason                  string                      `json:"reason"`
	ApprovalState           string                      `json:"approval_state"`
	RiskLevel               string                      `json:"risk_level"`
	PolicyRule              string                      `json:"policy_rule"`
	RejectionReason         string                      `json:"rejection_reason,omitempty"`
	Priority                int                         `json:"priority"`
	CreatedAt               string                      `json:"created_at"`
	TaskState               string                      `json:"task_state"`
	RunID                   string                      `json:"run_id,omitempty"`
	RunState                string                      `json:"run_state,omitempty"`
	RunDetailURL            string                      `json:"run_detail_url,omitempty"`
	AssistantSummaryPreview string                      `json:"assistant_summary_preview"`
	Artifacts               []ApprovalWorkbenchArtifact `json:"artifacts"`
}

type ApprovalWorkbenchDetailQuery struct {
	Repo ports.BoardQueryRepository
}

func NewApprovalWorkbenchDetailQuery(repo ports.BoardQueryRepository) *ApprovalWorkbenchDetailQuery {
	return &ApprovalWorkbenchDetailQuery{Repo: repo}
}

func (q *ApprovalWorkbenchDetailQuery) Execute(approvalID string) (ApprovalWorkbenchDetailView, error) {
	row, err := q.Repo.GetApprovalWorkbenchDetail(approvalID)
	if err != nil {
		return ApprovalWorkbenchDetailView{}, err
	}

	view := ApprovalWorkbenchDetailView{
		ApprovalID:              row.ApprovalID,
		TaskID:                  row.TaskID,
		Summary:                 row.Summary,
		Reason:                  row.Reason,
		ApprovalState:           row.ApprovalState,
		RiskLevel:               row.RiskLevel,
		PolicyRule:              row.PolicyRule,
		RejectionReason:         row.RejectionReason,
		Priority:                row.Priority,
		CreatedAt:               row.CreatedAt,
		TaskState:               row.TaskState,
		RunID:                   row.RunID,
		RunState:                row.RunState,
		AssistantSummaryPreview: row.AssistantSummary,
		Artifacts:               make([]ApprovalWorkbenchArtifact, 0, len(row.Artifacts)),
	}
	if row.RunID != "" {
		view.RunDetailURL = "/board/runs/" + row.RunID
	}

	for _, artifact := range row.Artifacts {
		view.Artifacts = append(view.Artifacts, ApprovalWorkbenchArtifact{
			ID:      artifact.ID,
			Kind:    artifact.Kind,
			Path:    artifact.Path,
			Summary: artifact.Summary,
		})
		if view.AssistantSummaryPreview == "" && artifact.Summary != "" {
			view.AssistantSummaryPreview = artifact.Summary
		}
	}

	return view, nil
}
