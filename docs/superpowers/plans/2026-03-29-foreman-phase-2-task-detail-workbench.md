# Foreman Phase 2 Task-Detail Workbench Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a dedicated task-detail workbench that acts as an operator hub for one task, exposing project-scoped task actions, latest run summary, approval status, and latest task-artifact summary from a stable `project_id` + `task_id` URL.

**Architecture:** Extend the current board/read-model and manager HTTP surface rather than adding a second task stack. The implementation should add one dedicated task workbench query model with a server-computed action matrix, then layer project-scoped manager action endpoints and a dedicated board workbench UI on top of the existing task/run/approval surfaces.

**Tech Stack:** Go, SQLite-backed repositories, existing command handlers, existing manager-agent service, `gin`, existing `web/board` assets

---

## Scope Check

This plan intentionally covers only the next Phase 2 sub-project:

- task-detail workbench read model
- project-scoped task workbench action endpoints
- dedicated task-detail workbench page and client
- docs and smoke coverage for the new task operator surface

Explicitly out of scope for this plan:

- full run history on the task page
- inline artifact viewer
- approval decision actions on the task page
- event timeline/audit page
- websocket push
- artifact-detail page
- board-wide redesign outside the task-detail flow
- RBAC or multi-user collaboration

Follow-on Phase 2 plans should cover:

- richer run-detail workbench
- artifact drill-down UX
- cross-page operator workflow polish beyond task detail

## File Structure

### Task workbench read model

- Create: `internal/app/query/task_workbench.go`
  Responsibility: assemble one task-detail workbench view, including latest run summary, approval summary, latest task-artifact summary, and the per-state action matrix.
- Create: `internal/app/query/task_workbench_test.go`
  Responsibility: verify action availability, disabled reasons, missing-run/no-approval states, and cross-project task rejection.
- Modify: `internal/app/manageragent/types.go`
  Responsibility change: expose task workbench view and action-result types to adapters.
- Modify: `internal/app/manageragent/service.go`
  Responsibility change: expose task workbench detail read API and reuse project validation for task-scoped actions.
- Modify: `internal/app/manageragent/service_test.go`
  Responsibility change: verify task workbench detail and action orchestration through the service layer.

### Project-scoped task action HTTP surface

- Modify: `internal/adapters/http/dto.go`
  Responsibility change: add task workbench detail and action DTOs, including reprioritize request body and compact action result response.
- Modify: `internal/adapters/http/manager_handlers.go`
  Responsibility change: add project-scoped task workbench detail and action endpoints with `404`/`409` mapping.
- Modify: `internal/adapters/http/manager_handlers_test.go`
  Responsibility change: verify detail endpoint, project-scoped actions, conflict handling, and compact action responses.
- Modify: `internal/adapters/http/router.go`
  Responsibility change: register `/board/tasks/workbench` and `/api/manager/tasks/:id/*` task-workbench routes.
- Modify: `internal/bootstrap/app.go`
  Responsibility change: wire the task workbench query and task action service methods into the live runtime.

### Board workbench UI

- Create: `web/board/task-workbench.html`
  Responsibility: dedicated task-detail workbench shell.
- Create: `web/board/task-workbench.js`
  Responsibility: load the task workbench view, preserve `project_id` + `task_id` URL state, render actions with disabled reasons, and re-fetch detail after actions.
- Modify: `web/board/index.html`
  Responsibility change: let operators open the task workbench from board overview affordances.
- Modify: `web/board/app.js`
  Responsibility change: link task cards into the task-detail workbench.
- Modify: `web/board/approval-workbench.js`
  Responsibility change: link the approval workbench right-side detail panel into the task-detail workbench.
- Modify: `web/board/styles.css`
  Responsibility change: add task workbench layout and state styles while preserving the current board language.
- Modify: `internal/adapters/http/board_handlers_test.go`
  Responsibility change: verify task workbench page/asset routes and critical client-state strings.

### Documentation

- Modify: `README.md`
  Responsibility change: explain the task-detail workbench entry flow and its relationship to board, approval workbench, and run detail.
