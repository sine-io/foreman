# Foreman Phase 2 Artifact Workbench Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a dedicated artifact drill-down workbench with durable artifact-to-run linkage, a manager-facing detail/content API, a bounded preview UI, same-run sibling switching, and safe navigation back to run workbench.

**Architecture:** First make artifact records durably link to runs and add a filesystem read path that can safely serve preview/raw content without exposing raw storage details to the browser. Then add a dedicated artifact workbench read model and manager API, layer a standalone board page on top, and finally update run workbench navigation plus docs/smoke guidance.

**Tech Stack:** Go, SQLite migrations and repositories, `artifactfs`, existing manager-agent service, `gin`, existing `web/board` assets

---

## Scope Check

This plan intentionally covers only the next Phase 2 sub-project:

- durable artifact-to-run linkage
- artifact workbench detail query and content access path
- manager artifact workbench and raw-content HTTP endpoints
- dedicated artifact workbench board page
- run-workbench to artifact-workbench navigation
- docs and smoke guidance

Explicitly out of scope for this plan:

- cross-run artifact comparison
- artifact editing, deletion, or regeneration
- full binary/media rendering
- inline raw log viewer
- task-wide artifact history explorer
- board-wide redesign

Follow-on Phase 2 plans should cover:

- richer artifact renderers by content type
- cross-run artifact comparison
- optional future audit/diagnostic expansion

## File Structure

### Artifact linkage and storage primitives

- Modify: `internal/ports/repositories.go`
  Responsibility change: extend artifact records and repository contracts with durable `run_id` linkage and metadata needed by artifact workbench.
- Modify: `internal/ports/artifacts.go`
  Responsibility change: expose read-oriented artifact store operations needed for bounded preview and raw-content serving.
- Create: `internal/infrastructure/store/sqlite/migrations/005_artifact_run_linkage.sql`
  Responsibility: add durable artifact `run_id` linkage and supporting index/shape changes without guessing linkage for legacy rows.
- Modify: `internal/infrastructure/store/sqlite/db.go`
  Responsibility change: register the new migration so SQLite bootstrap applies and records it.
- Modify: `internal/infrastructure/store/sqlite/db_test.go`
  Responsibility change: verify the new migration version is present in ordered/idempotent bootstrap.
- Modify: `internal/infrastructure/store/sqlite/artifact_repo.go`
  Responsibility change: persist and load artifact `run_id` linkage and richer artifact row data.
- Modify: `internal/infrastructure/store/sqlite/tx.go`
  Responsibility change: expose the updated artifact repository inside tx-bound command paths.
- Modify: `internal/infrastructure/store/sqlite/task_repo_test.go`
  Responsibility change: verify artifact repository round-trip, migration behavior, and linked/unlinked artifact cases.
- Modify: `internal/infrastructure/store/artifactfs/store.go`
  Responsibility change: add safe read/resolve helpers for preview and raw-content serving while keeping browser-facing paths sanitized.
- Create: `internal/infrastructure/store/artifactfs/store_test.go`
  Responsibility: verify bounded reads, safe resolution, and display-path sanitization behavior.
- Modify: `internal/app/command/dispatch_task.go`
  Responsibility change: persist new artifacts with durable `run_id` linkage at write time.
- Modify: `internal/app/command/dispatch_task_test.go`
  Responsibility change: verify dispatch persists linked artifacts with the authoritative run ID.

### Artifact workbench query and application service

- Create: `internal/app/query/artifact_workbench.go`
  Responsibility: assemble one artifact workbench view, including metadata, bounded preview, sibling artifacts from the same run, and navigation URLs.
- Create: `internal/app/query/artifact_workbench_test.go`
  Responsibility: verify linked artifact lookup, unlinked-artifact conflict, sibling scoping, bounded preview, non-text fallback, and navigation URL behavior.
- Modify: `internal/infrastructure/store/sqlite/board_query_repo.go`
  Responsibility change: add explicit sqlite-backed artifact workbench lookup by `artifact_id`, same-run sibling queries, and linked/unlinked artifact semantics.
- Modify: `internal/ports/repositories.go`
  Responsibility change: add artifact workbench row types and query methods required by the new read model.
- Modify: `internal/app/manageragent/types.go`
  Responsibility change: expose artifact workbench view/content types to adapters.
- Modify: `internal/app/manageragent/service.go`
  Responsibility change: add manager service entrypoints for artifact workbench detail and raw-content metadata/content access.
