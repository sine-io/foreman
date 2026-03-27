package query

import (
	"testing"

	"github.com/sine-io/foreman/internal/ports"
	"github.com/stretchr/testify/require"
)

func TestModuleBoardGroupsModulesByBoardState(t *testing.T) {
	query := NewModuleBoardQuery(fakeBoardReadRepo{
		modules: []ports.ModuleBoardRow{
			{ModuleID: "module-1", Name: "Bootstrap", BoardState: "active"},
		},
	})

	view, err := query.Execute("project-1")
	require.NoError(t, err)
	require.Contains(t, view.Columns, "Implementing")
	require.Len(t, view.Columns["Implementing"], 1)
}

func TestTaskBoardShowsPendingApprovals(t *testing.T) {
	query := NewTaskBoardQuery(fakeBoardReadRepo{
		tasks: []ports.TaskBoardRow{
			{TaskID: "task-1", Summary: "Review push", State: "running", PendingApproval: true},
		},
	})

	view, err := query.Execute("project-1")
	require.NoError(t, err)
	require.NotEmpty(t, view.Columns["Waiting Approval"])
}

func TestRunDetailIncludesArtifactSummaries(t *testing.T) {
	query := NewRunDetailQuery(fakeBoardReadRepo{
		runDetail: ports.RunDetailRecord{
			Run: ports.Run{
				ID:         "run-1",
				TaskID:     "task-1",
				RunnerKind: "codex",
				State:      "completed",
			},
			TaskSummary: "Persist board state",
			Artifacts: []ports.ArtifactRecord{
				{ID: "artifact-1", Kind: "assistant_summary", Summary: "Stored in sqlite"},
			},
		},
	})

	view, err := query.Execute("run-1")
	require.NoError(t, err)
	require.NotEmpty(t, view.Artifacts)
}

type fakeBoardReadRepo struct {
	modules   []ports.ModuleBoardRow
	tasks     []ports.TaskBoardRow
	runDetail ports.RunDetailRecord
	approvals []ports.ApprovalQueueRow
}

func (f fakeBoardReadRepo) ListModules(projectID string) ([]ports.ModuleBoardRow, error) {
	return f.modules, nil
}

func (f fakeBoardReadRepo) ListTasks(projectID string) ([]ports.TaskBoardRow, error) {
	return f.tasks, nil
}

func (f fakeBoardReadRepo) GetRunDetail(runID string) (ports.RunDetailRecord, error) {
	return f.runDetail, nil
}

func (f fakeBoardReadRepo) ListApprovals(projectID string) ([]ports.ApprovalQueueRow, error) {
	return f.approvals, nil
}
