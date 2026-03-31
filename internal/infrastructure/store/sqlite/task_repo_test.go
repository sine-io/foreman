package sqlite

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/sine-io/foreman/internal/domain/approval"
	"github.com/sine-io/foreman/internal/infrastructure/store/artifactfs"
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
	repo, root := newTestArtifactRepository(t, db)

	id, err := repo.Create(
		taskID,
		"",
		"assistant_summary",
		filepath.Join(root, "tasks", "task-1", "assistant.txt"),
	)
	require.NoError(t, err)

	row, err := repo.Get(id)
	require.NoError(t, err)
	require.Equal(t, "assistant_summary", row.Kind)
	require.Equal(t, taskID, row.TaskID)
	require.Equal(t, "tasks/task-1/assistant.txt", row.Path)
	require.Equal(t, filepath.Join(root, "tasks", "task-1", "assistant.txt"), row.StoragePath)
}

func TestArtifactRepositoryCreatePersistsRunID(t *testing.T) {
	db := OpenTestDB(t)
	taskID := seedTaskGraph(t, db)
	repo, root := newTestArtifactRepository(t, db)

	saveRunRow(t, db, ports.Run{
		ID:         "run-1",
		TaskID:     taskID,
		RunnerKind: "codex",
		State:      "completed",
	}, "2026-03-31T09:00:00.000000000Z")

	id, err := repo.Create(
		taskID,
		"run-1",
		"assistant_summary",
		filepath.Join(root, "tasks", "task-1", "assistant.txt"),
	)
	require.NoError(t, err)

	var runID string
	err = db.QueryRow(`select run_id from artifacts where id = ?`, id).Scan(&runID)
	require.NoError(t, err)
	require.Equal(t, "run-1", runID)
}

func TestArtifactRepositoryCreateUsesArtifactRootForDisplayPath(t *testing.T) {
	db := OpenTestDB(t)
	taskID := seedTaskGraph(t, db)
	repo, root := newTestArtifactRepository(t, db)

	id, err := repo.Create(
		taskID,
		"",
		"assistant_summary",
		filepath.Join(root, "assistant_summary.txt"),
	)
	require.NoError(t, err)

	row, err := repo.Get(id)
	require.NoError(t, err)
	require.Equal(t, "assistant_summary.txt", row.Path)
	require.Equal(t, filepath.Join(root, "assistant_summary.txt"), row.StoragePath)
}

func TestArtifactRepositoryGetRoundTripsRunID(t *testing.T) {
	db := OpenTestDB(t)
	taskID := seedTaskGraph(t, db)
	repo, root := newTestArtifactRepository(t, db)

	saveRunRow(t, db, ports.Run{
		ID:         "run-1",
		TaskID:     taskID,
		RunnerKind: "codex",
		State:      "completed",
	}, "2026-03-31T09:00:00.000000000Z")

	mustExec(
		t,
		db,
		`insert into artifacts (id, task_id, run_id, kind, path, storage_path, summary, created_at) values (?, ?, ?, ?, ?, ?, ?, ?)`,
		"artifact-1",
		taskID,
		"run-1",
		"assistant_summary",
		"tasks/task-1/assistant.txt",
		filepath.Join(root, "tasks", "task-1", "assistant.txt"),
		"summary",
		"2026-03-31T09:01:00.000000000Z",
	)

	row, err := repo.Get("artifact-1")
	require.NoError(t, err)
	require.Equal(t, "run-1", row.RunID)
	require.Equal(t, "tasks/task-1/assistant.txt", row.Path)
	require.Equal(t, filepath.Join(root, "tasks", "task-1", "assistant.txt"), row.StoragePath)
}

