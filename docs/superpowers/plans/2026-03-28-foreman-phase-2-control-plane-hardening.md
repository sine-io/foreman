# Foreman Phase 2 Control-Plane Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Harden Foreman’s control plane so manager-facing state is deterministic, approval/dispatch transitions are atomic under retry pressure, and task status can be reconstructed reliably from persisted state.

**Architecture:** This plan stays entirely inside Foreman’s control-plane layer. It strengthens persistence ordering, query reconstruction, and command-side atomicity without adding ACP, gateway, or new runner scope. The end state should make retries, reconnects, and concurrent dispatch attempts safe and auditable.

**Tech Stack:** Go, SQLite, existing command/query handlers, existing manager-agent service, existing board/read model packages

---

## Scope Check

This plan intentionally covers only the second Phase 2 sub-project:

- deterministic persistence ordering
- dedicated task-status reconstruction
- atomic/idempotent dispatch + approval transitions
- artifact metadata hardening

Explicitly out of scope for this plan:

- Nanobot adapter
- ZeroClaw adapter
- ACP adapter
- board visual redesign
- new runner types
- multi-user auth/RBAC

Follow-on Phase 2 plans should cover:

- upstream adapter packages beyond OpenClaw
- richer board/operator UX
- approval policy expansion and policy profiles

## File Structure

### Migration and SQLite infrastructure

- Modify: `internal/infrastructure/store/sqlite/db.go`
  Responsibility change: replace single embedded schema bootstrap with ordered migration execution.
- Create: `internal/infrastructure/store/sqlite/migrations/002_control_plane_hardening.sql`
  Responsibility: add explicit ordering and status metadata columns needed for deterministic lookup and hardening.
- Modify: `internal/infrastructure/store/sqlite/db_test.go`
  Responsibility change: verify both initial and follow-on migrations are applied.

### Repository hardening

- Modify: `internal/domain/approval/approval.go`
  Responsibility change: carry explicit approval ordering metadata used by deterministic latest-approval lookup.
- Modify: `internal/ports/repositories.go`
  Responsibility change: expose the deterministic latest-state and richer artifact/status repository methods required by the hardened read/write paths.
- Modify: `internal/infrastructure/store/sqlite/run_repo.go`
  Responsibility change: use explicit ordering fields instead of `rowid`, and expose stable latest-run lookup.
- Modify: `internal/infrastructure/store/sqlite/approval_repo.go`
  Responsibility change: expose latest approval lookup with explicit ordering and preserve approval status history.
- Modify: `internal/infrastructure/store/sqlite/artifact_repo.go`
  Responsibility change: persist artifact summary metadata and expose latest-by-task lookups if needed.
- Modify: `internal/infrastructure/store/sqlite/task_repo_test.go`
  Responsibility change: verify deterministic latest-run/latest-approval ordering and unique pending approval guarantees under the hardened schema.

### Query-side status reconstruction

- Create: `internal/app/query/task_status.go`
  Responsibility: task-status query model for manager-facing status reconstruction independent of board columns.
- Create: `internal/app/query/task_status_test.go`
  Responsibility: verify completed, in-progress, and approval-gated status reconstruction.
- Modify: `internal/app/manageragent/types.go`
  Responsibility change: align manager-facing task status to the dedicated query model.
- Modify: `internal/app/manageragent/service.go`
  Responsibility change: use the dedicated task-status query instead of stitching together status ad hoc.

### Command-side atomicity

- Create: `internal/ports/transactions.go`
  Responsibility: transaction boundary abstraction for atomic command-side updates.
- Create: `internal/infrastructure/store/sqlite/dbtx.go`
  Responsibility: shared `dbtx` abstraction implemented by `*sql.DB` and `*sql.Tx`.
- Create: `internal/infrastructure/store/sqlite/tx.go`
  Responsibility: SQLite transaction runner and tx-bound repository factory used by dispatch/approval orchestration.
- Modify: `internal/infrastructure/store/sqlite/project_repo.go`
  Responsibility change: operate on shared `dbtx` instead of `*sql.DB` only.
