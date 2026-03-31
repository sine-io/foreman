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

func TestBoardQueryRepositoryPortExposesRunWorkbenchLookup(t *testing.T) {
	var repo ports.BoardQueryRepository = fakeBoardReadRepo{}

	_, err := repo.GetRunWorkbench("run-1")
	require.NoError(t, err)
}

type fakeBoardReadRepo struct {
	modules      []ports.ModuleBoardRow
	tasks        []ports.TaskBoardRow
	runDetail    ports.RunDetailRecord
	runWorkbench ports.RunWorkbenchRow
	approvals    []ports.ApprovalQueueRow
	workbench    ports.TaskWorkbenchRow
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

func (f fakeBoardReadRepo) GetRunWorkbench(runID string) (ports.RunWorkbenchRow, error) {
	return f.runWorkbench, nil
}

func (f fakeBoardReadRepo) GetArtifactWorkbench(artifactID string) (ports.ArtifactWorkbenchRow, error) {
	return ports.ArtifactWorkbenchRow{}, nil
}

func (f fakeBoardReadRepo) ListApprovals(projectID string) ([]ports.ApprovalQueueRow, error) {
	return f.approvals, nil
}

func (f fakeBoardReadRepo) ListApprovalWorkbenchQueue(projectID string) ([]ports.ApprovalWorkbenchQueueRow, error) {
	return nil, nil
}

func (f fakeBoardReadRepo) GetApprovalWorkbenchDetail(approvalID string) (ports.ApprovalWorkbenchDetailRow, error) {
	return ports.ApprovalWorkbenchDetailRow{}, nil
}

func (f fakeBoardReadRepo) GetTaskWorkbench(taskID string) (ports.TaskWorkbenchRow, error) {
	return f.workbench, nil
}