- Modify: `internal/app/manageragent/service_test.go`
  Responsibility change: cover artifact workbench detail/content behavior through manageragent.Service.

### Run workbench artifact-target transition

- Modify: `internal/app/query/run_workbench.go`
  Responsibility change: emit artifact-target URLs as artifact workbench deep links for linked artifacts and anchor fallbacks for legacy unlinked artifacts.
- Modify: `internal/app/query/run_workbench_test.go`
  Responsibility change: cover linked-artifact URLs versus legacy anchor fallback semantics.

### Manager HTTP and bootstrap wiring

- Modify: `internal/adapters/http/dto.go`
  Responsibility change: add artifact workbench detail DTOs and any raw-content helper responses if needed.
- Modify: `internal/adapters/http/manager_handlers.go`
  Responsibility change: add `GET /api/manager/artifacts/:id/workbench` and `GET /api/manager/artifacts/:id/content` with explicit `404` / `409` / `410` / `500` semantics and safe raw-content headers.
- Modify: `internal/adapters/http/manager_handlers_test.go`
  Responsibility change: verify artifact workbench detail success, linked/unlinked/broken-linkage error mapping, and raw-content endpoint behavior/headers.
- Modify: `internal/adapters/http/router.go`
  Responsibility change: add `/board/artifacts/workbench` page route and manager artifact routes.
- Modify: `internal/bootstrap/app.go`
  Responsibility change: wire artifact store, artifact workbench query, and raw-content path through the live runtime.
- Modify: `internal/bootstrap/app_test.go`
  Responsibility change: verify artifact workbench API/content route through a live app instance.

### Artifact workbench UI

- Create: `web/board/artifact-workbench.html`
  Responsibility: dedicated artifact workbench shell.
- Create: `web/board/artifact-workbench.js`
  Responsibility: load by `artifact_id`, render bounded preview and metadata, switch between same-run siblings, and use server-provided navigation URLs.
- Modify: `web/board/run-workbench.js`
  Responsibility change: send linked artifacts into the artifact workbench route and keep legacy anchor behavior for unlinked compatibility rows.
- Modify: `web/board/styles.css`
  Responsibility change: add artifact workbench layout and preview-specific presentation while preserving current visual language.
- Modify: `internal/adapters/http/board_handlers_test.go`
  Responsibility change: verify artifact workbench page/asset routes and key client-state strings.

### Documentation

- Modify: `README.md`
  Responsibility change: explain the artifact workbench and its relationship to run workbench.
- Modify: `INSTALL.md`
  Responsibility change: add artifact workbench smoke commands.
- Modify: `CHANGELOG.md`
  Responsibility change: record the artifact workbench slice.

## Runtime Path and Ownership

The existing runtime path this plan extends is:

- `internal/app/command/dispatch_task.go`
- `internal/infrastructure/store/sqlite/db.go`
- `internal/infrastructure/store/sqlite/artifact_repo.go`
- `internal/infrastructure/store/sqlite/board_query_repo.go`
- `internal/infrastructure/store/artifactfs/store.go`
- `internal/app/query/run_workbench.go`
- `internal/app/manageragent/service.go`
- `internal/adapters/http/manager_handlers.go`
- `internal/adapters/http/router.go`
- `internal/bootstrap/app.go`
- `web/board/run-workbench.js`

Primary ownership by package:

- `internal/infrastructure/store/sqlite`: durable artifact-to-run persistence and artifact workbench read model
- `internal/infrastructure/store/artifactfs`: safe preview/raw-content reads and path handling
- `internal/app/query`: artifact workbench view assembly and preview selection
- `internal/app/manageragent`: artifact workbench service exposure and error semantics
- `internal/adapters/http`: artifact detail/content API and page route behavior
- `web/board`: dedicated page, sibling switching, and run-to-artifact navigation

## Task 1: Add Durable Artifact-To-Run Linkage And Safe Store Reads

**Files:**
- Modify: `internal/ports/repositories.go`
- Modify: `internal/ports/artifacts.go`
- Create: `internal/infrastructure/store/sqlite/migrations/005_artifact_run_linkage.sql`
- Modify: `internal/infrastructure/store/sqlite/artifact_repo.go`
- Modify: `internal/infrastructure/store/sqlite/db.go`
- Modify: `internal/infrastructure/store/sqlite/db_test.go`
- Modify: `internal/infrastructure/store/sqlite/tx.go`
- Modify: `internal/infrastructure/store/sqlite/task_repo_test.go`
- Modify: `internal/infrastructure/store/artifactfs/store.go`
- Create: `internal/infrastructure/store/artifactfs/store_test.go`
- Modify: `internal/app/command/dispatch_task.go`
- Modify: `internal/app/command/dispatch_task_test.go`