- Modify: `internal/infrastructure/store/sqlite/module_repo.go`
  Responsibility change: operate on shared `dbtx` instead of `*sql.DB` only.
- Modify: `internal/infrastructure/store/sqlite/task_repo.go`
  Responsibility change: operate on shared `dbtx` instead of `*sql.DB` only.
- Modify: `internal/infrastructure/store/sqlite/run_repo.go`
  Responsibility change: operate on shared `dbtx` instead of `*sql.DB` only.
- Modify: `internal/infrastructure/store/sqlite/approval_repo.go`
  Responsibility change: operate on shared `dbtx` instead of `*sql.DB` only.
- Modify: `internal/infrastructure/store/sqlite/artifact_repo.go`
  Responsibility change: operate on shared `dbtx` instead of `*sql.DB` only.
- Modify: `internal/infrastructure/store/sqlite/lease_repo.go`
  Responsibility change: operate on shared `dbtx` instead of `*sql.DB` only.
- Modify: `internal/app/command/dispatch_task.go`
  Responsibility change: make approval creation, task state update, run save, artifact save, and lease release consistent under retries/concurrency.
- Modify: `internal/app/command/approve_task.go`
  Responsibility change: ensure approval resolution and task transition remain consistent with the hardened approval model.
- Modify: `internal/app/command/dispatch_task_test.go`
  Responsibility change: add retry/concurrency/idempotency expectations around approvals and run state.
- Modify: `internal/app/command/task_commands_test.go`
  Responsibility change: verify latest approval status and post-approval state semantics.

### Bootstrap and docs

- Modify: `internal/bootstrap/app.go`
  Responsibility change: wire new query/transaction dependencies into the live runtime.
- Modify: `README.md`
  Responsibility change: describe the hardened manager-facing state semantics.
- Modify: `INSTALL.md`
  Responsibility change: add smoke commands for status reconstruction and repeated dispatch behavior.
- Modify: `CHANGELOG.md`
  Responsibility change: record the control-plane hardening slice.

## Runtime Path and Ownership

The runtime path this plan hardens is:

- `internal/adapters/http/manager_handlers.go`
- `internal/app/manageragent/service.go`
- `internal/app/command/*`
- `internal/app/query/*`
- `internal/infrastructure/store/sqlite/*`
- `internal/bootstrap/app.go`

Primary ownership by package:

- `internal/infrastructure/store/sqlite`: migrations, deterministic ordering, transaction plumbing
- `internal/app/command`: atomic write orchestration and retry-safe state transitions
- `internal/app/query` and `internal/app/manageragent`: reconstructed manager-facing status and board state
- `internal/bootstrap`: live runtime composition of the hardened pieces

Concrete runtime path being hardened:

- `internal/adapters/http/manager_handlers.go`
- `internal/bootstrap/app.go`
- `internal/app/manageragent/service.go`
- `internal/app/command/dispatch_task.go`
- `internal/app/command/approve_task.go`
- `internal/infrastructure/store/sqlite/db.go`
- `internal/infrastructure/store/sqlite/run_repo.go`
- `internal/infrastructure/store/sqlite/approval_repo.go`
- `internal/infrastructure/store/sqlite/artifact_repo.go`

## Task 1: Replace Ad-Hoc Schema Bootstrapping with Ordered Migrations

**Files:**
- Modify: `internal/infrastructure/store/sqlite/db.go`
- Create: `internal/infrastructure/store/sqlite/migrations/002_control_plane_hardening.sql`
- Modify: `internal/infrastructure/store/sqlite/db_test.go`

- [ ] **Step 1: Write the failing migration tests**

```go
func TestOpenAppliesAllMigrationsInOrder(t *testing.T) {
    db := OpenTestDB(t)
    requireColumn(t, db, "runs", "created_at")
    requireColumn(t, db, "approvals", "created_at")
    requireColumn(t, db, "artifacts", "created_at")
}

func TestOpenIsIdempotentAcrossRepeatedBoots(t *testing.T) {
    path := filepath.Join(t.TempDir(), "foreman.db")
    db, err := Open(path)
    require.NoError(t, err)
    require.NoError(t, db.Close())
    db, err = Open(path)
    require.NoError(t, err)
    require.NoError(t, db.Close())
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/infrastructure/store/sqlite -run Open`
Expected: FAIL because the second migration and ordered migration runner do not exist yet

