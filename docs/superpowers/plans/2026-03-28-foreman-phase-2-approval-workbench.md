# Foreman Phase 2 Approval Workbench Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a dedicated approval workbench that lets operators review pending approvals, approve and immediately continue dispatch, reject with a persisted reason, and recover `approved_pending_dispatch` tasks without reopening approval.

**Architecture:** Extend the existing approval vertical slice instead of building a second approval stack. The implementation should add approval metadata persistence and a single approval-workbench read model, then layer approval-centered commands, manager HTTP endpoints, and a dedicated board workbench UI on top of the current board/runtime path.

**Tech Stack:** Go, SQLite, existing command/query handlers, existing manager-agent service, `gin`, existing `web/board` assets

---

## Scope Check

This plan intentionally covers only the next Phase 2 sub-project:

- approval metadata persistence
- approval-centered command semantics
- approval workbench read models
- approval workbench manager API
- dedicated approval workbench board UI

Explicitly out of scope for this plan:

- batch approval actions
- websocket push
- artifact-detail page
- board-wide redesign outside approval flow
- Nanobot adapter
- ZeroClaw adapter
- ACP adapter
- RBAC or multi-user collaboration

Follow-on Phase 2 plans should cover:

- task-detail workbench
- richer run/artifact drill-down
- broader board/operator UX beyond approvals

## File Structure

### Domain and persistence groundwork

- Modify: `internal/domain/approval/approval.go`
  Responsibility change: carry persisted approval metadata such as risk level, policy rule, and rejection reason.
- Modify: `internal/domain/task/task.go`
  Responsibility change: add `approved_pending_dispatch` and define transitions needed by approval recovery.
- Modify: `internal/ports/repositories.go`
  Responsibility change: extend approval/query records and expose any approval-workbench specific repository contracts.
- Create: `internal/infrastructure/store/sqlite/migrations/004_approval_workbench.sql`
  Responsibility: add approval metadata columns needed by the workbench.
- Modify: `internal/infrastructure/store/sqlite/db_test.go`
  Responsibility change: verify the approval workbench migration is applied.
- Modify: `internal/infrastructure/store/sqlite/approval_repo.go`
  Responsibility change: persist and read workbench approval metadata.
- Modify: `internal/infrastructure/store/sqlite/task_repo_test.go`
  Responsibility change: verify approval metadata persistence and any task-state transitions that depend on the new state.

### Approval command flow

- Create: `internal/app/command/approve_approval.go`
  Responsibility: approve by `approval_id`, persist approval resolution, and continue dispatch immediately.
- Create: `internal/app/command/reject_approval.go`
  Responsibility: reject by `approval_id`, require and persist rejection reason, and return task to `ready`.
- Create: `internal/app/command/retry_approval_dispatch.go`
  Responsibility: recover `approved_pending_dispatch` by retrying dispatch without creating a new approval.
- Modify: `internal/app/command/approve_task.go`
  Responsibility change: keep old task-based approval entrypoints as a thin compatibility layer over the approval-centered handler.
- Modify: `internal/app/command/dispatch_task.go`
  Responsibility change: recognize approval-centered retry flows and preserve `approved_pending_dispatch` semantics.
- Create: `internal/app/command/approval_actions_test.go`
  Responsibility: cover approve / reject / retry-dispatch semantics.
- Modify: `internal/app/command/dispatch_task_test.go`
  Responsibility change: verify retry-dispatch does not reopen approval and does not create duplicate approval records.
- Modify: `internal/app/command/task_commands_test.go`
  Responsibility change: cover task-state compatibility with `approved_pending_dispatch`.

### Query and manager HTTP surface

- Create: `internal/app/query/approval_workbench_queue.go`
  Responsibility: queue view ordered by risk, priority, and approval creation time.
- Create: `internal/app/query/approval_workbench_detail.go`
  Responsibility: one approval review view with task/run/artifact context.
- Create: `internal/app/query/approval_workbench_test.go`
  Responsibility: verify ordering, detail assembly, and historical approval rendering.
- Modify: `internal/app/manageragent/types.go`
  Responsibility change: expose approval workbench views and action result views.
