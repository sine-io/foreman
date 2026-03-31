package query

import (
	"strings"
	"testing"

	"github.com/sine-io/foreman/internal/infrastructure/store/artifactfs"
	sqlitestore "github.com/sine-io/foreman/internal/infrastructure/store/sqlite"
	"github.com/sine-io/foreman/internal/ports"
	"github.com/stretchr/testify/require"
)

func TestArtifactCompareReturnsReadyForPreviousSameTaskSameKindArtifact(t *testing.T) {
	harness := newArtifactWorkbenchHarness(t)
	harness.seedTask(t, "project-1", "module-1", "task-1")
	harness.saveRun(t, ports.Run{
		ID:         "run-1",
		TaskID:     "task-1",
		RunnerKind: "codex",
		State:      "failed",
	}, "2026-03-31T09:00:00.000000000Z")
	harness.saveRun(t, ports.Run{
		ID:         "run-2",
		TaskID:     "task-1",
		RunnerKind: "codex",
		State:      "completed",
	}, "2026-03-31T10:00:00.000000000Z")
	harness.writeArtifactFile(t, "tasks/task-1/assistant-prev.txt", "line one\nold line\n")
	harness.writeArtifactFile(t, "tasks/task-1/assistant-current.txt", "line one\nnew line\n")
	harness.saveArtifact(t, artifactSeed{
		ID:          "artifact-prev",
		TaskID:      "task-1",
		RunID:       "run-1",
		Kind:        "assistant_summary",
		Path:        "tasks/task-1/assistant-prev.txt",
		StoragePath: "tasks/task-1/assistant-prev.txt",
		Summary:     "Older assistant summary",
		CreatedAt:   "2026-03-31T09:01:00.000000000Z",
	})
	harness.saveArtifact(t, artifactSeed{
		ID:          "artifact-current",
		TaskID:      "task-1",
		RunID:       "run-2",
		Kind:        "assistant_summary",
		Path:        "tasks/task-1/assistant-current.txt",
		StoragePath: "tasks/task-1/assistant-current.txt",
		Summary:     "Current assistant summary",
		CreatedAt:   "2026-03-31T10:01:00.000000000Z",
	})

	view, err := newArtifactCompareHarness(harness).Execute("artifact-current")
	require.NoError(t, err)
	require.Equal(t, "artifact-current", view.Current.ArtifactID)
	require.NotNil(t, view.Previous)
	require.Equal(t, "artifact-prev", view.Previous.ArtifactID)
	require.Equal(t, "ready", view.Status)
	require.Equal(t, "text/plain; charset=utf-8", view.Current.ContentType)
	require.Equal(t, "text/plain; charset=utf-8", view.Previous.ContentType)
	require.NotNil(t, view.Diff)
	require.Equal(t, "text/unified-diff", view.Diff.Format)
	require.Contains(t, view.Diff.Content, "-old line")
	require.Contains(t, view.Diff.Content, "+new line")
	require.NotEmpty(t, view.Messages.Title)
	require.NotEmpty(t, view.Messages.Detail)
}