- [ ] **Step 3: Implement ordered migrations**

```go
// db.go
func Open(path string) (*sql.DB, error) {
    db, err := sql.Open("sqlite", path)
    ...
    if err := applyMigrations(db); err != nil { ... }
    return db, nil
}
```

```sql
-- 002_control_plane_hardening.sql
create table if not exists schema_migrations (
  version text primary key
);
alter table runs add column created_at text not null default '';
alter table approvals add column created_at text not null default '';
alter table artifacts add column created_at text not null default '';
```

Migration bookkeeping rule:

- `applyMigrations(db)` must record each migration filename in `schema_migrations`
- migrations are applied exactly once in filename order
- a second `Open()` on the same DB must be a no-op

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/infrastructure/store/sqlite -run Open`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/infrastructure/store/sqlite/db.go internal/infrastructure/store/sqlite/migrations/002_control_plane_hardening.sql internal/infrastructure/store/sqlite/db_test.go
git commit -m "feat: add ordered sqlite migrations"
```

## Task 2: Make Latest State Lookups Deterministic

**Files:**
- Modify: `internal/domain/approval/approval.go`
- Modify: `internal/ports/repositories.go`
- Modify: `internal/infrastructure/store/sqlite/run_repo.go`
- Modify: `internal/infrastructure/store/sqlite/approval_repo.go`
- Modify: `internal/infrastructure/store/sqlite/artifact_repo.go`
- Modify: `internal/infrastructure/store/sqlite/task_repo_test.go`

- [ ] **Step 1: Write the failing repository tests**

```go
func TestRunRepositoryFindByTaskUsesCreatedAtOrdering(t *testing.T) {
    db := OpenTestDB(t)
    repo := NewRunRepository(db)
    saveRun(t, repo, ports.Run{ID: "run-b", TaskID: "task-1", State: "running"}, "2026-03-28T10:00:00Z")
    saveRun(t, repo, ports.Run{ID: "run-a", TaskID: "task-1", State: "completed"}, "2026-03-28T11:00:00Z")
    row, err := repo.FindByTask("task-1")
    require.NoError(t, err)
    require.Equal(t, "run-a", row.ID)
}

func TestApprovalRepositoryFindLatestByTaskUsesCreatedAtOrdering(t *testing.T) {
    db := OpenTestDB(t)
    repo := NewApprovalRepository(db)
    ...
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/infrastructure/store/sqlite -run 'FindByTask|FindLatestByTask'`
Expected: FAIL because ordering is still derived from non-explicit behavior

- [ ] **Step 3: Implement deterministic latest lookup**

```go
// repositories.go
type Run struct {
    ...
    CreatedAt string
}
```

```go
// approval.go
type Approval struct {
    ...
    CreatedAt string
}
```

```go
// run_repo.go
select ... from runs where task_id = ? order by created_at desc, id desc limit 1
```

```go
// approval_repo.go
select ... from approvals where task_id = ? order by created_at desc, id desc limit 1
```

Timestamp source and format:

- use `time.Now().UTC().Format(time.RFC3339Nano)` at write time
- migration backfill should set existing blank timestamps to a deterministic sentinel such as `1970-01-01T00:00:00Z`
- when timestamps are equal, fall back to `id desc`

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/infrastructure/store/sqlite -run 'FindByTask|FindLatestByTask'`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/domain/approval/approval.go internal/ports/repositories.go internal/infrastructure/store/sqlite/run_repo.go internal/infrastructure/store/sqlite/approval_repo.go internal/infrastructure/store/sqlite/artifact_repo.go internal/infrastructure/store/sqlite/task_repo_test.go
git commit -m "feat: make latest-state lookups deterministic"
```

## Task 3: Add a Dedicated Task-Status Query Model

**Files:**
- Create: `internal/app/query/task_status.go`
- Create: `internal/app/query/task_status_test.go`
- Modify: `internal/app/manageragent/types.go`
- Modify: `internal/app/manageragent/service.go`
- Modify: `internal/app/manageragent/service_test.go`
- Modify: `internal/bootstrap/app.go`