func TestRunRepositoryFindByTaskUsesCreatedAtOrdering(t *testing.T) {
	db := OpenTestDB(t)
	taskID := seedTaskGraph(t, db)
	repo := NewRunRepository(db)

	saveRunRow(t, db, ports.Run{
		ID:         "run-9",
		TaskID:     taskID,
		RunnerKind: "codex",
		State:      "completed",
	}, "2026-03-28T11:00:00.000000000Z")
	saveRunRow(t, db, ports.Run{
		ID:         "run-10",
		TaskID:     taskID,
		RunnerKind: "codex",
		State:      "running",
	}, "2026-03-28T10:00:00.000000000Z")

	row, err := repo.FindByTask(taskID)
	require.NoError(t, err)
	require.Equal(t, "run-9", row.ID)
	require.Equal(t, "completed", row.State)
}

func TestRunRepositoryFindByTaskUsesIDDescendingTieBreak(t *testing.T) {
	db := OpenTestDB(t)
	taskID := seedTaskGraph(t, db)
	repo := NewRunRepository(db)

	saveRunRow(t, db, ports.Run{
		ID:         "run-b",
		TaskID:     taskID,
		RunnerKind: "codex",
		State:      "running",
	}, "2026-03-28T11:00:00.000000000Z")
	saveRunRow(t, db, ports.Run{
		ID:         "run-a",
		TaskID:     taskID,
		RunnerKind: "codex",
		State:      "completed",
	}, "2026-03-28T11:00:00.000000000Z")

	row, err := repo.FindByTask(taskID)
	require.NoError(t, err)
	require.Equal(t, "run-b", row.ID)
	require.Equal(t, "running", row.State)
}

func TestRunRepositoryFindByTaskUsesMixedPrecisionTimestampsCorrectly(t *testing.T) {
	db := OpenTestDB(t)
	taskID := seedTaskGraph(t, db)
	repo := NewRunRepository(db)

	saveRunRow(t, db, ports.Run{
		ID:         "run-early",
		TaskID:     taskID,
		RunnerKind: "codex",
		State:      "running",
	}, "2026-03-28T10:00:00.000000000Z")
	saveRunRow(t, db, ports.Run{
		ID:         "run-late",
		TaskID:     taskID,
		RunnerKind: "codex",
		State:      "completed",
	}, "2026-03-28T10:00:00.900000000Z")

	row, err := repo.FindByTask(taskID)
	require.NoError(t, err)
	require.Equal(t, "run-late", row.ID)
}

func TestRunRepositorySaveNormalizesCallerProvidedTimestamp(t *testing.T) {
	db := OpenTestDB(t)
	taskID := seedTaskGraph(t, db)
	repo := NewRunRepository(db)

	require.NoError(t, repo.Save(ports.Run{
		ID:         "run-1",
		TaskID:     taskID,
		RunnerKind: "codex",
		State:      "completed",
		CreatedAt:  "2026-03-28T10:00:00.9Z",
	}))

	row, err := repo.Get("run-1")
	require.NoError(t, err)
	require.Equal(t, "2026-03-28T10:00:00.900000000Z", row.CreatedAt)
}

func TestApprovalRepositoryFindLatestByTaskUsesCreatedAtOrdering(t *testing.T) {
	db := OpenTestDB(t)
	taskID := seedTaskGraph(t, db)
	repo := NewApprovalRepository(db)

	first := approval.New("approval-9", taskID, "first")
	first.Status = approval.StatusApproved
	saveApprovalRow(t, db, first, "2026-03-28T11:00:00.000000000Z")
	second := approval.New("approval-10", taskID, "second")
	second.Status = approval.StatusPending
	saveApprovalRow(t, db, second, "2026-03-28T10:00:00.000000000Z")

	row, err := repo.FindLatestByTask(taskID)
	require.NoError(t, err)
	require.Equal(t, "approval-9", row.ID)
	require.Equal(t, approval.StatusApproved, row.Status)
}

