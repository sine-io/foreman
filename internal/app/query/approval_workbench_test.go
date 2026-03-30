package query

import (
	"testing"

	"github.com/sine-io/foreman/internal/ports"
	"github.com/stretchr/testify/require"
)

func TestApprovalWorkbenchQueueOrdersByRiskPriorityAndCreatedAt(t *testing.T) {
	query := NewApprovalWorkbenchQueueQuery(fakeApprovalWorkbenchRepo{
		queue: []ports.ApprovalWorkbenchQueueRow{
			{ApprovalID: "approval-low", TaskID: "task-low", Summary: "Low", RiskLevel: "low", Priority: 99, CreatedAt: "2026-03-28T10:00:00.000000000Z"},
			{ApprovalID: "approval-high-old", TaskID: "task-high-old", Summary: "High old", RiskLevel: "high", Priority: 5, CreatedAt: "2026-03-28T08:00:00.000000000Z"},
			{ApprovalID: "approval-critical", TaskID: "task-critical", Summary: "Critical", RiskLevel: "critical", Priority: 1, CreatedAt: "2026-03-28T12:00:00.000000000Z"},
			{ApprovalID: "approval-medium", TaskID: "task-medium", Summary: "Medium", RiskLevel: "medium", Priority: 10, CreatedAt: "2026-03-28T07:00:00.000000000Z"},
			{ApprovalID: "approval-high-new", TaskID: "task-high-new", Summary: "High new", RiskLevel: "high", Priority: 5, CreatedAt: "2026-03-28T09:00:00.000000000Z"},
			{ApprovalID: "approval-high-low-priority", TaskID: "task-high-low-priority", Summary: "High low priority", RiskLevel: "high", Priority: 2, CreatedAt: "2026-03-28T06:00:00.000000000Z"},
		},
	})

	view, err := query.Execute("project-1")
	require.NoError(t, err)
	require.Len(t, view.Items, 6)
	require.Equal(t, []string{
		"approval-critical",
		"approval-high-old",
		"approval-high-new",
		"approval-high-low-priority",
		"approval-medium",
		"approval-low",
	}, approvalIDs(view.Items))
}

func TestApprovalWorkbenchDetailIncludesRiskRunArtifactsAndPreviewFallback(t *testing.T) {
	query := NewApprovalWorkbenchDetailQuery(fakeApprovalWorkbenchRepo{
		details: map[string]ports.ApprovalWorkbenchDetailRow{
			"approval-1": {
				ApprovalID:    "approval-1",
				TaskID:        "task-1",
				Summary:       "Push release branch",
				Reason:        "git push requires approval",
				ApprovalState: "pending",
				RiskLevel:     "high",
				PolicyRule:    "strict.git_push",
				Priority:      7,
				CreatedAt:     "2026-03-28T08:00:00.000000000Z",
				TaskState:     "waiting_approval",
				RunID:         "run-9",
				RunState:      "running",
				Artifacts: []ports.ArtifactRecord{
					{ID: "artifact-1", TaskID: "task-1", Kind: "assistant_summary", Path: "artifacts/tasks/task-1/assistant_summary.txt", Summary: "Pushed branch and waiting for checks"},
					{ID: "artifact-2", TaskID: "task-1", Kind: "run_log", Path: "artifacts/tasks/task-1/run.log", Summary: "runner output"},
				},
			},
		},
	})

	view, err := query.Execute("approval-1")
	require.NoError(t, err)
	require.Equal(t, "approval-1", view.ApprovalID)
	require.Equal(t, "high", view.RiskLevel)
	require.Equal(t, "strict.git_push", view.PolicyRule)
	require.Equal(t, "run-9", view.RunID)
	require.Equal(t, "running", view.RunState)
	require.Equal(t, "/board/runs/run-9", view.RunDetailURL)
	require.Equal(t, "Pushed branch and waiting for checks", view.AssistantSummaryPreview)
	require.Len(t, view.Artifacts, 2)
	require.Equal(t, "assistant_summary", view.Artifacts[0].Kind)
}

func TestApprovalWorkbenchDetailReturnsHistoricalApprovedAndRejectedViewsByApprovalID(t *testing.T) {
	query := NewApprovalWorkbenchDetailQuery(fakeApprovalWorkbenchRepo{
		details: map[string]ports.ApprovalWorkbenchDetailRow{
			"approval-approved": {
				ApprovalID:    "approval-approved",
				TaskID:        "task-1",
				Summary:       "Deploy release",
				ApprovalState: "approved",
				RiskLevel:     "critical",
				TaskState:     "completed",
				RunID:         "run-1",
				RunState:      "completed",
			},
			"approval-rejected": {
				ApprovalID:      "approval-rejected",
				TaskID:          "task-2",
				Summary:         "Delete cache",
				ApprovalState:   "rejected",
				RiskLevel:       "medium",
				TaskState:       "ready",
				RejectionReason: "missing rollback plan",
			},
		},
	})

	approvedView, err := query.Execute("approval-approved")
	require.NoError(t, err)
	require.Equal(t, "approved", approvedView.ApprovalState)
	require.Equal(t, "completed", approvedView.RunState)

	rejectedView, err := query.Execute("approval-rejected")
	require.NoError(t, err)
	require.Equal(t, "rejected", rejectedView.ApprovalState)
	require.Equal(t, "missing rollback plan", rejectedView.RejectionReason)
	require.Equal(t, "ready", rejectedView.TaskState)
}

type fakeApprovalWorkbenchRepo struct {
	queue   []ports.ApprovalWorkbenchQueueRow
	details map[string]ports.ApprovalWorkbenchDetailRow
}

func (f fakeApprovalWorkbenchRepo) ListModules(projectID string) ([]ports.ModuleBoardRow, error) {
	return nil, nil
}

func (f fakeApprovalWorkbenchRepo) ListTasks(projectID string) ([]ports.TaskBoardRow, error) {
	return nil, nil
}

func (f fakeApprovalWorkbenchRepo) GetRunDetail(runID string) (ports.RunDetailRecord, error) {
	return ports.RunDetailRecord{}, nil
}

func (f fakeApprovalWorkbenchRepo) GetRunWorkbench(runID string) (ports.RunWorkbenchRow, error) {
	return ports.RunWorkbenchRow{}, nil
}

func (f fakeApprovalWorkbenchRepo) ListApprovals(projectID string) ([]ports.ApprovalQueueRow, error) {
	return nil, nil
}

func (f fakeApprovalWorkbenchRepo) ListApprovalWorkbenchQueue(projectID string) ([]ports.ApprovalWorkbenchQueueRow, error) {
	return f.queue, nil
}

func (f fakeApprovalWorkbenchRepo) GetApprovalWorkbenchDetail(approvalID string) (ports.ApprovalWorkbenchDetailRow, error) {
	return f.details[approvalID], nil
}

func (f fakeApprovalWorkbenchRepo) GetTaskWorkbench(taskID string) (ports.TaskWorkbenchRow, error) {
	return ports.TaskWorkbenchRow{}, nil
}

func approvalIDs(items []ApprovalWorkbenchItem) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ApprovalID)
	}
	return ids
}
