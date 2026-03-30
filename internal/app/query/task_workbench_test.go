package query

import (
	"database/sql"
	"testing"

	"github.com/sine-io/foreman/internal/ports"
	"github.com/stretchr/testify/require"
)

func TestTaskWorkbenchIncludesLatestRunApprovalArtifactsAndMetadata(t *testing.T) {
	query := NewTaskWorkbenchQuery(fakeTaskWorkbenchRepo{
		row: ports.TaskWorkbenchRow{
			TaskID:               "task-1",
			ProjectID:            "project-1",
			ModuleID:             "module-1",
			Summary:              "Ship task workbench query",
			TaskState:            "failed",
			Priority:             7,
			WriteScope:           "repo:project-1",
			TaskType:             "write",
			Acceptance:           "Query returns latest task workbench state",
			LatestRunID:          "run-2",
			LatestRunState:       "failed",
			LatestRunSummary:     "Latest run failed after syncing artifacts",
			LatestApprovalID:     "approval-2",
			LatestApprovalState:  "approved",
			LatestApprovalReason: "git push origin main requires approval",
			Artifacts: []ports.ArtifactRecord{
				{ID: "artifact-3", TaskID: "task-1", Kind: "assistant_summary", Path: "artifacts/tasks/task-1/assistant_summary.txt", Summary: "Latest run failed after syncing artifacts"},
				{ID: "artifact-2", TaskID: "task-1", Kind: "run_log", Path: "artifacts/tasks/task-1/run.log", Summary: "runner output"},
			},
		},
	})

	view, err := query.Execute("project-1", "task-1")
	require.NoError(t, err)
	require.Equal(t, "task-1", view.TaskID)
	require.Equal(t, "project-1", view.ProjectID)
	require.Equal(t, "module-1", view.ModuleID)
	require.Equal(t, "Ship task workbench query", view.Summary)
	require.Equal(t, "failed", view.TaskState)
	require.Equal(t, 7, view.Priority)
	require.Equal(t, "repo:project-1", view.WriteScope)
	require.Equal(t, "write", view.TaskType)
	require.Equal(t, "Query returns latest task workbench state", view.Acceptance)
	require.Equal(t, "run-2", view.LatestRunID)
	require.Equal(t, "failed", view.LatestRunState)
	require.Equal(t, "Latest run failed after syncing artifacts", view.LatestRunSummary)
	require.Equal(t, "approval-2", view.LatestApprovalID)
	require.Equal(t, "approved", view.LatestApprovalState)
	require.Equal(t, "git push origin main requires approval", view.LatestApprovalReason)
	require.Equal(t, "/board/approvals/workbench?project_id=project-1&approval_id=approval-2", view.ApprovalWorkbenchURL)
	require.Equal(t, "/board/runs/run-2", view.RunDetailURL)
	require.Len(t, view.Artifacts, 2)
	require.Equal(t, "assistant_summary", view.Artifacts[0].Kind)
}

func TestTaskWorkbenchComputesDisabledReasonsPerAction(t *testing.T) {
	query := NewTaskWorkbenchQuery(fakeTaskWorkbenchRepo{
		row: ports.TaskWorkbenchRow{
			TaskID:      "task-1",
			ProjectID:   "project-1",
			ModuleID:    "module-1",
			Summary:     "Recover failed task",
			TaskState:   "approved_pending_dispatch",
			Priority:    5,
			WriteScope:  "repo:project-1",
			TaskType:    "write",
			Acceptance:  "Recovery path exposed",
			LatestRunID: "run-3",
		},
	})

	view, err := query.Execute("project-1", "task-1")
	require.NoError(t, err)
	require.False(t, taskWorkbenchAction(view.AvailableActions, "dispatch").Enabled)
	require.Equal(t, "Use approval workbench retry-dispatch", taskWorkbenchAction(view.AvailableActions, "dispatch").DisabledReason)
	require.False(t, taskWorkbenchAction(view.AvailableActions, "retry").Enabled)
	require.Equal(t, "Task not failed", taskWorkbenchAction(view.AvailableActions, "retry").DisabledReason)
	require.True(t, taskWorkbenchAction(view.AvailableActions, "cancel").Enabled)
	require.True(t, taskWorkbenchAction(view.AvailableActions, "reprioritize").Enabled)
	require.True(t, taskWorkbenchAction(view.AvailableActions, "open_latest_run").Enabled)
	require.Equal(t, "Use approval workbench retry-dispatch", view.DisabledReasons["dispatch"])
	require.Equal(t, "Task not failed", view.DisabledReasons["retry"])
}