- Modify: `internal/app/manageragent/service.go`
  Responsibility change: route approval workbench queries and actions through one application service.
- Modify: `internal/adapters/http/dto.go`
  Responsibility change: add queue/detail/action DTOs for approval workbench endpoints.
- Modify: `internal/adapters/http/manager_handlers.go`
  Responsibility change: add approval workbench manager endpoints.
- Modify: `internal/adapters/http/manager_handlers_test.go`
  Responsibility change: verify queue/detail/action HTTP contracts.
- Modify: `internal/adapters/http/router.go`
  Responsibility change: register the approval workbench page route and approval manager endpoints.
- Modify: `internal/bootstrap/app.go`
  Responsibility change: wire new command/query handlers into the live runtime.
- Modify: `internal/infrastructure/store/sqlite/board_query_repo.go`
  Responsibility change: provide ordered pending approval rows and approval-detail reads backed by SQLite.

### Board workbench UI

- Create: `web/board/approval-workbench.html`
  Responsibility: dedicated approval workbench shell.
- Create: `web/board/approval-workbench.js`
  Responsibility: queue selection, detail loading, approve/reject/retry-dispatch actions, and next-item advancement.
- Modify: `web/board/index.html`
  Responsibility change: link the existing approval area to the new workbench.
- Modify: `web/board/app.js`
  Responsibility change: surface the workbench entry from the overview board.
- Modify: `web/board/styles.css`
  Responsibility change: add workbench layout and approval action styling while preserving the current visual language.
- Modify: `internal/adapters/http/board_handlers_test.go`
  Responsibility change: verify workbench route and asset exposure.

### Documentation

- Modify: `README.md`
  Responsibility change: explain the approval workbench and new approval recovery flow.
- Modify: `INSTALL.md`
  Responsibility change: add approval workbench smoke commands.
- Modify: `CHANGELOG.md`
  Responsibility change: record the approval workbench slice.

## Runtime Path and Ownership

The existing runtime path this plan extends is:

- `internal/adapters/http/router.go`
- `internal/adapters/http/board_handlers.go`
- `internal/adapters/http/manager_handlers.go`
- `internal/app/query/approval_queue.go`
- `internal/app/query/task_status.go`
- `internal/app/command/approve_task.go`
- `internal/app/command/dispatch_task.go`
- `internal/bootstrap/app.go`
- `internal/infrastructure/store/sqlite/board_query_repo.go`
- `internal/infrastructure/store/sqlite/approval_repo.go`

Primary ownership by package:

- `internal/domain` and `internal/infrastructure/store/sqlite`: approval metadata, new task state, and persistence changes
- `internal/app/command`: approve / reject / retry-dispatch orchestration
- `internal/app/query` and `internal/app/manageragent`: queue/detail read models and action result views
- `internal/adapters/http`: workbench route, manager endpoints, DTOs, and page serving
- `web/board`: dedicated approval workbench operator UI

## Task 1: Add Approval Metadata And Recovery State Groundwork

**Files:**
- Create: `internal/infrastructure/store/sqlite/migrations/004_approval_workbench.sql`
- Modify: `internal/infrastructure/store/sqlite/db.go`
- Modify: `internal/domain/approval/approval.go`
- Modify: `internal/domain/policy/policy.go`
- Modify: `internal/domain/task/task.go`
- Modify: `internal/ports/repositories.go`
- Modify: `internal/infrastructure/store/sqlite/approval_repo.go`
- Modify: `internal/infrastructure/store/sqlite/db_test.go`
- Modify: `internal/infrastructure/store/sqlite/task_repo_test.go`

- [ ] **Step 1: Write the failing persistence and state tests**