func TestArtifactCompareUsesCreatedAtAndArtifactIDAsTieBreaker(t *testing.T) {
	harness := newArtifactWorkbenchHarness(t)
	harness.seedTask(t, "project-1", "module-1", "task-1")
	harness.saveRun(t, ports.Run{
		ID:         "run-1",
		TaskID:     "task-1",
		RunnerKind: "codex",
		State:      "failed",
	}, "2026-03-31T09:00:00.000000000Z")
	harness.saveRun(t, ports.Run{
		ID:         "run-2",
		TaskID:     "task-1",
		RunnerKind: "codex",
		State:      "completed",
	}, "2026-03-31T10:00:00.000000000Z")
	harness.writeArtifactFile(t, "tasks/task-1/assistant-0001.txt", "older\n")
	harness.writeArtifactFile(t, "tasks/task-1/assistant-0002.txt", "selected previous\n")
	harness.writeArtifactFile(t, "tasks/task-1/assistant-0003.txt", "current\n")
	harness.writeArtifactFile(t, "tasks/task-1/assistant-0004.txt", "should not be selected\n")
	harness.saveArtifact(t, artifactSeed{
		ID:          "artifact-0001",
		TaskID:      "task-1",
		RunID:       "run-1",
		Kind:        "assistant_summary",
		Path:        "tasks/task-1/assistant-0001.txt",
		StoragePath: "tasks/task-1/assistant-0001.txt",
		Summary:     "Older artifact",
		CreatedAt:   "2026-03-31T09:59:00.000000000Z",
	})
	harness.saveArtifact(t, artifactSeed{
		ID:          "artifact-0002",
		TaskID:      "task-1",
		RunID:       "run-1",
		Kind:        "assistant_summary",
		Path:        "tasks/task-1/assistant-0002.txt",
		StoragePath: "tasks/task-1/assistant-0002.txt",
		Summary:     "Selected previous artifact",
		CreatedAt:   "2026-03-31T10:01:00.000000000Z",
	})
	harness.saveArtifact(t, artifactSeed{
		ID:          "artifact-0004",
		TaskID:      "task-1",
		RunID:       "run-2",
		Kind:        "assistant_summary",
		Path:        "tasks/task-1/assistant-0004.txt",
		StoragePath: "tasks/task-1/assistant-0004.txt",
		Summary:     "Later stable artifact id",
		CreatedAt:   "2026-03-31T10:01:00.000000000Z",
	})
	harness.saveArtifact(t, artifactSeed{
		ID:          "artifact-0003",
		TaskID:      "task-1",
		RunID:       "run-2",
		Kind:        "assistant_summary",
		Path:        "tasks/task-1/assistant-0003.txt",
		StoragePath: "tasks/task-1/assistant-0003.txt",
		Summary:     "Current artifact",
		CreatedAt:   "2026-03-31T10:01:00.000000000Z",
	})

	view, err := newArtifactCompareHarness(harness).Execute("artifact-0003")
	require.NoError(t, err)
	require.Equal(t, "ready", view.Status)
	require.NotNil(t, view.Previous)
	require.Equal(t, "artifact-0002", view.Previous.ArtifactID)
	require.Contains(t, view.Diff.Content, "-selected previous")
	require.Contains(t, view.Diff.Content, "+current")
}

func TestArtifactCompareReturnsNoPreviousWhenNoEarlierArtifactExists(t *testing.T) {
	harness := newArtifactWorkbenchHarness(t)
	harness.seedTask(t, "project-1", "module-1", "task-1")
	harness.saveRun(t, ports.Run{
		ID:         "run-1",
		TaskID:     "task-1",
		RunnerKind: "codex",
		State:      "completed",
	}, "2026-03-31T09:00:00.000000000Z")
	harness.saveRun(t, ports.Run{
		ID:         "run-2",
		TaskID:     "task-1",
		RunnerKind: "codex",
		State:      "completed",
	}, "2026-03-31T11:00:00.000000000Z")
	harness.writeArtifactFile(t, "tasks/task-1/assistant-first.txt", "first\n")
	harness.writeArtifactFile(t, "tasks/task-1/assistant-later.txt", "later\n")
	harness.saveArtifact(t, artifactSeed{
		ID:          "artifact-first",
		TaskID:      "task-1",
		RunID:       "run-1",
		Kind:        "assistant_summary",
		Path:        "tasks/task-1/assistant-first.txt",
		StoragePath: "tasks/task-1/assistant-first.txt",
		Summary:     "First artifact",
		CreatedAt:   "2026-03-31T09:01:00.000000000Z",
	})
	harness.saveArtifact(t, artifactSeed{
		ID:          "artifact-later",
		TaskID:      "task-1",
		RunID:       "run-2",
		Kind:        "assistant_summary",
		Path:        "tasks/task-1/assistant-later.txt",
		StoragePath: "tasks/task-1/assistant-later.txt",
		Summary:     "Later artifact",
		CreatedAt:   "2026-03-31T11:01:00.000000000Z",
	})

	view, err := newArtifactCompareHarness(harness).Execute("artifact-first")
	require.NoError(t, err)
	require.Equal(t, "artifact-first", view.Current.ArtifactID)
	require.Equal(t, "no_previous", view.Status)
	require.Nil(t, view.Previous)
	require.Nil(t, view.Diff)
	require.NotEmpty(t, view.Messages.Title)
	require.NotEmpty(t, view.Messages.Detail)
	require.Empty(t, view.Navigation.PreviousWorkbenchURL)
}