- [ ] **Step 1: Write the failing task-status query tests**

```go
func TestTaskStatusIncludesRunAndApprovalFields(t *testing.T) {
    q := NewTaskStatusQuery(fakeStatusRepo{})
    view, err := q.Execute("project-1", "task-1")
    require.NoError(t, err)
    require.Equal(t, "run-1", view.RunID)
    require.Equal(t, "approved", view.ApprovalState)
}

func TestTaskStatusReturnsNotFoundForCrossProjectTask(t *testing.T) {
    ...
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/app/query -run TaskStatus`
Expected: FAIL with missing query model

- [ ] **Step 3: Implement the dedicated query model**

```go
type TaskStatusView struct {
    TaskID         string
    ProjectID      string
    ModuleID       string
    Summary        string
    State          string
    RunID          string
    RunState       string
    ApprovalID     string
    ApprovalReason string
    ApprovalState  string
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/app/query -run TaskStatus`
Expected: PASS

Run: `go test ./internal/app/manageragent`
Expected: PASS with the manager-facing service now using the dedicated status query

- [ ] **Step 5: Commit**

```bash
git add internal/app/query/task_status.go internal/app/query/task_status_test.go internal/app/manageragent/types.go internal/app/manageragent/service.go internal/app/manageragent/service_test.go internal/bootstrap/app.go
git commit -m "feat: add dedicated task status query"
```

## Task 4: Make Dispatch and Approval Persistence Atomic and Retry-Safe

**Files:**
- Create: `internal/ports/transactions.go`
- Create: `internal/infrastructure/store/sqlite/dbtx.go`
- Create: `internal/infrastructure/store/sqlite/tx.go`
- Modify: `internal/infrastructure/store/sqlite/project_repo.go`
- Modify: `internal/infrastructure/store/sqlite/module_repo.go`
- Modify: `internal/infrastructure/store/sqlite/task_repo.go`
- Modify: `internal/infrastructure/store/sqlite/run_repo.go`
- Modify: `internal/infrastructure/store/sqlite/approval_repo.go`
- Modify: `internal/infrastructure/store/sqlite/artifact_repo.go`
- Modify: `internal/infrastructure/store/sqlite/lease_repo.go`
- Modify: `internal/app/command/dispatch_task.go`
- Modify: `internal/app/command/approve_task.go`
- Modify: `internal/app/command/dispatch_task_test.go`
- Modify: `internal/app/command/task_commands_test.go`
- Modify: `internal/bootstrap/app.go`

**Important implementation rule:**

- external side effects such as `Runner.Dispatch(...)` and lease acquisition/release are not made part of a single SQLite transaction
- the transaction boundary in this task is only for persisted DB state
- the plan must therefore make the DB writes atomic and define compensation for failures around external calls
- repeated non-approval dispatches must be idempotent at the control-plane layer; the implementer must not blindly call `Runner.Dispatch` again if the task already has a persisted active/completed run

- [ ] **Step 1: Write the failing orchestration tests**

```go
func TestDispatchDoesNotLeaveDuplicatePendingApprovalsUnderRetry(t *testing.T) {
    handler := NewDispatchTaskHandler(...)
    _, err := handler.Handle(DispatchTaskCommand{TaskID: "task-1"})
    require.NoError(t, err)
    _, err = handler.Handle(DispatchTaskCommand{TaskID: "task-1"})
    require.NoError(t, err)
    require.Equal(t, 1, countPendingApprovals(t))
}

func TestApproveTaskTransitionsWithinSingleTransaction(t *testing.T) {
    ...
}

func TestDispatchReleasesLeaseIfPersistenceFailsAfterRunnerReturns(t *testing.T) {
    ...
}

func TestDispatchDoesNotReinvokeRunnerWhenTaskAlreadyHasPersistedRun(t *testing.T) {
    ...
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/app/command -run 'Dispatch|Approve'`
Expected: FAIL because orchestration is not transactionally coordinated yet

- [ ] **Step 3: Implement transaction runner and atomic orchestration**