- Modify: `INSTALL.md`
  Responsibility change: add task workbench smoke commands and expected operator flows.
- Modify: `CHANGELOG.md`
  Responsibility change: record the task-detail workbench slice.

## Runtime Path and Ownership

The existing runtime path this plan extends is:

- `internal/adapters/http/router.go`
- `internal/adapters/http/manager_handlers.go`
- `internal/app/manageragent/service.go`
- `internal/app/query/task_status.go`
- `internal/app/query/run_detail.go`
- `internal/bootstrap/app.go`
- `web/board/app.js`
- `web/board/approval-workbench.js`

Primary ownership by package:

- `internal/app/query`: task workbench detail assembly and action matrix
- `internal/app/manageragent`: project validation and task-action orchestration
- `internal/adapters/http`: task workbench endpoints and DTOs
- `web/board`: dedicated task page plus entry links from board and approval workbench
- `internal/bootstrap`: live wiring for the new query/action surface

## Task 1: Add The Task Workbench Query Model

**Files:**
- Create: `internal/app/query/task_workbench.go`
- Create: `internal/app/query/task_workbench_test.go`
- Modify: `internal/app/manageragent/types.go`
- Modify: `internal/app/manageragent/service.go`
- Modify: `internal/app/manageragent/service_test.go`

- [ ] **Step 1: Write the failing query and service tests**

```go
func TestTaskWorkbenchShowsLatestRunApprovalAndTaskArtifacts(t *testing.T) {}
func TestTaskWorkbenchReturnsDisabledReasonsPerAction(t *testing.T) {}
func TestTaskWorkbenchHandlesNoRunAndNoApproval(t *testing.T) {}
func TestTaskWorkbenchRejectsCrossProjectTask(t *testing.T) {}
func TestTaskWorkbenchApprovalLinkDeepLinksToLatestApproval(t *testing.T) {}
func TestTaskWorkbenchIncludesTaskMetadataFields(t *testing.T) {}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/app/query -run TaskWorkbench`
Expected: FAIL because the task workbench query does not exist yet

Run: `go test ./internal/app/manageragent -run TaskWorkbench`
Expected: FAIL because the service does not expose the task workbench view yet

- [ ] **Step 3: Implement the minimal query model**

The task workbench view should include:

```go
type TaskWorkbenchAction struct {
    ActionID       string `json:"action_id"`
    Enabled        bool   `json:"enabled"`
    DisabledReason string `json:"disabled_reason,omitempty"`
    CurrentValue   any    `json:"current_value,omitempty"`
}
```

Rules:

- use a dedicated query/view rather than extending `TaskStatusView`
- approval summary comes from latest approval, with `approval_workbench_url` deep-linking to `approval_id=<latest-approval-id>` when one exists
- if no approval exists, keep the approval-workbench link visible but disabled with reason `No approval history`
- latest run summary uses the latest run only
- artifact summary uses the latest task artifacts approximation, not new run-to-artifact linkage
- task metadata section must include `write_scope`, `task_type`, and `acceptance`
- action availability matrix must exactly match the approved spec:
  - `ready`: dispatch/cancel/reprioritize enabled, retry disabled
  - `leased`: dispatch/cancel/reprioritize enabled, retry disabled
  - `waiting_approval`: dispatch disabled, cancel/reprioritize enabled, retry disabled
  - `approved_pending_dispatch`: dispatch disabled in favor of approval-workbench retry-dispatch, cancel/reprioritize enabled, retry disabled
  - `running`: dispatch disabled, cancel/reprioritize enabled, retry disabled
  - `failed`: retry/cancel/reprioritize enabled, dispatch disabled
  - `completed`: all write actions disabled
  - `canceled`: all write actions disabled
  - `open_latest_run`: enabled only when a latest run exists

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/app/query -run TaskWorkbench`
Expected: PASS

Run: `go test ./internal/app/manageragent -run TaskWorkbench`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/query/task_workbench.go internal/app/query/task_workbench_test.go internal/app/manageragent/types.go internal/app/manageragent/service.go internal/app/manageragent/service_test.go
git commit -m "feat: add task workbench query model"
```