```go
func TestOpenAppliesApprovalWorkbenchMigration(t *testing.T) {
    db := openTestDB(t)
    requireColumn(t, db, "approvals", "risk_level")
    requireColumn(t, db, "approvals", "policy_rule")
    requireColumn(t, db, "approvals", "rejection_reason")
}

func TestApprovalRepositoryPersistsWorkbenchMetadata(t *testing.T) {
    repo := NewApprovalRepository(openTestDB(t))
    record := approval.Approval{
        ID:              "approval-1",
        TaskID:          "task-1",
        Reason:          "git push origin main requires approval",
        RiskLevel:       approval.RiskHigh,
        PolicyRule:      "strict.git.push.main",
        RejectionReason: "use release branch instead",
        Status:          approval.StatusRejected,
    }
    require.NoError(t, repo.Save(record))
    got, err := repo.Get("approval-1")
    require.NoError(t, err)
    require.Equal(t, approval.RiskHigh, got.RiskLevel)
    require.Equal(t, "strict.git.push.main", got.PolicyRule)
    require.Equal(t, "use release branch instead", got.RejectionReason)
}

func TestTaskCanTransitionFromApprovedPendingDispatch(t *testing.T) {
    record := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "Deploy", "repo:demo")
    record.State = task.TaskStateApprovedPendingDispatch
    require.True(t, record.CanTransition(task.TaskStateRunning))
    require.True(t, record.CanTransition(task.TaskStateCompleted))
}
```

- [ ] **Step 2: Run targeted tests to verify they fail**

Run: `go test ./internal/infrastructure/store/sqlite -run 'Open|Approval'`
Expected: FAIL because the migration and metadata fields do not exist yet

Run: `go test ./internal/domain/task -run ApprovedPendingDispatch`
Expected: FAIL because the new task state is not defined

- [ ] **Step 3: Implement minimal metadata and state support**

```sql
-- 004_approval_workbench.sql
alter table approvals add column risk_level text not null default 'medium';
alter table approvals add column policy_rule text not null default '';
alter table approvals add column rejection_reason text not null default '';
```

```go
// db.go
var migrations = []migration{
    ...
    {version: "004_approval_workbench.sql", sql: migration004},
}

// policy.go
type Decision struct {
    RequiresApproval bool
    Reason           string
    RiskLevel        approval.RiskLevel
    PolicyRule       string
}

// approval.go
type RiskLevel string
const (
    RiskCritical RiskLevel = "critical"
    RiskHigh     RiskLevel = "high"
    RiskMedium   RiskLevel = "medium"
    RiskLow      RiskLevel = "low"
)
```

```go
// task.go
const TaskStateApprovedPendingDispatch TaskState = "approved_pending_dispatch"
```

Rules:

- `risk_level`, `policy_rule`, and spec `approval_reason` are persisted approval metadata, with `approval_reason` continuing to map to the existing `approvals.reason` column
- `rejection_reason` is persisted only when reject occurs
- `task_state` is read-model data and must not be added to the approval table
- `db.go` must embed and register `004_approval_workbench.sql` so the migration actually runs
- the policy decision model must become the source of truth for `risk_level` and `policy_rule`

- [ ] **Step 4: Run targeted tests to verify they pass**

Run: `go test ./internal/infrastructure/store/sqlite -run 'Open|Approval'`
Expected: PASS

Run: `go test ./internal/domain/task -run ApprovedPendingDispatch`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/infrastructure/store/sqlite/migrations/004_approval_workbench.sql internal/infrastructure/store/sqlite/db.go internal/domain/approval/approval.go internal/domain/policy/policy.go internal/domain/task/task.go internal/ports/repositories.go internal/infrastructure/store/sqlite/approval_repo.go internal/infrastructure/store/sqlite/db_test.go internal/infrastructure/store/sqlite/task_repo_test.go
git commit -m "feat: add approval workbench metadata"
```

## Task 2: Build Approval-Centered Commands And Recovery Flow

**Files:**
- Create: `internal/app/command/approve_approval.go`
- Create: `internal/app/command/reject_approval.go`
- Create: `internal/app/command/retry_approval_dispatch.go`
- Create: `internal/app/command/approval_actions_test.go`
- Modify: `internal/app/command/approve_task.go`
- Modify: `internal/app/command/dispatch_task.go`
- Modify: `internal/app/command/dispatch_task_test.go`
- Modify: `internal/app/command/task_commands_test.go`

- [ ] **Step 1: Write failing approval-action tests**

```go
func TestApproveApprovalDispatchesImmediately(t *testing.T) {
    // pending approval by approval_id -> approve -> run starts/completes
}

func TestApproveApprovalMarksApprovedPendingDispatchWhenDispatchFails(t *testing.T) {
    // approval approved, task becomes approved_pending_dispatch, no reopened approval
}

