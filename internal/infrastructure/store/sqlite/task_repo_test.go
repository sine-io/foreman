package sqlite

import (
	"database/sql"
	"testing"

	"github.com/sine-io/foreman/internal/domain/approval"
	"github.com/sine-io/foreman/internal/ports"
	"github.com/stretchr/testify/require"
)

func TestOnlyOneActiveLeaseCanExistForScope(t *testing.T) {
	db := OpenTestDB(t)
	taskID := seedTaskGraph(t, db)
	repo := NewLeaseRepository(db)

	err := repo.Acquire(taskID, "repo:project-1")
	require.NoError(t, err)

	err = repo.Acquire(taskID, "repo:project-1")
	require.Error(t, err)
}

func TestArtifactIndexRoundTrip(t *testing.T) {
	db := OpenTestDB(t)
	taskID := seedTaskGraph(t, db)
	repo := NewArtifactRepository(db)

	id, err := repo.Create(taskID, "assistant_summary", "artifacts/tasks/task-1/assistant.txt")
	require.NoError(t, err)

	row, err := repo.Get(id)
	require.NoError(t, err)
	require.Equal(t, "assistant_summary", row.Kind)
	require.Equal(t, taskID, row.TaskID)
	require.Equal(t, "artifacts/tasks/task-1/assistant.txt", row.Path)
}

func TestRunRepositoryFindByTaskUsesInsertionOrder(t *testing.T) {
	db := OpenTestDB(t)
	taskID := seedTaskGraph(t, db)
	repo := NewRunRepository(db)

	require.NoError(t, repo.Save(ports.Run{
		ID:         "run-9",
		TaskID:     taskID,
		RunnerKind: "codex",
		State:      "running",
	}))
	require.NoError(t, repo.Save(ports.Run{
		ID:         "run-10",
		TaskID:     taskID,
		RunnerKind: "codex",
		State:      "completed",
	}))

	row, err := repo.FindByTask(taskID)
	require.NoError(t, err)
	require.Equal(t, "run-10", row.ID)
	require.Equal(t, "completed", row.State)
}

func TestApprovalRepositoryFindLatestByTaskUsesInsertionOrder(t *testing.T) {
	db := OpenTestDB(t)
	taskID := seedTaskGraph(t, db)
	repo := NewApprovalRepository(db)

	require.NoError(t, repo.Save(approval.New("approval-9", taskID, "first")))
	second := approval.New("approval-10", taskID, "second")
	second.Status = approval.StatusApproved
	require.NoError(t, repo.Save(second))

	row, err := repo.FindLatestByTask(taskID)
	require.NoError(t, err)
	require.Equal(t, "approval-10", row.ID)
	require.Equal(t, approval.StatusApproved, row.Status)
}

func TestOnlyOnePendingApprovalCanExistForTask(t *testing.T) {
	db := OpenTestDB(t)
	taskID := seedTaskGraph(t, db)
	repo := NewApprovalRepository(db)

	require.NoError(t, repo.Save(approval.New("approval-1", taskID, "first")))
	err := repo.Save(approval.New("approval-2", taskID, "second"))
	require.Error(t, err)
}

func seedTaskGraph(t *testing.T, db *sql.DB) string {
	t.Helper()

	mustExec(t, db, `insert into projects (id, name, repo_root) values (?, ?, ?)`, "project-1", "Foreman", "/tmp/foreman")
	mustExec(t, db, `insert into modules (id, project_id, name, board_state) values (?, ?, ?, ?)`, "module-1", "project-1", "bootstrap", "planned")
	mustExec(t, db, `insert into tasks (id, module_id, task_type, state, write_scope) values (?, ?, ?, ?, ?)`, "task-1", "module-1", "write", "ready", "repo:project-1")

	return "task-1"
}

func mustExec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()

	_, err := db.Exec(query, args...)
	require.NoError(t, err)
}