- [ ] **Step 1: Write the failing persistence and store tests**

```go
func TestArtifactRepositoryCreatePersistsRunID(t *testing.T) {}
func TestArtifactRepositoryGetRoundTripsRunID(t *testing.T) {}
func TestOpenRegistersArtifactRunLinkageMigration(t *testing.T) {}
func TestArtifactStoreReadPreviewBoundsTextArtifacts(t *testing.T) {}
func TestArtifactStoreResolveDisplayPathStripsArtifactRoot(t *testing.T) {}
func TestDispatchPersistsArtifactLinkedToRun(t *testing.T) {}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/infrastructure/store/sqlite -run Artifact`
Expected: FAIL because artifact persistence does not yet store `run_id`

Run: `go test ./internal/infrastructure/store/sqlite -run Open`
Expected: FAIL because the new migration is not registered yet

Run: `go test ./internal/infrastructure/store/artifactfs -run Artifact`
Expected: FAIL because read/resolve helpers do not exist yet

Run: `go test ./internal/app/command -run Dispatch.*Artifact`
Expected: FAIL because dispatch does not yet persist linked artifacts

- [ ] **Step 3: Implement the minimal linkage and safe-read primitives**

Rules:

- add durable `run_id` linkage for artifacts in SQLite
- do not backfill ambiguous legacy rows by timestamp inference
- updated artifact writes must persist the authoritative `run.ID`
- artifact store read helpers must support bounded preview reads
- artifact display paths must be sanitized relative paths, not raw absolute filesystem paths
- preserve enough raw path information internally for safe server-side content reads

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/infrastructure/store/sqlite -run Artifact`
Expected: PASS

Run: `go test ./internal/infrastructure/store/sqlite -run Open`
Expected: PASS

Run: `go test ./internal/infrastructure/store/artifactfs -run Artifact`
Expected: PASS

Run: `go test ./internal/app/command -run Dispatch.*Artifact`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ports/repositories.go internal/ports/artifacts.go internal/infrastructure/store/sqlite/migrations/005_artifact_run_linkage.sql internal/infrastructure/store/sqlite/db.go internal/infrastructure/store/sqlite/db_test.go internal/infrastructure/store/sqlite/artifact_repo.go internal/infrastructure/store/sqlite/tx.go internal/infrastructure/store/sqlite/task_repo_test.go internal/infrastructure/store/artifactfs/store.go internal/infrastructure/store/artifactfs/store_test.go internal/app/command/dispatch_task.go internal/app/command/dispatch_task_test.go
git commit -m "feat: link artifacts to runs"
```

## Task 2: Add The Artifact Workbench Query And Manager Service

**Files:**
- Create: `internal/app/query/artifact_workbench.go`
- Create: `internal/app/query/artifact_workbench_test.go`
- Modify: `internal/infrastructure/store/sqlite/board_query_repo.go`
- Modify: `internal/ports/repositories.go`
- Modify: `internal/app/manageragent/types.go`
- Modify: `internal/app/manageragent/service.go`
- Modify: `internal/app/manageragent/service_test.go`
- Modify: `internal/app/query/run_workbench.go`
- Modify: `internal/app/query/run_workbench_test.go`

- [ ] **Step 1: Write the failing query and service tests**

```go
func TestArtifactWorkbenchReturnsLinkedArtifactView(t *testing.T) {}
func TestArtifactWorkbenchReturnsConflictForLegacyUnlinkedArtifact(t *testing.T) {}
func TestArtifactWorkbenchReturnsErrorForBrokenLinkage(t *testing.T) {}
func TestArtifactWorkbenchScopesSiblingsToSameRun(t *testing.T) {}
func TestArtifactWorkbenchIncludesBoundedPreviewAndTruncationFlag(t *testing.T) {}
func TestArtifactWorkbenchFallsBackForNonTextArtifact(t *testing.T) {}
func TestArtifactWorkbenchIncludesRunAndRawContentURLs(t *testing.T) {}
func TestRunWorkbenchArtifactTargetsUseArtifactWorkbenchForLinkedArtifacts(t *testing.T) {}
func TestRunWorkbenchArtifactTargetsKeepAnchorFallbackForLegacyArtifacts(t *testing.T) {}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/app/query -run ArtifactWorkbench`
Expected: FAIL because the artifact workbench query does not exist yet

