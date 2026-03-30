# Foreman Phase 2 Run-Detail Workbench Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a dedicated run-detail workbench that acts as the run-level troubleshooting hub, showing run state, primary summary, a task-scoped latest-artifacts approximation, related task context, and stable navigation from both task workbench and legacy run-detail routes.

**Architecture:** Extend the current run-detail query and board route rather than inventing a second execution model. The implementation should add a dedicated run-workbench read model and manager API, then layer a dedicated board page and a compatibility redirect from `/board/runs/:id` to `/board/runs/workbench?run_id=...`.

**Tech Stack:** Go, SQLite-backed read models, existing manager-agent service, `gin`, existing `web/board` assets

---

## Scope Check

This plan intentionally covers only the next Phase 2 sub-project:

- dedicated run-detail workbench read model
- run workbench manager HTTP detail endpoint
- dedicated run workbench board page
- compatibility transition from legacy `/board/runs/:id`
- docs and smoke guidance

Explicitly out of scope for this plan:

- stop / rerun controls
- inline raw log viewer
- inline artifact viewer
- artifact-detail page
- websocket push
- board-wide redesign
- approval actions on the run page

Follow-on Phase 2 plans should cover:

- richer artifact drill-down
- deeper run diagnostics / trace view
- optional future run control actions

## File Structure

### Run workbench read model

- Create: `internal/app/query/run_workbench.go`
  Responsibility: assemble one run-detail workbench view, including run state, primary summary, a task-scoped latest-artifacts approximation, related task context, and navigation URLs.
- Create: `internal/app/query/run_workbench_test.go`
  Responsibility: verify summary selection, missing-run handling, missing-summary fallback, task-artifact approximation behavior, and run-to-task linkage expectations.
- Modify: `internal/ports/repositories.go`
  Responsibility change: add the row types and repository methods required by the run workbench query.
- Modify: `internal/infrastructure/store/sqlite/board_query_repo.go`
  Responsibility change: expose an explicit sqlite-backed run-workbench data source and compatibility redirect inputs.
- Modify: `internal/app/manageragent/types.go`
  Responsibility change: expose run workbench view types to adapters.
- Modify: `internal/app/manageragent/service.go`
  Responsibility change: add manager service entrypoint for the run workbench detail query.
- Modify: `internal/app/manageragent/service_test.go`
  Responsibility change: cover the run workbench view through manageragent.Service.

### Manager HTTP and compatibility route

- Modify: `internal/adapters/http/dto.go`
  Responsibility change: add run workbench detail DTOs if the existing run-detail shape is too narrow.
- Modify: `internal/adapters/http/manager_handlers.go`
  Responsibility change: add `GET /api/manager/runs/:id/workbench` with explicit `404` / `500` semantics.
- Modify: `internal/adapters/http/manager_handlers_test.go`
  Responsibility change: verify run workbench HTTP success, missing-run `404`, and broken-linkage `500`.
- Modify: `internal/adapters/http/router.go`
  Responsibility change: add `/board/runs/workbench` page route, keep `/board/runs/:id` as compatibility entrypoint, and add manager run-workbench API route.
- Modify: `internal/bootstrap/app.go`
  Responsibility change: wire the run workbench query through manageragent.Service into the live runtime.
- Modify: `internal/bootstrap/app_test.go`
  Responsibility change: verify run workbench API and compatibility route through a live app instance.

### Run workbench UI

- Create: `web/board/run-workbench.html`
  Responsibility: dedicated run-detail workbench shell.
- Create: `web/board/run-workbench.js`
  Responsibility: load by `run_id`, render summary/artifact/task context, preserve URL state, and use server-provided navigation URLs.
- Modify: `web/board/task-workbench.js`
  Responsibility change: link latest-run section into the new run workbench URL rather than the legacy route.
- Modify: `web/board/styles.css`
  Responsibility change: add run workbench layout and troubleshooting-specific presentation while preserving current visual language.
- Modify: `internal/adapters/http/board_handlers_test.go`
  Responsibility change: verify run workbench page/asset routes, legacy route compatibility behavior, and key client-state strings.

### Documentation

- Modify: `README.md`
  Responsibility change: explain the run-detail workbench and its relationship to task workbench.
- Modify: `INSTALL.md`
  Responsibility change: add run workbench smoke commands.
- Modify: `CHANGELOG.md`
  Responsibility change: record the run-detail workbench slice.

## Runtime Path and Ownership

The existing runtime path this plan extends is:

- `internal/adapters/http/router.go`
- `internal/adapters/http/board_handlers.go`
- `internal/adapters/http/manager_handlers.go`
- `internal/app/query/run_detail.go`
- `internal/app/manageragent/service.go`
- `internal/bootstrap/app.go`
- `web/board/task-workbench.js`