func TestArtifactCompareReturnsUnsupportedForBinaryArtifactKinds(t *testing.T) {
	harness := newArtifactWorkbenchHarness(t)
	harness.seedTask(t, "project-1", "module-1", "task-1")
	harness.saveRun(t, ports.Run{
		ID:         "run-1",
		TaskID:     "task-1",
		RunnerKind: "codex",
		State:      "completed",
	}, "2026-03-31T09:00:00.000000000Z")
	harness.saveRun(t, ports.Run{
		ID:         "run-2",
		TaskID:     "task-1",
		RunnerKind: "codex",
		State:      "completed",
	}, "2026-03-31T10:00:00.000000000Z")
	harness.writeArtifactFile(t, "tasks/task-1/shot-prev.png", string([]byte{0x89, 'P', 'N', 'G'}))
	harness.writeArtifactFile(t, "tasks/task-1/shot-current.png", string([]byte{0x89, 'P', 'N', 'G', 0x01}))
	harness.saveArtifact(t, artifactSeed{
		ID:          "artifact-prev",
		TaskID:      "task-1",
		RunID:       "run-1",
		Kind:        "screenshot",
		Path:        "tasks/task-1/shot-prev.png",
		StoragePath: "tasks/task-1/shot-prev.png",
		Summary:     "Previous screenshot",
		CreatedAt:   "2026-03-31T09:01:00.000000000Z",
	})
	harness.saveArtifact(t, artifactSeed{
		ID:          "artifact-current",
		TaskID:      "task-1",
		RunID:       "run-2",
		Kind:        "screenshot",
		Path:        "tasks/task-1/shot-current.png",
		StoragePath: "tasks/task-1/shot-current.png",
		Summary:     "Current screenshot",
		CreatedAt:   "2026-03-31T10:01:00.000000000Z",
	})

	view, err := newArtifactCompareHarness(harness).Execute("artifact-current")
	require.NoError(t, err)
	require.Equal(t, "unsupported", view.Status)
	require.NotNil(t, view.Previous)
	require.Equal(t, "image/png", view.Current.ContentType)
	require.Equal(t, "image/png", view.Previous.ContentType)
	require.Nil(t, view.Diff)
	require.NotEmpty(t, view.Messages.Title)
	require.NotEmpty(t, view.Messages.Detail)
}

func TestArtifactCompareReturnsReadyForJSONArtifacts(t *testing.T) {
	harness := newArtifactWorkbenchHarness(t)
	harness.seedTask(t, "project-1", "module-1", "task-1")
	harness.saveRun(t, ports.Run{
		ID:         "run-1",
		TaskID:     "task-1",
		RunnerKind: "codex",
		State:      "completed",
	}, "2026-03-31T09:00:00.000000000Z")
	harness.saveRun(t, ports.Run{
		ID:         "run-2",
		TaskID:     "task-1",
		RunnerKind: "codex",
		State:      "completed",
	}, "2026-03-31T10:00:00.000000000Z")
	harness.writeArtifactFile(t, "tasks/task-1/report-prev.json", "{\"status\":\"old\",\"count\":1}\n")
	harness.writeArtifactFile(t, "tasks/task-1/report-current.json", "{\"status\":\"new\",\"count\":2}\n")
	harness.saveArtifact(t, artifactSeed{
		ID:          "artifact-prev",
		TaskID:      "task-1",
		RunID:       "run-1",
		Kind:        "command_result",
		Path:        "tasks/task-1/report-prev.json",
		StoragePath: "tasks/task-1/report-prev.json",
		Summary:     "Previous JSON report",
		CreatedAt:   "2026-03-31T09:01:00.000000000Z",
	})
	harness.saveArtifact(t, artifactSeed{
		ID:          "artifact-current",
		TaskID:      "task-1",
		RunID:       "run-2",
		Kind:        "command_result",
		Path:        "tasks/task-1/report-current.json",
		StoragePath: "tasks/task-1/report-current.json",
		Summary:     "Current JSON report",
		CreatedAt:   "2026-03-31T10:01:00.000000000Z",
	})

	view, err := newArtifactCompareHarness(harness).Execute("artifact-current")
	require.NoError(t, err)
	require.Equal(t, "ready", view.Status)
	require.Equal(t, "application/json", view.Current.ContentType)
	require.Equal(t, "application/json", view.Previous.ContentType)
	require.NotNil(t, view.Diff)
	require.Contains(t, view.Diff.Content, "-{\"status\":\"old\",\"count\":1}")
	require.Contains(t, view.Diff.Content, "+{\"status\":\"new\",\"count\":2}")
}