Run: `go test ./internal/app/manageragent -run ArtifactWorkbench`
Expected: FAIL because manageragent.Service does not expose artifact workbench yet

- [ ] **Step 3: Implement the minimal artifact workbench query**

Rules:

- page identity is based on `artifact_id` only
- same-run sibling lookup must use persisted `run_id`
- legacy unlinked artifacts must surface a conflict path rather than guessed run ownership
- broken artifact linkage must return an explicit non-not-found error so HTTP can map it to `500`
- workbench detail should include bounded preview content directly
- non-text artifacts should remain readable via metadata + fallback rather than forced inline rendering
- the query should compute `run_workbench_url` and `raw_content_url`
- run workbench must emit dedicated artifact workbench URLs for linked artifacts and anchor fallback URLs for legacy artifacts

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/app/query -run ArtifactWorkbench`
Expected: PASS

Run: `go test ./internal/app/manageragent -run ArtifactWorkbench`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/query/artifact_workbench.go internal/app/query/artifact_workbench_test.go internal/app/query/run_workbench.go internal/app/query/run_workbench_test.go internal/infrastructure/store/sqlite/board_query_repo.go internal/ports/repositories.go internal/app/manageragent/types.go internal/app/manageragent/service.go internal/app/manageragent/service_test.go
git commit -m "feat: add artifact workbench query"
```

## Task 3: Add Artifact Manager HTTP Endpoints And Runtime Wiring

**Files:**
- Modify: `internal/adapters/http/dto.go`
- Modify: `internal/adapters/http/manager_handlers.go`
- Modify: `internal/adapters/http/manager_handlers_test.go`
- Modify: `internal/adapters/http/router.go`
- Modify: `internal/bootstrap/app.go`
- Modify: `internal/bootstrap/app_test.go`

- [ ] **Step 1: Write the failing HTTP and bootstrap tests**

```go
func TestManagerArtifactWorkbenchEndpointReturnsDetailView(t *testing.T) {}
func TestManagerArtifactWorkbenchEndpointMapsMissingArtifactTo404(t *testing.T) {}
func TestManagerArtifactWorkbenchEndpointMapsLegacyArtifactTo409(t *testing.T) {}
func TestManagerArtifactWorkbenchEndpointMapsBrokenLinkageTo500(t *testing.T) {}
func TestManagerArtifactContentEndpointReturnsSafeHeaders(t *testing.T) {}
func TestManagerArtifactContentEndpointMapsMissingFileTo410(t *testing.T) {}
func TestServeExposesArtifactWorkbenchAPI(t *testing.T) {}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/adapters/http -run ArtifactWorkbench`
Expected: FAIL because artifact workbench routes do not exist yet

Run: `go test ./internal/bootstrap -run ArtifactWorkbench`
Expected: FAIL because live wiring does not exist yet

- [ ] **Step 3: Implement the manager API and runtime wiring**

Add endpoints:

- `GET /api/manager/artifacts/:id/workbench`
- `GET /api/manager/artifacts/:id/content`

Error rules:

- missing artifact -> `404`
- legacy unlinked artifact -> `409`
- broken linkage -> `500`
- missing backing file -> `410`
- unreadable file -> `500`

Raw-content rules:

- set `X-Content-Type-Options: nosniff`
- allow inline rendering only for explicitly safe text-like types
- force `Content-Disposition: attachment` for active or untrusted content types

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/adapters/http -run ArtifactWorkbench`
Expected: PASS

Run: `go test ./internal/bootstrap -run ArtifactWorkbench`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/adapters/http/dto.go internal/adapters/http/manager_handlers.go internal/adapters/http/manager_handlers_test.go internal/adapters/http/router.go internal/bootstrap/app.go internal/bootstrap/app_test.go
git commit -m "feat: add artifact workbench api"
```

## Task 4: Build The Artifact Workbench UI And Run Navigation

**Files:**
- Create: `web/board/artifact-workbench.html`
- Create: `web/board/artifact-workbench.js`
- Modify: `web/board/run-workbench.js`
- Modify: `web/board/styles.css`
- Modify: `internal/adapters/http/board_handlers_test.go`
- Modify: `internal/adapters/http/router.go`