func TestApprovalRepositoryFindLatestByTaskUsesIDDescendingTieBreak(t *testing.T) {
	db := OpenTestDB(t)
	taskID := seedTaskGraph(t, db)
	repo := NewApprovalRepository(db)

	first := approval.New("approval-b", taskID, "first")
	first.Status = approval.StatusRejected
	saveApprovalRow(t, db, first, "2026-03-28T11:00:00.000000000Z")
	second := approval.New("approval-a", taskID, "second")
	second.Status = approval.StatusApproved
	saveApprovalRow(t, db, second, "2026-03-28T11:00:00.000000000Z")

	row, err := repo.FindLatestByTask(taskID)
	require.NoError(t, err)
	require.Equal(t, "approval-b", row.ID)
	require.Equal(t, approval.StatusRejected, row.Status)
}

func TestApprovalRepositoryFindLatestByTaskUsesMixedPrecisionTimestampsCorrectly(t *testing.T) {
	db := OpenTestDB(t)
	taskID := seedTaskGraph(t, db)
	repo := NewApprovalRepository(db)

	first := approval.New("approval-early", taskID, "first")
	first.Status = approval.StatusRejected
	saveApprovalRow(t, db, first, "2026-03-28T10:00:00.000000000Z")
	second := approval.New("approval-late", taskID, "second")
	second.Status = approval.StatusApproved
	saveApprovalRow(t, db, second, "2026-03-28T10:00:00.900000000Z")

	row, err := repo.FindLatestByTask(taskID)
	require.NoError(t, err)
	require.Equal(t, "approval-late", row.ID)
}

func TestApprovalRepositorySaveNormalizesCallerProvidedTimestamp(t *testing.T) {
	db := OpenTestDB(t)
	taskID := seedTaskGraph(t, db)
	repo := NewApprovalRepository(db)

	record := approval.New("approval-1", taskID, "first")
	record.CreatedAt = "2026-03-28T10:00:00.9Z"
	require.NoError(t, repo.Save(record))

	row, err := repo.Get("approval-1")
	require.NoError(t, err)
	require.Equal(t, "2026-03-28T10:00:00.900000000Z", row.CreatedAt)
}

func TestApprovalRepositoryPersistsMetadataFields(t *testing.T) {
	db := OpenTestDB(t)
	taskID := seedTaskGraph(t, db)
	repo := NewApprovalRepository(db)

	record := approval.New("approval-1", taskID, "policy blocked dispatch")
	record.Status = approval.StatusRejected
	record.RiskLevel = approval.RiskHigh
	record.PolicyRule = "strict.git_push"
	record.RejectionReason = "manual reviewer rejected the action"

	require.NoError(t, repo.Save(record))

	row, err := repo.Get("approval-1")
	require.NoError(t, err)
	require.Equal(t, approval.RiskHigh, row.RiskLevel)
	require.Equal(t, "strict.git_push", row.PolicyRule)
	require.Equal(t, "manual reviewer rejected the action", row.RejectionReason)

	latest, err := repo.FindLatestByTask(taskID)
	require.NoError(t, err)
	require.Equal(t, approval.RiskHigh, latest.RiskLevel)
	require.Equal(t, "strict.git_push", latest.PolicyRule)
	require.Equal(t, "manual reviewer rejected the action", latest.RejectionReason)
}