func TestRejectApprovalPersistsReasonAndReturnsTaskToReady(t *testing.T) {
    // rejection_reason stored, task ready, approval rejected
}

func TestRetryApprovalDispatchDoesNotCreateNewApproval(t *testing.T) {
    // approved_pending_dispatch + approved approval -> retry-dispatch -> no new approval row
}

func TestApprovalActionIdempotencyAndConflicts(t *testing.T) {
    // same-direction repeat -> no-op success
    // opposite-direction or ineligible retry -> conflict
}
```

- [ ] **Step 2: Run targeted tests to verify they fail**

Run: `go test ./internal/app/command -run 'ApproveApproval|RejectApproval|RetryApprovalDispatch'`
Expected: FAIL because the approval-centered handlers do not exist

- [ ] **Step 3: Implement approval-centered command handlers**

```go
// approve_approval.go
type ApproveApprovalCommand struct { ApprovalID string }

// reject_approval.go
type RejectApprovalCommand struct {
    ApprovalID string
    Reason     string
}

// retry_approval_dispatch.go
type RetryApprovalDispatchCommand struct { ApprovalID string }
```

Rules:

- `ApproveApprovalHandler` resolves approval by `approval_id`, marks it approved, and immediately calls dispatch
- if post-approval dispatch fails, task becomes `approved_pending_dispatch`
- `RejectApprovalHandler` requires `Reason`, persists it, and returns task to `ready`
- `RetryApprovalDispatchHandler` only works for `approved` + `approved_pending_dispatch`
- repeated same-direction action must return current authoritative state without new side effects
- opposite-direction or ineligible actions must return a conflict result that the HTTP layer maps to `409`
- existing `ApproveTaskHandler` should remain as a thin compatibility wrapper for task-id based routes
- approval creation in `dispatch_task.go` must persist `risk_level` and `policy_rule` from the policy decision alongside the existing approval reason
- `ApproveApprovalHandler` should persist the approved state before reusing `DispatchTaskHandler`, so `hasApprovedDispatch()` remains the single dispatch gate

- [ ] **Step 4: Run targeted tests to verify they pass**

Run: `go test ./internal/app/command -run 'ApproveApproval|RejectApproval|RetryApprovalDispatch|Dispatch|Approve'`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/command/approve_approval.go internal/app/command/reject_approval.go internal/app/command/retry_approval_dispatch.go internal/app/command/approval_actions_test.go internal/app/command/approve_task.go internal/app/command/dispatch_task.go internal/app/command/dispatch_task_test.go internal/app/command/task_commands_test.go
git commit -m "feat: add approval workbench action handlers"
```

## Task 3: Add Workbench Read Models And Manager API

**Files:**
- Create: `internal/app/query/approval_workbench_queue.go`
- Create: `internal/app/query/approval_workbench_detail.go`
- Create: `internal/app/query/approval_workbench_test.go`
- Modify: `internal/ports/repositories.go`
- Modify: `internal/app/manageragent/types.go`
- Modify: `internal/app/manageragent/service.go`
- Modify: `internal/adapters/http/dto.go`
- Modify: `internal/adapters/http/manager_handlers.go`
- Modify: `internal/adapters/http/manager_handlers_test.go`
- Modify: `internal/adapters/http/router.go`
- Modify: `internal/bootstrap/app.go`
- Modify: `internal/infrastructure/store/sqlite/board_query_repo.go`

- [ ] **Step 1: Write failing query and HTTP tests**

```go
func TestApprovalWorkbenchQueueOrdersByRiskPriorityAndCreatedAt(t *testing.T) {}
func TestApprovalWorkbenchDetailIncludesRiskRunAndArtifacts(t *testing.T) {}
func TestApprovalWorkbenchDetailSupportsHistoricalApprovedAndRejectedViews(t *testing.T) {}
func TestManagerApprovalEndpoints(t *testing.T) {
    // GET /api/manager/projects/:id/approvals
    // GET /api/manager/approvals/:id
    // POST /api/manager/approvals/:id/approve
    // POST /api/manager/approvals/:id/reject
    // POST /api/manager/approvals/:id/retry-dispatch
}
```