- [ ] **Step 1: Write the failing route and asset tests**

```go
func TestArtifactWorkbenchPageServes(t *testing.T) {}
func TestArtifactWorkbenchJavaScriptUsesArtifactIDURLState(t *testing.T) {}
func TestArtifactWorkbenchJavaScriptRendersSiblingArtifacts(t *testing.T) {}
func TestArtifactWorkbenchJavaScriptUsesServerProvidedRunWorkbenchURL(t *testing.T) {}
func TestArtifactWorkbenchJavaScriptUsesRawContentURL(t *testing.T) {}
func TestRunWorkbenchJavaScriptLinksLinkedArtifactsToArtifactWorkbench(t *testing.T) {}
func TestRunWorkbenchJavaScriptKeepsLegacyArtifactAnchorFallback(t *testing.T) {}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/adapters/http -run ArtifactWorkbench`
Expected: FAIL because the artifact workbench page and assets do not exist

- [ ] **Step 3: Implement the dedicated page and client**

UI rules:

- standalone page at `/board/artifacts/workbench?artifact_id=<artifact-id>`
- visible `Refresh artifact` control that re-fetches the current `artifact_id`
- left column shows same-run sibling artifacts only
- center column shows summary plus bounded preview or non-text fallback
- right column shows metadata and `Back to run workbench`
- use server-provided `run_workbench_url` and `raw_content_url`
- no edit / delete / compare / regenerate actions
- run workbench should deep-link linked artifacts into artifact workbench
- legacy unlinked artifact rows should retain compatibility behavior rather than broken deep links

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/adapters/http -run ArtifactWorkbench`
Expected: PASS

Run: `node --check web/board/artifact-workbench.js`
Expected: PASS

Run: `node --check web/board/run-workbench.js`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/board/artifact-workbench.html web/board/artifact-workbench.js web/board/run-workbench.js web/board/styles.css internal/adapters/http/board_handlers_test.go internal/adapters/http/router.go
git commit -m "feat: add artifact workbench ui"
```

## Task 5: Update Docs And Smoke Instructions

**Files:**
- Modify: `README.md`
- Modify: `INSTALL.md`
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Write the failing docs checklist**

Create a short checklist in your working notes:

```text
- README links the artifact workbench spec and plan
- README explains run workbench -> artifact workbench flow
- INSTALL includes artifact workbench and raw-content smoke
- CHANGELOG records the artifact workbench slice
```

- [ ] **Step 2: Verify the docs are incomplete**

Run: `rg -n "artifact workbench|/api/manager/artifacts/.*/workbench|/api/manager/artifacts/.*/content|/board/artifacts/workbench" README.md INSTALL.md CHANGELOG.md`
Expected: missing or incomplete matches

- [ ] **Step 3: Implement the docs update**

Add smoke guidance such as:

```bash
curl http://localhost:8080/api/manager/artifacts/<artifact-id>/workbench
curl http://localhost:8080/api/manager/artifacts/<artifact-id>/content
curl http://localhost:8080/board/artifacts/workbench?artifact_id=<artifact-id>
```

- [ ] **Step 4: Run verification**

Run: `rg -n "artifact workbench|/api/manager/artifacts/.*/workbench|/api/manager/artifacts/.*/content|/board/artifacts/workbench" README.md INSTALL.md CHANGELOG.md`
Expected: matches present where intended

- [ ] **Step 5: Commit**

```bash
git add README.md INSTALL.md CHANGELOG.md
git commit -m "docs: add artifact workbench guidance"
```

## Milestone Verification

Run these after all tasks complete:

```bash
go test ./internal/infrastructure/store/sqlite -run Artifact
go test ./internal/infrastructure/store/artifactfs -run Artifact
go test ./internal/app/query -run ArtifactWorkbench
go test ./internal/app/manageragent -run ArtifactWorkbench
go test ./internal/adapters/http -run ArtifactWorkbench
go test ./internal/bootstrap -run ArtifactWorkbench
go test ./...
node --check web/board/artifact-workbench.js
node --check web/board/run-workbench.js
```

Manual smoke:

```bash
go run ./cmd/foreman serve
curl http://localhost:8080/api/manager/artifacts/<artifact-id>/workbench
curl -D - http://localhost:8080/api/manager/artifacts/<artifact-id>/content -o /tmp/foreman-artifact.out
curl http://localhost:8080/board/artifacts/workbench?artifact_id=<artifact-id>
```