Primary ownership by package:

- `internal/app/query`: run workbench view assembly and summary/artifact selection
- `internal/infrastructure/store/sqlite`: sqlite-backed run workbench read model
- `internal/app/manageragent`: run workbench service exposure and error semantics
- `internal/adapters/http`: run workbench API and compatibility route behavior
- `web/board`: dedicated page, compatibility page behavior, and task-to-run navigation

## Task 1: Add The Run Workbench Query Model

**Files:**
- Create: `internal/app/query/run_workbench.go`
- Create: `internal/app/query/run_workbench_test.go`
- Modify: `internal/ports/repositories.go`
- Modify: `internal/infrastructure/store/sqlite/board_query_repo.go`
- Modify: `internal/app/manageragent/types.go`
- Modify: `internal/app/manageragent/service.go`
- Modify: `internal/app/manageragent/service_test.go`

- [ ] **Step 1: Write the failing query and service tests**

```go
func TestRunWorkbenchUsesAssistantSummaryWhenPresent(t *testing.T) {}
func TestRunWorkbenchFallsBackToRunStateAndArtifactSummaries(t *testing.T) {}
func TestRunWorkbenchHandlesNoArtifacts(t *testing.T) {}
func TestRunWorkbenchReturnsNotFoundForMissingRun(t *testing.T) {}
func TestRunWorkbenchIncludesTaskWorkbenchURLAndArtifactTargets(t *testing.T) {}
func TestRunWorkbenchIncludesSupplementalMetadata(t *testing.T) {}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/app/query -run RunWorkbench`
Expected: FAIL because the run workbench query does not exist yet

Run: `go test ./internal/app/manageragent -run RunWorkbench`
Expected: FAIL because the manager service does not expose the run workbench view yet

- [ ] **Step 3: Implement the minimal run workbench query**

Rules:

- add an explicit run-workbench read-model row/repository method rather than stretching the current `RunDetailView`
- page identity is based on `run_id` only
- `primary_summary` selection order must be deterministic:
  1. `assistant_summary` artifact summary
  2. fallback to run state plus the most useful artifact summaries
  3. no raw log content
- artifact rows are a task-scoped latest-artifacts approximation in v1; do not add new run-to-artifact persistence or schema changes in this plan
- `artifact_target_urls` in v1 are in-page anchors or selection targets only; they are not raw filesystem paths and do not imply a dedicated artifact page
- include `task_workbench_url`
- include related `project_id`, `module_id`, `task_summary`, and key run metadata when available
- include enough fields to render the `Supplemental Run Metadata` section
- missing run returns `sql.ErrNoRows`
- broken run-to-task linkage should return an explicit non-not-found error so HTTP can map it to `500`

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/app/query -run RunWorkbench`
Expected: PASS

Run: `go test ./internal/app/manageragent -run RunWorkbench`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/query/run_workbench.go internal/app/query/run_workbench_test.go internal/ports/repositories.go internal/infrastructure/store/sqlite/board_query_repo.go internal/app/manageragent/types.go internal/app/manageragent/service.go internal/app/manageragent/service_test.go
git commit -m "feat: add run workbench query model"
```

## Task 2: Add Run Workbench Manager API And Compatibility Route

**Files:**
- Modify: `internal/adapters/http/dto.go`
- Modify: `internal/adapters/http/manager_handlers.go`
- Modify: `internal/adapters/http/manager_handlers_test.go`
- Modify: `internal/adapters/http/router.go`
- Modify: `internal/bootstrap/app.go`
- Modify: `internal/bootstrap/app_test.go`

- [ ] **Step 1: Write the failing HTTP and bootstrap tests**

```go
func TestManagerRunWorkbenchEndpointReturnsDetailView(t *testing.T) {}
func TestManagerRunWorkbenchEndpointMapsMissingRunTo404(t *testing.T) {}
func TestManagerRunWorkbenchEndpointMapsBrokenTaskLinkageTo500(t *testing.T) {}
func TestLegacyBoardRunRouteRedirectsToWorkbench(t *testing.T) {}
func TestServeExposesRunWorkbenchAPI(t *testing.T) {}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/adapters/http -run RunWorkbench`
Expected: FAIL because run workbench API and compatibility route do not exist

Run: `go test ./internal/bootstrap -run RunWorkbench`
Expected: FAIL because live wiring does not exist yet

- [ ] **Step 3: Implement the manager API and compatibility route**

Add endpoints:

- `GET /api/manager/runs/:id/workbench`
- `GET /board/runs/workbench?run_id=<run-id>`