## Task 2: Add Project-Scoped Task Workbench Actions And HTTP Endpoints

**Files:**
- Modify: `internal/adapters/http/dto.go`
- Modify: `internal/adapters/http/manager_handlers.go`
- Modify: `internal/adapters/http/manager_handlers_test.go`
- Modify: `internal/adapters/http/router.go`
- Modify: `internal/app/manageragent/service.go`
- Modify: `internal/app/manageragent/service_test.go`
- Modify: `internal/bootstrap/app.go`

- [ ] **Step 1: Write the failing HTTP and service tests**

```go
func TestManagerTaskWorkbenchDetailEndpointReturnsWorkView(t *testing.T) {}
func TestManagerTaskWorkbenchActionEndpointsRespectProjectScope(t *testing.T) {}
func TestManagerTaskWorkbenchActionEndpointsReturnConflictForIneligibleActions(t *testing.T) {}
func TestManagerTaskWorkbenchReprioritizeBindsPriorityBody(t *testing.T) {}
func TestTaskWorkbenchActionResponsesAreCompactAndRequireRefresh(t *testing.T) {}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/adapters/http -run TaskWorkbench`
Expected: FAIL because task workbench manager endpoints do not exist

Run: `go test ./internal/app/manageragent -run TaskWorkbenchAction`
Expected: FAIL because service action methods do not exist yet

- [ ] **Step 3: Implement project-scoped detail and action endpoints**

Add endpoints:

- `GET /api/manager/tasks/:id/workbench?project_id=<project-id>`
- `POST /api/manager/tasks/:id/dispatch?project_id=<project-id>`
- `POST /api/manager/tasks/:id/retry?project_id=<project-id>`
- `POST /api/manager/tasks/:id/cancel?project_id=<project-id>`
- `POST /api/manager/tasks/:id/reprioritize?project_id=<project-id>`

Rules:

- all action endpoints must validate that the task belongs to the supplied `project_id`
- cross-project mismatches return a not-found style result
- action results are compact payloads, not full refreshed workbench views
- reprioritize binds JSON body `{"priority": <int>}` and enforces integer `>= 1`
- ineligible actions return conflict-style results with stable operator-facing messages
- define typed or sentinel task-action errors for not-found versus conflict so the task workbench HTTP layer can map `404` and `409` consistently
- service action methods should reuse existing command handlers after eligibility validation rather than duplicating task mutation logic
- Task 2 ownership explicitly includes `internal/app/manageragent.Service`, the `internal/adapters/http.ManagerApp` interface, and `internal/bootstrap.app` wiring for the new task-workbench methods

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/adapters/http -run TaskWorkbench`
Expected: PASS

Run: `go test ./internal/app/manageragent -run TaskWorkbenchAction`
Expected: PASS

Run: `go test ./internal/bootstrap -run TaskWorkbench`
Expected: PASS after live wiring

- [ ] **Step 5: Commit**

```bash
git add internal/adapters/http/dto.go internal/adapters/http/manager_handlers.go internal/adapters/http/manager_handlers_test.go internal/adapters/http/router.go internal/app/manageragent/service.go internal/app/manageragent/service_test.go internal/bootstrap/app.go
git commit -m "feat: add task workbench endpoints"
```

## Task 3: Build The Task-Detail Workbench UI

**Files:**
- Create: `web/board/task-workbench.html`
- Create: `web/board/task-workbench.js`
- Modify: `web/board/index.html`
- Modify: `web/board/app.js`
- Modify: `web/board/approval-workbench.js`
- Modify: `web/board/styles.css`
- Modify: `internal/adapters/http/router.go`
- Modify: `internal/adapters/http/board_handlers_test.go`

- [ ] **Step 1: Write the failing route and asset tests**

```go
func TestTaskWorkbenchPageServes(t *testing.T) {}
func TestTaskWorkbenchJavaScriptUsesProjectAndTaskURLState(t *testing.T) {}
func TestTaskWorkbenchJavaScriptIncludesDisabledActionReasons(t *testing.T) {}
func TestApprovalWorkbenchDetailLinksToTaskWorkbench(t *testing.T) {}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/adapters/http -run TaskWorkbench`
Expected: FAIL because the task workbench page and client assets do not exist

- [ ] **Step 3: Implement the dedicated page and client**

UI rules:

- standalone page at `/board/tasks/workbench?project_id=<project-id>&task_id=<task-id>`
- top section shows current state and primary actions
- actions remain visible when disabled and explain why
- latest run summary shows latest run only, with link to `/board/runs/:id`
- approval summary links to the approval workbench using latest approval deep-link when one exists
- if no approval exists, show disabled approval-workbench link with reason `No approval history`
- artifact summary shows latest task artifacts approximation, not inline artifact contents
- task metadata section renders `write_scope`, `task_type`, and `acceptance`
- board overview and approval workbench must both link into the task workbench
- action success triggers a detail re-fetch using the same `project_id` + `task_id`

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/adapters/http -run TaskWorkbench`
Expected: PASS