```go
type Transactor interface {
    WithinTransaction(context.Context, func(context.Context, TxRepositories) error) error
}
```

```go
// dbtx.go
type dbtx interface {
    Exec(string, ...any) (sql.Result, error)
    Query(string, ...any) (*sql.Rows, error)
    QueryRow(string, ...any) *sql.Row
}
```

```go
// tx.go
type TxRepositories struct {
    Tasks     ports.TaskRepository
    Runs      ports.RunRepository
    Approvals ports.ApprovalRepository
    Artifacts ports.ArtifactRepository
    Leases    ports.LeaseRepository
}

func (t *SQLiteTransactor) WithinTransaction(
    ctx context.Context,
    fn func(context.Context, TxRepositories) error,
) error {
    tx, err := t.db.BeginTx(ctx, nil)
    ...
    repos := bindRepositories(tx)
    return fn(ctx, repos)
}
```

```go
// dispatch_task.go
// Sequence:
// 1. Evaluate policy
// 2. If approval path: use WithinTransaction to create/reuse approval + set task waiting_approval atomically
// 3. If execution path:
//    - check for an existing persisted run/status first; if present and still authoritative, return it instead of redispatching
//    - acquire lease outside transaction
//    - call Runner.Dispatch outside transaction only when no authoritative persisted run exists
//    - use WithinTransaction to save run/artifact/task state atomically
//    - if DB persistence fails after runner success, release lease and return explicit persistence error
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/app/command -run 'Dispatch|Approve'`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ports/transactions.go internal/infrastructure/store/sqlite/tx.go internal/app/command/dispatch_task.go internal/app/command/approve_task.go internal/app/command/dispatch_task_test.go internal/app/command/task_commands_test.go internal/bootstrap/app.go
git commit -m "feat: make dispatch and approval transitions atomic"
```

## Task 5: Harden Artifact Metadata and Update Docs

**Files:**
- Modify: `internal/infrastructure/store/sqlite/artifact_repo.go`
- Modify: `README.md`
- Modify: `INSTALL.md`
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Write the failing artifact metadata expectation**

Create a short checklist in your working notes:

```text
- artifact rows carry deterministic ordering metadata
- README explains hardened control-plane guarantees
- INSTALL includes repeated-dispatch/status smoke
- CHANGELOG records the hardening slice
```

- [ ] **Step 2: Verify the docs and artifact metadata are incomplete**

Run: `rg -n "created_at|control-plane hardening|repeated dispatch|latest approval" internal/infrastructure/store/sqlite/artifact_repo.go README.md INSTALL.md CHANGELOG.md`
Expected: missing or incomplete matches

- [ ] **Step 3: Implement the minimal metadata + docs update**

```go
// artifact_repo.go
insert into artifacts (..., created_at) values (..., ?)
```

Add smoke guidance such as:

```bash
curl -X POST http://localhost:8080/api/manager/commands \
  -H 'Content-Type: application/json' \
  -d '{"kind":"create_task","summary":"git push origin main"}'
```

- [ ] **Step 4: Run verification**

Run: `rg -n "created_at|control-plane hardening|repeated dispatch|latest approval" internal/infrastructure/store/sqlite/artifact_repo.go README.md INSTALL.md CHANGELOG.md`
Expected: matches present where intended

- [ ] **Step 5: Commit**

```bash
git add internal/infrastructure/store/sqlite/artifact_repo.go README.md INSTALL.md CHANGELOG.md
git commit -m "docs: add control-plane hardening guidance"
```

## Milestone Verification

Run these after all tasks complete:

```bash
go test ./internal/infrastructure/store/sqlite
go test ./internal/app/query -run TaskStatus
go test ./internal/app/command -run 'Dispatch|Approve'
go test ./internal/app/manageragent
go test ./...
```

Manual smoke:

```bash
go run ./cmd/foreman serve
curl -X POST http://localhost:8080/api/manager/commands \
  -H 'Content-Type: application/json' \
  -d '{"kind":"create_task","summary":"git push origin main"}'
curl http://localhost:8080/api/manager/tasks/<task-id>?project_id=demo
curl http://localhost:8080/api/manager/projects/demo/board
```