func TestBoardQueryRepositoryGetTaskWorkbenchIncludesLatestRunSummaryAndArtifacts(t *testing.T) {
	db := OpenTestDB(t)
	taskID := seedTaskGraph(t, db)
	repo := NewBoardQueryRepository(db)

	mustExec(
		t,
		db,
		`update tasks set summary = ?, acceptance = ?, priority = ?, task_type = ?, state = ? where id = ?`,
		"Ship task workbench query",
		"Query returns latest task workbench state",
		7,
		"write",
		"failed",
		taskID,
	)
	saveRunRow(t, db, ports.Run{
		ID:         "run-2",
		TaskID:     taskID,
		RunnerKind: "codex",
		State:      "failed",
	}, "2026-03-29T10:00:00.000000000Z")

	record := approval.New("approval-2", taskID, "git push origin main requires approval")
	record.Status = approval.StatusApproved
	saveApprovalRow(t, db, record, "2026-03-29T11:00:00.000000000Z")

	mustExec(
		t,
		db,
		`insert into artifacts (id, task_id, kind, path, summary, created_at) values (?, ?, ?, ?, ?, ?)`,
		"artifact-3",
		taskID,
		"assistant_summary",
		"artifacts/tasks/task-1/assistant_summary.txt",
		"Latest run failed after syncing artifacts",
		"2026-03-29T12:00:00.000000000Z",
	)
	mustExec(
		t,
		db,
		`insert into artifacts (id, task_id, kind, path, summary, created_at) values (?, ?, ?, ?, ?, ?)`,
		"artifact-2",
		taskID,
		"run_log",
		"artifacts/tasks/task-1/run.log",
		"runner output",
		"2026-03-29T11:59:00.000000000Z",
	)

	row, err := repo.GetTaskWorkbench(taskID)
	require.NoError(t, err)
	require.Equal(t, "task-1", row.TaskID)
	require.Equal(t, "project-1", row.ProjectID)
	require.Equal(t, "module-1", row.ModuleID)
	require.Equal(t, "Ship task workbench query", row.Summary)
	require.Equal(t, "failed", row.TaskState)
	require.Equal(t, 7, row.Priority)
	require.Equal(t, "repo:project-1", row.WriteScope)
	require.Equal(t, "write", row.TaskType)
	require.Equal(t, "Query returns latest task workbench state", row.Acceptance)
	require.Equal(t, "run-2", row.LatestRunID)
	require.Equal(t, "failed", row.LatestRunState)
	require.Equal(t, "codex run is failed", row.LatestRunSummary)
	require.Equal(t, "approval-2", row.LatestApprovalID)
	require.Equal(t, "approved", row.LatestApprovalState)
	require.Equal(t, "git push origin main requires approval", row.LatestApprovalReason)
	require.Len(t, row.Artifacts, 2)
	require.Equal(t, "assistant_summary", row.Artifacts[0].Kind)
}