Run: `node --check web/board/task-workbench.js`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/board/task-workbench.html web/board/task-workbench.js web/board/index.html web/board/app.js web/board/approval-workbench.js web/board/styles.css internal/adapters/http/router.go internal/adapters/http/board_handlers_test.go
git commit -m "feat: add task workbench ui"
```

## Task 4: Update Docs And Smoke Instructions

**Files:**
- Modify: `README.md`
- Modify: `INSTALL.md`
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Write the failing docs checklist**

Create a short checklist in your working notes:

```text
- README links the task-detail workbench spec and plan
- README explains board -> task workbench -> run detail flow
- INSTALL includes task workbench detail/action smoke
- CHANGELOG records the task workbench slice
```

- [ ] **Step 2: Verify the docs are incomplete**

Run: `rg -n "task-detail workbench|task workbench|/api/manager/tasks/.*/workbench|No approval history" README.md INSTALL.md CHANGELOG.md`
Expected: missing or incomplete matches

- [ ] **Step 3: Implement the docs update**

Add smoke guidance such as:

```bash
curl http://localhost:8080/api/manager/tasks/<task-id>/workbench?project_id=demo
curl -X POST "http://localhost:8080/api/manager/tasks/<task-id>/dispatch?project_id=demo"
curl -X POST "http://localhost:8080/api/manager/tasks/<task-id>/retry?project_id=demo"
curl -X POST "http://localhost:8080/api/manager/tasks/<task-id>/cancel?project_id=demo"
curl -X POST "http://localhost:8080/api/manager/tasks/<task-id>/reprioritize?project_id=demo" \
  -H 'Content-Type: application/json' \
  -d '{"priority":42}'
```

- [ ] **Step 4: Run verification**

Run: `rg -n "task-detail workbench|task workbench|/api/manager/tasks/.*/workbench|No approval history" README.md INSTALL.md CHANGELOG.md`
Expected: matches present where intended

- [ ] **Step 5: Commit**

```bash
git add README.md INSTALL.md CHANGELOG.md
git commit -m "docs: add task workbench guidance"
```

## Milestone Verification

Run these after all tasks complete:

```bash
go test ./internal/app/query -run TaskWorkbench
go test ./internal/app/manageragent -run TaskWorkbench
go test ./internal/adapters/http -run TaskWorkbench
go test ./internal/bootstrap -run TaskWorkbench
go test ./...
```

Manual smoke:

```bash
go run ./cmd/foreman serve
curl -X POST http://localhost:8080/api/manager/commands \
  -H 'Content-Type: application/json' \
  -d '{"kind":"create_task","summary":"Summarize current project status"}'
curl "http://localhost:8080/api/manager/tasks/<task-id>/workbench?project_id=demo"
curl -X POST "http://localhost:8080/api/manager/tasks/<task-id>/dispatch?project_id=demo"
curl -X POST "http://localhost:8080/api/manager/tasks/<task-id>/reprioritize?project_id=demo" \
  -H 'Content-Type: application/json' \
  -d '{"priority":42}'
curl "/board/tasks/workbench?project_id=demo&task_id=<task-id>"
curl "/board/runs/<run-id>"
```
