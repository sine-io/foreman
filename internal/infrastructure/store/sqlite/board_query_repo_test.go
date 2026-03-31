package sqlite

import (
	"database/sql"
	"testing"

	"github.com/sine-io/foreman/internal/ports"
	"github.com/stretchr/testify/require"
)

func TestArtifactCompareBoardQueryRepositoryReturnsPreviousArtifactByCreatedAtAndArtifactID(t *testing.T) {
	db := OpenTestDB(t)
	taskID := seedTaskGraph(t, db)
	repo := NewBoardQueryRepository(db)

	saveRunRow(t, db, testRun("run-1", taskID), "2026-03-31T09:00:00.000000000Z")
	saveRunRow(t, db, testRun("run-2", taskID), "2026-03-31T10:00:00.000000000Z")

	mustExec(
		t,
		db,
		`insert into tasks (id, module_id, summary, acceptance, priority, task_type, state, write_scope) values (?, ?, ?, ?, ?, ?, ?, ?)`,
		"task-2",
		"module-1",
		"task-2 summary",
		"task-2 acceptance",
		5,
		"write",
		"completed",
		"repo:project-1",
	)
	saveRunRow(t, db, testRun("run-3", "task-2"), "2026-03-31T08:00:00.000000000Z")

	saveArtifactCompareSeed(t, db, artifactCompareRepoSeed{
		ID:        "artifact-0001",
		TaskID:    taskID,
		RunID:     "run-1",
		Kind:      "assistant_summary",
		Path:      "tasks/task-1/assistant-0001.txt",
		CreatedAt: "2026-03-31T09:30:00.000000000Z",
	})
	saveArtifactCompareSeed(t, db, artifactCompareRepoSeed{
		ID:        "artifact-0002",
		TaskID:    taskID,
		RunID:     "run-1",
		Kind:      "assistant_summary",
		Path:      "tasks/task-1/assistant-0002.txt",
		CreatedAt: "2026-03-31T10:00:00.000000000Z",
	})
	saveArtifactCompareSeed(t, db, artifactCompareRepoSeed{
		ID:        "artifact-0004",
		TaskID:    taskID,
		RunID:     "run-2",
		Kind:      "assistant_summary",
		Path:      "tasks/task-1/assistant-0004.txt",
		CreatedAt: "2026-03-31T10:00:00.000000000Z",
	})
	saveArtifactCompareSeed(t, db, artifactCompareRepoSeed{
		ID:        "artifact-0100",
		TaskID:    taskID,
		RunID:     "run-2",
		Kind:      "command_result",
		Path:      "tasks/task-1/command.txt",
		CreatedAt: "2026-03-31T09:59:00.000000000Z",
	})
	saveArtifactCompareSeed(t, db, artifactCompareRepoSeed{
		ID:        "artifact-0200",
		TaskID:    "task-2",
		RunID:     "run-3",
		Kind:      "assistant_summary",
		Path:      "tasks/task-2/assistant.txt",
		CreatedAt: "2026-03-31T11:00:00.000000000Z",
	})
	saveArtifactCompareSeed(t, db, artifactCompareRepoSeed{
		ID:        "artifact-0003",
		TaskID:    taskID,
		RunID:     "run-2",
		Kind:      "assistant_summary",
		Path:      "tasks/task-1/current.txt",
		CreatedAt: "2026-03-31T10:00:00.000000000Z",
	})

	row, err := repo.GetArtifactCompare("artifact-0003")
	require.NoError(t, err)
	require.Equal(t, "artifact-0003", row.Current.ArtifactID)
	require.Equal(t, "run-2", row.Current.RunID)
	require.Equal(t, "assistant_summary", row.Current.Kind)
	require.Equal(t, "2026-03-31T10:00:00.000000000Z", row.Current.CreatedAt)
	require.NotNil(t, row.Previous)
	require.Equal(t, "artifact-0002", row.Previous.ArtifactID)
	require.Equal(t, "run-1", row.Previous.RunID)
	require.Equal(t, taskID, row.Previous.TaskID)
	require.Equal(t, "assistant_summary", row.Previous.Kind)
	require.Equal(t, "2026-03-31T10:00:00.000000000Z", row.Previous.CreatedAt)
}

func TestArtifactCompareBoardQueryRepositoryReturnsNoPreviousArtifactWhenCurrentIsFirst(t *testing.T) {
	db := OpenTestDB(t)
	taskID := seedTaskGraph(t, db)
	repo := NewBoardQueryRepository(db)

	saveRunRow(t, db, testRun("run-1", taskID), "2026-03-31T09:00:00.000000000Z")
	saveRunRow(t, db, testRun("run-2", taskID), "2026-03-31T11:00:00.000000000Z")

	saveArtifactCompareSeed(t, db, artifactCompareRepoSeed{
		ID:        "artifact-first",
		TaskID:    taskID,
		RunID:     "run-1",
		Kind:      "assistant_summary",
		Path:      "tasks/task-1/assistant-first.txt",
		CreatedAt: "2026-03-31T09:01:00.000000000Z",
	})
	saveArtifactCompareSeed(t, db, artifactCompareRepoSeed{
		ID:        "artifact-later",
		TaskID:    taskID,
		RunID:     "run-2",
		Kind:      "assistant_summary",
		Path:      "tasks/task-1/assistant-later.txt",
		CreatedAt: "2026-03-31T11:01:00.000000000Z",
	})
	saveArtifactCompareSeed(t, db, artifactCompareRepoSeed{
		ID:        "artifact-other-kind",
		TaskID:    taskID,
		RunID:     "run-2",
		Kind:      "command_result",
		Path:      "tasks/task-1/command.txt",
		CreatedAt: "2026-03-31T08:59:00.000000000Z",
	})

	row, err := repo.GetArtifactCompare("artifact-first")
	require.NoError(t, err)
	require.Equal(t, "artifact-first", row.Current.ArtifactID)
	require.Nil(t, row.Previous)
}

type artifactCompareRepoSeed struct {
	ID          string
	TaskID      string
	RunID       string
	Kind        string
	Path        string
	StoragePath string
	CreatedAt   string
}

func saveArtifactCompareSeed(t *testing.T, db *sql.DB, artifact artifactCompareRepoSeed) {
	t.Helper()

	storagePath := artifact.StoragePath
	if storagePath == "" {
		storagePath = artifact.Path
	}

	_, err := db.Exec(
		`insert into artifacts (id, task_id, run_id, kind, path, storage_path, summary, created_at) values (?, ?, ?, ?, ?, ?, ?, ?)`,
		artifact.ID,
		artifact.TaskID,
		artifact.RunID,
		artifact.Kind,
		artifact.Path,
		storagePath,
		artifact.ID,
		artifact.CreatedAt,
	)
	require.NoError(t, err)
}

func testRun(runID, taskID string) ports.Run {
	return ports.Run{
		ID:         runID,
		TaskID:     taskID,
		RunnerKind: "codex",
		State:      "completed",
	}
}