func TestBoardQueryRepositoryGetRunWorkbenchUsesRequestedRunWithTaskScopedArtifactApproximation(t *testing.T) {
	db := OpenTestDB(t)
	taskID := seedTaskGraph(t, db)
	repo := NewBoardQueryRepository(db)

	mustExec(
		t,
		db,
		`update tasks set summary = ?, state = ? where id = ?`,
		"Inspect multi-run artifact approximation",
		"failed",
		taskID,
	)
	saveRunRow(t, db, ports.Run{
		ID:         "run-old",
		TaskID:     taskID,
		RunnerKind: "codex",
		State:      "failed",
	}, "2026-03-29T09:00:00.000000000Z")
	saveRunRow(t, db, ports.Run{
		ID:         "run-new",
		TaskID:     taskID,
		RunnerKind: "codex",
		State:      "completed",
	}, "2026-03-29T11:00:00.000000000Z")

	mustExec(
		t,
		db,
		`insert into artifacts (id, task_id, kind, path, summary, created_at) values (?, ?, ?, ?, ?, ?)`,
		"artifact-before-new-run",
		taskID,
		"command_result",
		"artifacts/tasks/task-1/command-before.txt",
		"Older run failed before retry",
		"2026-03-29T10:00:00.000000000Z",
	)
	mustExec(
		t,
		db,
		`insert into artifacts (id, task_id, kind, path, summary, created_at) values (?, ?, ?, ?, ?, ?)`,
		"artifact-after-new-run",
		taskID,
		"assistant_summary",
		"artifacts/tasks/task-1/assistant-latest.txt",
		"Latest run completed successfully",
		"2026-03-29T12:00:00.000000000Z",
	)

	row, err := repo.GetRunWorkbench("run-old")
	require.NoError(t, err)
	require.Equal(t, "run-old", row.RunID)
	require.Equal(t, "failed", row.RunState)
	require.Equal(t, "2026-03-29T09:00:00.000000000Z", row.RunCreatedAt)
	require.Equal(t, "Inspect multi-run artifact approximation", row.TaskSummary)
	require.Len(t, row.Artifacts, 2)
	require.Equal(t, []string{"artifact-after-new-run", "artifact-before-new-run"}, []string{row.Artifacts[0].ID, row.Artifacts[1].ID})
	require.Equal(t, "Latest run completed successfully", row.Artifacts[0].Summary)
	require.Equal(t, "Older run failed before retry", row.Artifacts[1].Summary)
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

func TestOpenBackfillsBlankCreatedAtForPreviouslyMigratedDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "backfill.db")

	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)

	_, err = db.Exec(`PRAGMA foreign_keys = ON`)
	require.NoError(t, err)
	_, err = db.Exec(initSchema)
	require.NoError(t, err)

	taskID := seedTaskGraph(t, db)
	mustExec(t, db, `insert into runs (id, task_id, runner_kind, state) values (?, ?, ?, ?)`, "run-1", taskID, "codex", "running")
	mustExec(t, db, `insert into approvals (id, task_id, reason, state) values (?, ?, ?, ?)`, "approval-1", taskID, "need approval", approval.StatusPending)
	mustExec(t, db, `insert into artifacts (id, task_id, kind, path, summary) values (?, ?, ?, ?, '')`, "artifact-1", taskID, "assistant_summary", "artifacts/tasks/task-1/assistant.txt")

	mustExec(t, db, `alter table runs add column created_at text not null default ''`)
	mustExec(t, db, `alter table approvals add column created_at text not null default ''`)
	mustExec(t, db, `alter table artifacts add column created_at text not null default ''`)
	mustExec(t, db, `create table schema_migrations (version text primary key)`)
	mustExec(t, db, `insert into schema_migrations(version) values (?)`, "001_init.sql")
	mustExec(t, db, `insert into schema_migrations(version) values (?)`, "002_control_plane_hardening.sql")
	require.NoError(t, db.Close())

	db, err = Open(path)
	require.NoError(t, err)
	defer func() { require.NoError(t, db.Close()) }()

	require.Equal(t, "1970-01-01T00:00:00.000000000Z", readCreatedAt(t, db, "runs", "run-1"))
	require.Equal(t, "1970-01-01T00:00:00.000000000Z", readCreatedAt(t, db, "approvals", "approval-1"))
	require.Equal(t, "1970-01-01T00:00:00.000000000Z", readCreatedAt(t, db, "artifacts", "artifact-1"))
}

func saveRunRow(t *testing.T, db *sql.DB, run ports.Run, createdAt string) {
	t.Helper()

	mustExec(
		t,
		db,
		`insert into runs (id, task_id, runner_kind, state, created_at) values (?, ?, ?, ?, ?)`,
		run.ID,
		run.TaskID,
		run.RunnerKind,
		run.State,
		createdAt,
	)
}

func saveApprovalRow(t *testing.T, db *sql.DB, record approval.Approval, createdAt string) {
	t.Helper()

	mustExec(
		t,
		db,
		`insert into approvals (id, task_id, reason, state, risk_level, policy_rule, rejection_reason, created_at) values (?, ?, ?, ?, ?, ?, ?, ?)`,
		record.ID,
		record.TaskID,
		record.Reason,
		record.Status,
		record.RiskLevel,
		record.PolicyRule,
		record.RejectionReason,
		createdAt,
	)
}

func readCreatedAt(t *testing.T, db *sql.DB, table, id string) string {
	t.Helper()

	var createdAt string
	err := db.QueryRow(`select created_at from `+table+` where id = ?`, id).Scan(&createdAt)
	require.NoError(t, err)

	return createdAt
}

func newTestArtifactRepository(t *testing.T, db *sql.DB) (*ArtifactRepository, string) {
	t.Helper()

	root := filepath.Join(t.TempDir(), "runtime-artifacts")
	require.NoError(t, os.MkdirAll(root, 0o755))

	return NewArtifactRepository(db, artifactfs.New(root)), root
}
