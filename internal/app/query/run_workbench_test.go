package query

import (
	"database/sql"
	"testing"

	"github.com/sine-io/foreman/internal/ports"
	"github.com/stretchr/testify/require"
)

func TestRunWorkbenchUsesAssistantSummaryWhenPresent(t *testing.T) {
	query := NewRunWorkbenchQuery(fakeRunWorkbenchRepo{
		row: ports.RunWorkbenchRow{
			RunID:       "run-1",
			TaskID:      "task-1",
			ProjectID:   "project-1",
			ModuleID:    "module-1",
			TaskSummary: "Ship the run workbench",
			RunState:    "failed",
			RunnerKind:  "codex",
			Artifacts: []ports.ArtifactRecord{
				{ID: "artifact-2", Kind: "assistant_summary", Summary: "Assistant summary explains the failure"},
				{ID: "artifact-1", Kind: "run_log", Summary: "log lines should not win"},
			},
		},
	})

	view, err := query.Execute("run-1")
	require.NoError(t, err)
	require.Equal(t, "Assistant summary explains the failure", view.PrimarySummary)
}

func TestRunWorkbenchFallsBackToRunStateAndArtifactSummaries(t *testing.T) {
	query := NewRunWorkbenchQuery(fakeRunWorkbenchRepo{
		row: ports.RunWorkbenchRow{
			RunID:    "run-1",
			TaskID:   "task-1",
			RunState: "failed",
			Artifacts: []ports.ArtifactRecord{
				{ID: "artifact-3", Kind: "run_log", Summary: "raw log content should be ignored"},
				{ID: "artifact-2", Kind: "command_result", Summary: "Unit tests failed in internal/app/query"},
				{ID: "artifact-1", Kind: "diff_summary", Summary: "Updated the run workbench query files"},
			},
		},
	})

	view, err := query.Execute("run-1")
	require.NoError(t, err)
	require.Equal(t, "failed — command_result: Unit tests failed in internal/app/query — diff_summary: Updated the run workbench query files", view.PrimarySummary)
}

func TestRunWorkbenchHandlesNoArtifacts(t *testing.T) {
	query := NewRunWorkbenchQuery(fakeRunWorkbenchRepo{
		row: ports.RunWorkbenchRow{
			RunID:    "run-1",
			TaskID:   "task-1",
			RunState: "completed",
		},
	})

	view, err := query.Execute("run-1")
	require.NoError(t, err)
	require.Equal(t, "completed", view.PrimarySummary)
	require.Empty(t, view.Artifacts)
	require.Empty(t, view.ArtifactTargetURLs)
}

func TestRunWorkbenchReturnsNotFoundForMissingRun(t *testing.T) {
	query := NewRunWorkbenchQuery(fakeRunWorkbenchRepo{err: sql.ErrNoRows})

	_, err := query.Execute("run-missing")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestRunWorkbenchArtifactTargetsUseArtifactWorkbenchForLinkedArtifacts(t *testing.T) {
	query := NewRunWorkbenchQuery(fakeRunWorkbenchRepo{
		row: ports.RunWorkbenchRow{
			RunID:     "run-1",
			TaskID:    "task-1",
			ProjectID: "project-1",
			RunState:  "running",
			Artifacts: []ports.ArtifactRecord{
				{ID: "artifact-2", RunID: "run-1", Kind: "assistant_summary", Path: "artifacts/tasks/task-1/assistant_summary.txt", Summary: "Working through the task"},
				{ID: "artifact-1", RunID: "run-1", Kind: "command_result", Path: "artifacts/tasks/task-1/command.txt", Summary: "Command output summary"},
			},
		},
	})

	view, err := query.Execute("run-1")
	require.NoError(t, err)
	require.Equal(t, "/board/tasks/workbench?project_id=project-1&task_id=task-1", view.TaskWorkbenchURL)
	require.Equal(t, map[string]string{
		"artifact-2": "/board/artifacts/workbench?artifact_id=artifact-2",
		"artifact-1": "/board/artifacts/workbench?artifact_id=artifact-1",
	}, view.ArtifactTargetURLs)
	require.Len(t, view.Artifacts, 2)
	require.Equal(t, "assistant_summary", view.Artifacts[0].Kind)
	require.Equal(t, "artifacts/tasks/task-1/assistant_summary.txt", view.Artifacts[0].Path)
}

func TestRunWorkbenchArtifactTargetsKeepAnchorFallbackForLegacyArtifacts(t *testing.T) {
	query := NewRunWorkbenchQuery(fakeRunWorkbenchRepo{
		row: ports.RunWorkbenchRow{
			RunID:     "run-1",
			TaskID:    "task-1",
			ProjectID: "project-1",
			RunState:  "running",
			Artifacts: []ports.ArtifactRecord{
				{ID: "artifact-2", Kind: "assistant_summary", Path: "artifacts/tasks/task-1/assistant_summary.txt", Summary: "Working through the task"},
				{ID: "artifact-1", RunID: "run-1", Kind: "command_result", Path: "artifacts/tasks/task-1/command.txt", Summary: "Command output summary"},
			},
		},
	})

	view, err := query.Execute("run-1")
	require.NoError(t, err)
	require.Equal(t, map[string]string{
		"artifact-2": "#artifact-artifact-2",
		"artifact-1": "/board/artifacts/workbench?artifact_id=artifact-1",
	}, view.ArtifactTargetURLs)
}

func TestRunWorkbenchEscapesTaskWorkbenchURLQueryParameters(t *testing.T) {
	query := NewRunWorkbenchQuery(fakeRunWorkbenchRepo{
		row: ports.RunWorkbenchRow{
			RunID:     "run-1",
			TaskID:    "task&1#frag",
			ProjectID: "project=1?x/y",
			RunState:  "running",
		},
	})

	view, err := query.Execute("run-1")
	require.NoError(t, err)
	require.Equal(t, "/board/tasks/workbench?project_id=project%3D1%3Fx%2Fy&task_id=task%261%23frag", view.TaskWorkbenchURL)
}

func TestRunWorkbenchIncludesSupplementalMetadata(t *testing.T) {
	query := NewRunWorkbenchQuery(fakeRunWorkbenchRepo{
		row: ports.RunWorkbenchRow{
			RunID:        "run-42",
			TaskID:       "task-9",
			ProjectID:    "project-7",
			ModuleID:     "module-3",
			TaskSummary:  "Investigate a flaky workbench test",
			RunState:     "failed",
			RunnerKind:   "codex",
			RunCreatedAt: "2026-03-30T09:15:00.000000000Z",
		},
	})

	view, err := query.Execute("run-42")
	require.NoError(t, err)
	require.Equal(t, "run-42", view.RunID)
	require.Equal(t, "task-9", view.TaskID)
	require.Equal(t, "project-7", view.ProjectID)
	require.Equal(t, "module-3", view.ModuleID)
	require.Equal(t, "Investigate a flaky workbench test", view.TaskSummary)
	require.Equal(t, "failed", view.RunState)
	require.Equal(t, "codex", view.RunnerKind)
	require.Equal(t, "2026-03-30T09:15:00.000000000Z", view.RunCreatedAt)
}

type fakeRunWorkbenchRepo struct {
	row ports.RunWorkbenchRow
	err error
}

func (f fakeRunWorkbenchRepo) GetRunWorkbench(runID string) (ports.RunWorkbenchRow, error) {
	if f.err != nil {
		return ports.RunWorkbenchRow{}, f.err
	}
	return f.row, nil
}