- [ ] **Step 2: Run targeted tests to verify they fail**

Run: `go test ./internal/app/query -run ApprovalWorkbench`
Expected: FAIL because the workbench query models do not exist

Run: `go test ./internal/adapters/http -run Approval`
Expected: FAIL because the approval workbench manager endpoints do not exist

- [ ] **Step 3: Implement query models and HTTP contract**

Use a queue/detail split:

```go
type ApprovalWorkbenchItem struct {
    ApprovalID string `json:"approval_id"`
    TaskID     string `json:"task_id"`
    Summary    string `json:"summary"`
    RiskLevel  string `json:"risk_level"`
    Priority   int    `json:"priority"`
}

type ApprovalWorkbenchActionResponse struct {
    ApprovalID      string `json:"approval_id"`
    ApprovalState   string `json:"approval_state"`
    RejectionReason string `json:"rejection_reason,omitempty"`
    TaskID          string `json:"task_id"`
    TaskState       string `json:"task_state"`
    RunID           string `json:"run_id,omitempty"`
    RunState        string `json:"run_state,omitempty"`
}
```

Implement endpoints:

- `GET /api/manager/projects/:id/approvals`
- `GET /api/manager/approvals/:id`
- `POST /api/manager/approvals/:id/approve`
- `POST /api/manager/approvals/:id/reject`
- `POST /api/manager/approvals/:id/retry-dispatch`

Rules:

- queue ordering must follow `critical > high > medium > low`, then priority desc, then approval `created_at`; implement this as explicit server-side precedence rather than lexical SQL ordering
- detail must join current task state, latest run, assistant summary preview, and run-detail link
- if no summarized artifact text exists yet, assistant summary preview may fall back to the artifact summary column or empty preview text
- `GET /api/manager/approvals/:id` must render pending, approved, and rejected approvals, including persisted `rejection_reason`
- processed approvals must disappear from the pending queue while remaining directly viewable by `approval_id`
- not-found approvals must return a not-found result, not an empty pending view
- `internal/ports/repositories.go` must define the row types and repository methods needed by the workbench queue/detail queries before the SQLite implementation is added
- define typed or sentinel action errors for not-found versus conflict so command, manager service, and HTTP layers can map `404` and `409` consistently
- update the manager HTTP layer to map typed conflict errors to `409 Conflict` explicitly rather than falling back to generic client-error handling

- [ ] **Step 4: Run targeted tests to verify they pass**

Run: `go test ./internal/app/query -run ApprovalWorkbench`
Expected: PASS

Run: `go test ./internal/adapters/http -run Approval`
Expected: PASS

Run: `go test ./internal/bootstrap -run Approval`
Expected: PASS after wiring

- [ ] **Step 5: Commit**

```bash
git add internal/app/query/approval_workbench_queue.go internal/app/query/approval_workbench_detail.go internal/app/query/approval_workbench_test.go internal/ports/repositories.go internal/app/manageragent/types.go internal/app/manageragent/service.go internal/adapters/http/dto.go internal/adapters/http/manager_handlers.go internal/adapters/http/manager_handlers_test.go internal/adapters/http/router.go internal/bootstrap/app.go internal/infrastructure/store/sqlite/board_query_repo.go
git commit -m "feat: add approval workbench query and api"
```

## Task 4: Build The Approval Workbench UI

**Files:**
- Create: `web/board/approval-workbench.html`
- Create: `web/board/approval-workbench.js`
- Modify: `web/board/index.html`
- Modify: `web/board/app.js`
- Modify: `web/board/styles.css`
- Modify: `internal/adapters/http/router.go`
- Modify: `internal/adapters/http/board_handlers_test.go`

- [ ] **Step 1: Write the failing route and asset tests**