func TestTaskWorkbenchActionMatrixMatchesTaskState(t *testing.T) {
	testCases := []struct {
		name               string
		taskState          string
		expectDispatch     bool
		expectCancel       bool
		expectReprioritize bool
		expectRetry        bool
		dispatchReason     string
		cancelReason       string
		reprioritizeReason string
		retryReason        string
	}{
		{name: "ready", taskState: "ready", expectDispatch: true, expectCancel: true, expectReprioritize: true, expectRetry: false, retryReason: "Task not failed"},
		{name: "leased", taskState: "leased", expectDispatch: true, expectCancel: true, expectReprioritize: true, expectRetry: false, retryReason: "Task not failed"},
		{name: "waiting approval", taskState: "waiting_approval", expectDispatch: false, expectCancel: true, expectReprioritize: true, expectRetry: false, dispatchReason: "Waiting approval", retryReason: "Task not failed"},
		{name: "approved pending dispatch", taskState: "approved_pending_dispatch", expectDispatch: false, expectCancel: true, expectReprioritize: true, expectRetry: false, dispatchReason: "Use approval workbench retry-dispatch", retryReason: "Task not failed"},
		{name: "running", taskState: "running", expectDispatch: false, expectCancel: true, expectReprioritize: true, expectRetry: false, dispatchReason: "Already running", retryReason: "Task not failed"},
		{name: "failed", taskState: "failed", expectDispatch: false, expectCancel: true, expectReprioritize: true, expectRetry: true, dispatchReason: "Use retry for failed tasks"},
		{name: "completed", taskState: "completed", expectDispatch: false, expectCancel: false, expectReprioritize: false, expectRetry: false, dispatchReason: "Already completed", cancelReason: "Already completed", reprioritizeReason: "Already completed", retryReason: "Task not failed"},
		{name: "canceled", taskState: "canceled", expectDispatch: false, expectCancel: false, expectReprioritize: false, expectRetry: false, dispatchReason: "Task canceled", cancelReason: "Task canceled", reprioritizeReason: "Task canceled", retryReason: "Task canceled"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			query := NewTaskWorkbenchQuery(fakeTaskWorkbenchRepo{
				row: ports.TaskWorkbenchRow{
					TaskID:    "task-1",
					ProjectID: "project-1",
					ModuleID:  "module-1",
					TaskState: tc.taskState,
				},
			})

			view, err := query.Execute("project-1", "task-1")
			require.NoError(t, err)
			require.Equal(t, tc.expectDispatch, taskWorkbenchAction(view.AvailableActions, "dispatch").Enabled)
			require.Equal(t, tc.dispatchReason, taskWorkbenchAction(view.AvailableActions, "dispatch").DisabledReason)
			require.Equal(t, tc.expectCancel, taskWorkbenchAction(view.AvailableActions, "cancel").Enabled)
			require.Equal(t, tc.cancelReason, taskWorkbenchAction(view.AvailableActions, "cancel").DisabledReason)
			require.Equal(t, tc.expectReprioritize, taskWorkbenchAction(view.AvailableActions, "reprioritize").Enabled)
			require.Equal(t, tc.reprioritizeReason, taskWorkbenchAction(view.AvailableActions, "reprioritize").DisabledReason)
			require.Equal(t, tc.expectRetry, taskWorkbenchAction(view.AvailableActions, "retry").Enabled)
			require.Equal(t, tc.retryReason, taskWorkbenchAction(view.AvailableActions, "retry").DisabledReason)
		})
	}
}