func TestArtifactCompareReturnsTooLargeWhenEitherArtifactExceedsLimit(t *testing.T) {
	harness := newArtifactWorkbenchHarness(t)
	harness.seedTask(t, "project-1", "module-1", "task-1")
	harness.saveRun(t, ports.Run{
		ID:         "run-1",
		TaskID:     "task-1",
		RunnerKind: "codex",
		State:      "completed",
	}, "2026-03-31T09:00:00.000000000Z")
	harness.saveRun(t, ports.Run{
		ID:         "run-2",
		TaskID:     "task-1",
		RunnerKind: "codex",
		State:      "completed",
	}, "2026-03-31T10:00:00.000000000Z")
	harness.writeArtifactFile(t, "tasks/task-1/assistant-prev.txt", "small\n")
	harness.writeArtifactFile(t, "tasks/task-1/assistant-current.txt", strings.Repeat("x", artifactCompareMaxBytes+1))
	harness.saveArtifact(t, artifactSeed{
		ID:          "artifact-prev",
		TaskID:      "task-1",
		RunID:       "run-1",
		Kind:        "assistant_summary",
		Path:        "tasks/task-1/assistant-prev.txt",
		StoragePath: "tasks/task-1/assistant-prev.txt",
		Summary:     "Previous summary",
		CreatedAt:   "2026-03-31T09:01:00.000000000Z",
	})
	harness.saveArtifact(t, artifactSeed{
		ID:          "artifact-current",
		TaskID:      "task-1",
		RunID:       "run-2",
		Kind:        "assistant_summary",
		Path:        "tasks/task-1/assistant-current.txt",
		StoragePath: "tasks/task-1/assistant-current.txt",
		Summary:     "Current summary",
		CreatedAt:   "2026-03-31T10:01:00.000000000Z",
	})

	view, err := newArtifactCompareHarness(harness).Execute("artifact-current")
	require.NoError(t, err)
	require.Equal(t, "too_large", view.Status)
	require.NotNil(t, view.Previous)
	require.Nil(t, view.Diff)
	require.Equal(t, artifactCompareMaxBytes, view.Limits.MaxCompareBytes)
	require.NotEmpty(t, view.Messages.Title)
	require.NotEmpty(t, view.Messages.Detail)
}

func TestArtifactCompareIncludesCurrentAndPreviousWorkbenchURLs(t *testing.T) {
	harness := newArtifactWorkbenchHarness(t)
	harness.seedTask(t, "project-1", "module-1", "task-1")
	harness.saveRun(t, ports.Run{
		ID:         "run-1",
		TaskID:     "task-1",
		RunnerKind: "codex",
		State:      "completed",
	}, "2026-03-31T09:00:00.000000000Z")
	harness.saveRun(t, ports.Run{
		ID:         "run-2",
		TaskID:     "task-1",
		RunnerKind: "codex",
		State:      "completed",
	}, "2026-03-31T10:00:00.000000000Z")
	harness.writeArtifactFile(t, "tasks/task-1/assistant-prev.txt", "before\n")
	harness.writeArtifactFile(t, "tasks/task-1/assistant-current.txt", "after\n")
	harness.saveArtifact(t, artifactSeed{
		ID:          "artifact-prev",
		TaskID:      "task-1",
		RunID:       "run-1",
		Kind:        "assistant_summary",
		Path:        "tasks/task-1/assistant-prev.txt",
		StoragePath: "tasks/task-1/assistant-prev.txt",
		Summary:     "Previous summary",
		CreatedAt:   "2026-03-31T09:01:00.000000000Z",
	})
	harness.saveArtifact(t, artifactSeed{
		ID:          "artifact-current",
		TaskID:      "task-1",
		RunID:       "run-2",
		Kind:        "assistant_summary",
		Path:        "tasks/task-1/assistant-current.txt",
		StoragePath: "tasks/task-1/assistant-current.txt",
		Summary:     "Current summary",
		CreatedAt:   "2026-03-31T10:01:00.000000000Z",
	})

	view, err := newArtifactCompareHarness(harness).Execute("artifact-current")
	require.NoError(t, err)
	require.Equal(t, "/board/artifacts/workbench?artifact_id=artifact-current", view.Navigation.CurrentWorkbenchURL)
	require.Equal(t, "/board/artifacts/workbench?artifact_id=artifact-prev", view.Navigation.PreviousWorkbenchURL)
	require.Equal(t, "/board/runs/workbench?run_id=run-2", view.Navigation.BackToRunURL)
}

func newArtifactCompareHarness(harness *artifactWorkbenchHarness) *ArtifactCompareQuery {
	return NewArtifactCompareQuery(
		sqlitestore.NewBoardQueryRepository(harness.db),
		artifactfs.New(harness.root),
	)
}