Compatibility rule:

- `/board/runs/:id` remains in v1 but issues `302` or `303` redirect to `/board/runs/workbench?run_id=<id>`

Error rules:

- missing run -> `404`
- broken run-to-task linkage -> `500`
- no `200 + error payload` contract

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/adapters/http -run RunWorkbench`
Expected: PASS

Run: `go test ./internal/bootstrap -run RunWorkbench`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/adapters/http/dto.go internal/adapters/http/manager_handlers.go internal/adapters/http/manager_handlers_test.go internal/adapters/http/router.go internal/bootstrap/app.go internal/bootstrap/app_test.go
git commit -m "feat: add run workbench api"
```

## Task 3: Build The Run-Detail Workbench UI

**Files:**
- Create: `web/board/run-workbench.html`
- Create: `web/board/run-workbench.js`
- Modify: `web/board/task-workbench.js`
- Modify: `web/board/styles.css`
- Modify: `internal/adapters/http/board_handlers_test.go`

- [ ] **Step 1: Write the failing route and asset tests**

```go
func TestRunWorkbenchPageServes(t *testing.T) {}
func TestRunWorkbenchJavaScriptUsesRunIDURLState(t *testing.T) {}
func TestRunWorkbenchJavaScriptUsesServerProvidedTaskWorkbenchURL(t *testing.T) {}
func TestRunWorkbenchJavaScriptUsesArtifactTargetURLs(t *testing.T) {}
func TestRunWorkbenchJavaScriptIncludesRefreshControl(t *testing.T) {}
func TestRunWorkbenchJavaScriptRendersSupplementalMetadata(t *testing.T) {}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/adapters/http -run RunWorkbench`
Expected: FAIL because the workbench page and assets do not exist

- [ ] **Step 3: Implement the dedicated page and client**

UI rules:

- standalone page at `/board/runs/workbench?run_id=<run-id>`
- visible `Refresh run` control that re-fetches the current `run_id`
- top section shows run state and primary conclusion
- use server-provided `task_workbench_url`
- use server-provided `artifact_target_urls`
- artifact rows are summary rows from the task-scoped latest-artifacts approximation with in-page target behavior only
- no inline raw log content
- no stop / retry / dispatch / approval actions
- render the `Supplemental Run Metadata` section from the run workbench view
- `task-workbench.js` should now link latest runs into the run workbench route rather than the legacy `/board/runs/:id` page

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/adapters/http -run RunWorkbench`
Expected: PASS

Run: `node --check web/board/run-workbench.js`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/board/run-workbench.html web/board/run-workbench.js web/board/task-workbench.js web/board/styles.css internal/adapters/http/board_handlers_test.go
git commit -m "feat: add run workbench ui"
```

## Task 4: Update Docs And Smoke Instructions

**Files:**
- Modify: `README.md`
- Modify: `INSTALL.md`
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Write the failing docs checklist**

Create a short checklist in your working notes:

```text
- README links the run-detail workbench spec and plan
- README explains task workbench -> run workbench flow
- INSTALL includes run workbench smoke
- CHANGELOG records the run workbench slice
```

- [ ] **Step 2: Verify the docs are incomplete**

Run: `rg -n "run-detail workbench|run workbench|/api/manager/runs/.*/workbench|/board/runs/workbench" README.md INSTALL.md CHANGELOG.md`
Expected: missing or incomplete matches

- [ ] **Step 3: Implement the docs update**

Add smoke guidance such as:

```bash
curl http://localhost:8080/api/manager/runs/<run-id>/workbench
curl -I http://localhost:8080/board/runs/<run-id>
```

- [ ] **Step 4: Run verification**

Run: `rg -n "run-detail workbench|run workbench|/api/manager/runs/.*/workbench|/board/runs/workbench" README.md INSTALL.md CHANGELOG.md`
Expected: matches present where intended

- [ ] **Step 5: Commit**

```bash
git add README.md INSTALL.md CHANGELOG.md
git commit -m "docs: add run workbench guidance"
```

## Milestone Verification

Run these after all tasks complete:

```bash
go test ./internal/app/query -run RunWorkbench
go test ./internal/app/manageragent -run RunWorkbench
go test ./internal/adapters/http -run RunWorkbench
go test ./internal/bootstrap -run RunWorkbench
go test ./...
```

Manual smoke:

```bash
go run ./cmd/foreman serve
curl http://localhost:8080/api/manager/runs/<run-id>/workbench
curl -I http://localhost:8080/board/runs/<run-id>
curl http://localhost:8080/board/runs/workbench?run_id=<run-id>
```