```go
func TestApprovalWorkbenchPageServes(t *testing.T) {
    req := httptest.NewRequest(http.MethodGet, "/board/approvals/workbench?project_id=demo", nil)
    // expect 200 and workbench shell
}

func TestApprovalWorkbenchAssetsReferenceManagerApprovalApis(t *testing.T) {
    req := httptest.NewRequest(http.MethodGet, "/board/assets/approval-workbench.js", nil)
    // expect queue/detail/action endpoint strings
}

func TestApprovalWorkbenchSelectionUsesApprovalIDInURL(t *testing.T) {
    // expect page/js to preserve approval_id in refresh-safe URL state
}

func TestApprovalWorkbenchAdvancesToNextItemAfterAction(t *testing.T) {
    // explicit UI behavior requirement from the spec
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/adapters/http -run ApprovalWorkbench`
Expected: FAIL because the route and page do not exist

- [ ] **Step 3: Implement the dedicated workbench UI**

UI rules:

- left queue, right review panel
- risk and approval reason first
- `Approve` and `Reject` always visible for pending items
- `Retry Dispatch` only visible for `approved_pending_dispatch`
- after action success, load the next queue item if one exists
- keep `approval_id` in URL query state for refresh-safe deep links
- loading a processed `approval_id` must show historical detail, not redirect back to the pending queue
- loading a missing `approval_id` must show a clear not-found state
- artifact links point to the existing `/board/runs/:id` view

Add an overview entry from `/board` into the workbench:

```html
<a href="/board/approvals/workbench?project_id=demo">Open Approval Workbench</a>
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/adapters/http -run ApprovalWorkbench`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/board/approval-workbench.html web/board/approval-workbench.js web/board/index.html web/board/app.js web/board/styles.css internal/adapters/http/router.go internal/adapters/http/board_handlers_test.go
git commit -m "feat: add approval workbench ui"
```

## Task 5: Update Docs And Smoke Instructions

**Files:**
- Modify: `README.md`
- Modify: `INSTALL.md`
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Write the failing docs checklist**

Create a short checklist in your working notes:

```text
- README links the approval workbench spec and plan
- README explains the approval workbench entry and recovery flow
- INSTALL includes approval workbench API smoke
- CHANGELOG records the approval workbench slice
```

- [ ] **Step 2: Verify the docs are incomplete**

Run: `rg -n "approval workbench|approved_pending_dispatch|retry-dispatch" README.md INSTALL.md CHANGELOG.md`
Expected: missing or incomplete matches

- [ ] **Step 3: Implement the docs update**

Add smoke guidance such as:

```bash
curl http://localhost:8080/api/manager/projects/demo/approvals
curl http://localhost:8080/api/manager/approvals/<approval-id>
curl -X POST http://localhost:8080/api/manager/approvals/<approval-id>/approve
curl -X POST http://localhost:8080/api/manager/approvals/<approval-id>/reject \
  -H 'Content-Type: application/json' \
  -d '{"reason":"use release branch instead"}'
curl -X POST http://localhost:8080/api/manager/approvals/<approval-id>/retry-dispatch
```

- [ ] **Step 4: Run verification**

Run: `rg -n "approval workbench|approved_pending_dispatch|retry-dispatch" README.md INSTALL.md CHANGELOG.md`
Expected: matches present where intended

- [ ] **Step 5: Commit**

```bash
git add README.md INSTALL.md CHANGELOG.md
git commit -m "docs: add approval workbench guidance"
```

## Milestone Verification

Run these after all tasks complete:

```bash
go test ./internal/app/command -run 'ApproveApproval|RejectApproval|RetryApprovalDispatch|Dispatch|Approve'
go test ./internal/app/query -run ApprovalWorkbench
go test ./internal/adapters/http -run 'Approval|ApprovalWorkbench'
go test ./internal/bootstrap -run Approval
go test ./...
```

Manual smoke:

```bash
go run ./cmd/foreman serve
curl -X POST http://localhost:8080/api/manager/commands \
  -H 'Content-Type: application/json' \
  -d '{"kind":"create_task","summary":"git push origin main"}'
curl http://localhost:8080/api/manager/projects/demo/approvals
curl http://localhost:8080/api/manager/approvals/<approval-id>
curl -X POST http://localhost:8080/api/manager/approvals/<approval-id>/approve
curl -X POST http://localhost:8080/api/manager/approvals/<approval-id>/reject \
  -H 'Content-Type: application/json' \
  -d '{"reason":"use release branch instead"}'
curl -X POST http://localhost:8080/api/manager/approvals/<approval-id>/retry-dispatch
```