func TestTaskWorkbenchDoesNotDeriveLatestRunSummaryFromArtifacts(t *testing.T) {
	query := NewTaskWorkbenchQuery(fakeTaskWorkbenchRepo{
		row: ports.TaskWorkbenchRow{
			TaskID:         "task-1",
			ProjectID:      "project-1",
			ModuleID:       "module-1",
			TaskState:      "failed",
			LatestRunID:    "run-2",
			LatestRunState: "failed",
			Artifacts: []ports.ArtifactRecord{
				{
					ID:      "artifact-1",
					TaskID:  "task-1",
					Kind:    "assistant_summary",
					Path:    "artifacts/tasks/task-1/assistant_summary.txt",
					Summary: "artifact fallback should not populate latest run summary",
				},
			},
		},
	})

	view, err := query.Execute("project-1", "task-1")
	require.NoError(t, err)
	require.Empty(t, view.LatestRunSummary)
	require.Len(t, view.Artifacts, 1)
}

func TestTaskWorkbenchSupportsNoRunAndNoApprovalCases(t *testing.T) {
	query := NewTaskWorkbenchQuery(fakeTaskWorkbenchRepo{
		row: ports.TaskWorkbenchRow{
			TaskID:     "task-1",
			ProjectID:  "project-1",
			ModuleID:   "module-1",
			Summary:    "Fresh task",
			TaskState:  "ready",
			Priority:   1,
			WriteScope: "repo:project-1",
			TaskType:   "read",
			Acceptance: "No approval or run yet",
			Artifacts:  nil,
		},
	})

	view, err := query.Execute("project-1", "task-1")
	require.NoError(t, err)
	require.Empty(t, view.LatestRunID)
	require.Empty(t, view.LatestApprovalID)
	require.Equal(t, "/board/approvals/workbench?project_id=project-1", view.ApprovalWorkbenchURL)
	require.Empty(t, view.RunDetailURL)
	require.False(t, taskWorkbenchAction(view.AvailableActions, "open_latest_run").Enabled)
	require.Equal(t, "No latest run", taskWorkbenchAction(view.AvailableActions, "open_latest_run").DisabledReason)
	require.False(t, taskWorkbenchAction(view.AvailableActions, "open_approval_workbench").Enabled)
	require.Equal(t, "No approval history", taskWorkbenchAction(view.AvailableActions, "open_approval_workbench").DisabledReason)
}

func TestTaskWorkbenchRejectsCrossProjectTasks(t *testing.T) {
	query := NewTaskWorkbenchQuery(fakeTaskWorkbenchRepo{
		row: ports.TaskWorkbenchRow{
			TaskID:    "task-1",
			ProjectID: "project-2",
			ModuleID:  "module-1",
		},
	})

	_, err := query.Execute("project-1", "task-1")
	require.EqualError(t, err, "task task-1 does not belong to project project-1")
}

type fakeTaskWorkbenchRepo struct {
	row ports.TaskWorkbenchRow
	err error
}

func (f fakeTaskWorkbenchRepo) GetTaskWorkbench(taskID string) (ports.TaskWorkbenchRow, error) {
	if f.err != nil {
		return ports.TaskWorkbenchRow{}, f.err
	}
	if f.row.TaskID != taskID {
		return ports.TaskWorkbenchRow{}, sql.ErrNoRows
	}

	return f.row, nil
}

func taskWorkbenchAction(actions []TaskWorkbenchAction, id string) TaskWorkbenchAction {
	for _, action := range actions {
		if action.ActionID == id {
			return action
		}
	}
	return TaskWorkbenchAction{}
}
