# Foreman Phase 2 Artifact Compare Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a minimal read-only artifact compare slice that lets operators open a dedicated compare page for one artifact and see a unified diff against the immediately previous same-task same-kind text artifact.

**Architecture:** Keep compare rules on the server. First extend the existing board query path with deterministic previous-artifact lookup and a dedicated artifact-compare read model that produces the four stable business states from the spec. Then expose that read model through manageragent, add one compare endpoint and board page, wire an artifact-workbench navigation action into it, and finish with docs and smoke guidance.

**Tech Stack:** Go, SQLite-backed board query repository, existing `artifactfs` bounded reads, existing manager-agent service, `gin`, existing `web/board` assets, `node --test`

---

## Scope Check

This plan intentionally covers only the next Phase 2 sub-project:

- dedicated artifact compare query and DTO
- deterministic previous-artifact selection by `task_id`, `kind`, `created_at desc`, `artifact_id desc`
- manager compare endpoint
- dedicated board compare page
- artifact-workbench navigation into compare
- docs and smoke guidance

Explicitly out of scope for this plan:

- manual history selection
- image or binary compare
- arbitrary artifact-to-artifact compare
- search, filtering, folding, or diff controls
- compare-triggered write actions
- new storage backends or schema changes

Follow-on Phase 2 plans should cover:

- selectable artifact history
- image/media compare
- richer compare controls

## File Structure

### Compare query and deterministic previous-artifact selection

- Modify: `internal/ports/repositories.go`
  Responsibility change: add compare-specific row shapes and repository methods without changing artifact workbench responsibilities.
- Modify: `internal/infrastructure/store/sqlite/board_query_repo.go`
  Responsibility change: add deterministic lookup for current artifact + previous same-task same-kind artifact and expose compare base rows from SQLite.
- Create: `internal/infrastructure/store/sqlite/board_query_repo_test.go`
  Responsibility: verify deterministic previous-artifact selection and no-previous behavior at the SQLite repository layer.
- Create: `internal/app/query/artifact_compare.go`
  Responsibility: build the compare DTO, enforce `ready` / `no_previous` / `unsupported` / `too_large`, read bounded text content from `ArtifactStore`, and generate server-side unified diff text.
- Create: `internal/app/query/artifact_compare_test.go`
  Responsibility: verify deterministic previous-artifact selection, business states, diff generation, navigation URLs, and too-large behavior.
- Modify: `internal/app/query/board_query_test.go`
  Responsibility change: keep `ports.BoardQueryRepository` test doubles compiling after the compare repository method is added.
- Modify: `internal/app/query/approval_workbench_test.go`
  Responsibility change: keep existing approval-workbench test doubles compiling after the compare repository method is added.

### Manager service and HTTP surface

- Modify: `internal/app/manageragent/types.go`
  Responsibility change: expose artifact compare types to adapters.
- Modify: `internal/app/manageragent/service.go`
  Responsibility change: add read-only artifact compare service entrypoint and normalize compare-specific error semantics.
- Modify: `internal/app/manageragent/service_test.go`
  Responsibility change: cover artifact compare through manageragent.Service.
- Modify: `internal/adapters/http/dto.go`
  Responsibility change: add compare response DTOs if adapter-local projection is needed.
- Modify: `internal/adapters/http/manager_handlers.go`
  Responsibility change: add `GET /api/manager/artifacts/:id/compare` and map `404` / `500` correctly while leaving business states in the JSON payload.
- Modify: `internal/adapters/http/manager_handlers_test.go`
  Responsibility change: verify success, `404`, `500`, and compare-state payload semantics.
- Modify: `internal/adapters/http/router.go`
  Responsibility change: add `/board/artifacts/compare` and the manager compare endpoint.
- Modify: `internal/bootstrap/app.go`
  Responsibility change: wire the new compare query into the live runtime and manager service.
- Modify: `internal/bootstrap/app_test.go`
  Responsibility change: verify compare endpoint wiring through a live app instance.

### Compare page and artifact-workbench navigation

- Create: `web/board/artifact-compare.html`
  Responsibility: provide the dedicated compare page shell.
- Create: `web/board/artifact-compare.js`
  Responsibility: load compare detail by `artifact_id`, render the four compare states, show unified diff text, and expose navigation back to artifact and run workbenches.
- Create: `web/board/artifact-compare.test.js`
  Responsibility: cover compare-page state rendering and navigation behavior under `node --test`.
- Modify: `web/board/artifact-workbench.js`
  Responsibility change: add `Compare with previous` action using the current artifact as the only entry identity.
- Modify: `web/board/artifact-workbench.test.js`
  Responsibility change: verify compare navigation action appears with the expected compare URL.
- Modify: `web/board/styles.css`
  Responsibility change: add compare-page layout and unified-diff presentation while preserving the existing visual language.
- Modify: `internal/adapters/http/board_handlers_test.go`
  Responsibility change: verify compare page/asset routes and key client-state strings.

### Documentation

- Modify: `README.md`
  Responsibility change: explain the compare page as the next layer below artifact workbench.
- Modify: `INSTALL.md`
  Responsibility change: add compare smoke commands and optional browser flow.
- Modify: `CHANGELOG.md`
  Responsibility change: record the artifact compare slice.

## Runtime Path and Ownership

The existing runtime path this plan extends is:

- `internal/infrastructure/store/sqlite/board_query_repo.go`
- `internal/app/query/artifact_workbench.go`
- `internal/app/manageragent/service.go`
- `internal/adapters/http/manager_handlers.go`
- `internal/adapters/http/router.go`
- `internal/bootstrap/app.go`
- `web/board/artifact-workbench.js`

Primary ownership by package:

- `internal/infrastructure/store/sqlite`: deterministic previous-artifact selection
- `internal/app/query`: compare view assembly and diff generation
- `internal/app/manageragent`: compare exposure to adapters and normalized errors
- `internal/adapters/http`: compare endpoint and compare page route
- `web/board`: compare page rendering and artifact-workbench navigation

## Task 1: Add Compare Query Primitives And Deterministic Previous-Artifact Selection

**Files:**
- Modify: `internal/ports/repositories.go`
- Modify: `internal/infrastructure/store/sqlite/board_query_repo.go`
- Create: `internal/infrastructure/store/sqlite/board_query_repo_test.go`
- Create: `internal/app/query/artifact_compare.go`
- Create: `internal/app/query/artifact_compare_test.go`
- Modify: `internal/app/query/board_query_test.go`
- Modify: `internal/app/query/approval_workbench_test.go`

- [ ] **Step 1: Write the failing repository and query tests**

```go
func TestArtifactCompareBoardQueryRepositoryReturnsPreviousArtifactByCreatedAtAndArtifactID(t *testing.T) {}
func TestArtifactCompareBoardQueryRepositoryReturnsNoPreviousArtifactWhenCurrentIsFirst(t *testing.T) {}
func TestArtifactCompareReturnsReadyForPreviousSameTaskSameKindArtifact(t *testing.T) {}
func TestArtifactCompareUsesCreatedAtAndArtifactIDAsTieBreaker(t *testing.T) {}
func TestArtifactCompareReturnsNoPreviousWhenNoEarlierArtifactExists(t *testing.T) {}
func TestArtifactCompareReturnsUnsupportedForBinaryArtifactKinds(t *testing.T) {}
func TestArtifactCompareReturnsReadyForJSONArtifacts(t *testing.T) {}
func TestArtifactCompareReturnsTooLargeWhenEitherArtifactExceedsLimit(t *testing.T) {}
func TestArtifactCompareIncludesCurrentAndPreviousWorkbenchURLs(t *testing.T) {}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/app/query -run ArtifactCompare`
Expected: FAIL because the compare query does not exist yet

Run: `go test ./internal/infrastructure/store/sqlite -run ArtifactCompare`
Expected: FAIL because the board query repository does not expose compare lookup yet

- [ ] **Step 3: Implement the minimal compare query path**

Rules:

- previous-artifact selection must use:
  - same `task_id`
  - same `kind`
  - lower `created_at` or same `created_at` with lower stable `artifact_id`
  - final ordering by `created_at desc, artifact_id desc`
- keep compare read-only
- do not add schema changes
- update existing `ports.BoardQueryRepository` test doubles so the package still compiles after the new compare method is added
- compare becomes `ready` only when both current and previous artifacts resolve to supported text content
- use a bounded read limit for compare source content
- if either side exceeds the limit, return `too_large` and no diff
- `diff` must exist only for `ready`
- `messages` must be present for every business state

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/app/query -run ArtifactCompare`
Expected: PASS

Run: `go test ./internal/infrastructure/store/sqlite -run ArtifactCompare`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ports/repositories.go internal/infrastructure/store/sqlite/board_query_repo.go internal/infrastructure/store/sqlite/board_query_repo_test.go internal/app/query/artifact_compare.go internal/app/query/artifact_compare_test.go internal/app/query/board_query_test.go internal/app/query/approval_workbench_test.go
git commit -m "feat: add artifact compare query"
```

## Task 2: Expose Artifact Compare Through Manager Service, HTTP, And Bootstrap

**Files:**
- Modify: `internal/app/manageragent/types.go`
- Modify: `internal/app/manageragent/service.go`
- Modify: `internal/app/manageragent/service_test.go`
- Modify: `internal/adapters/http/dto.go`
- Modify: `internal/adapters/http/manager_handlers.go`
- Modify: `internal/adapters/http/manager_handlers_test.go`
- Modify: `internal/adapters/http/router.go`
- Modify: `internal/bootstrap/app.go`
- Modify: `internal/bootstrap/app_test.go`

- [ ] **Step 1: Write the failing service, HTTP, and wiring tests**

```go
func TestManagerArtifactCompareReturnsReadyView(t *testing.T) {}
func TestManagerArtifactCompareMapsMissingArtifactTo404(t *testing.T) {}
func TestManagerArtifactCompareMapsBrokenLinkageTo500(t *testing.T) {}
func TestServeArtifactCompareExposesLiveCompareRoute(t *testing.T) {}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/app/manageragent -run ArtifactCompare`
Expected: FAIL because manageragent.Service does not expose artifact compare yet

Run: `go test ./internal/adapters/http -run ArtifactCompare`
Expected: FAIL because the compare endpoint and route do not exist yet

Run: `go test ./internal/bootstrap -run ArtifactCompare`
Expected: FAIL because live runtime wiring does not expose compare yet

- [ ] **Step 3: Implement the minimal compare service and endpoint**

Rules:

- add `GET /api/manager/artifacts/:id/compare`
- return `404` when the current artifact does not exist
- return `500` when task/run linkage is broken
- keep `no_previous`, `unsupported`, and `too_large` inside successful JSON payloads
- do not add compare write actions
- keep the compare DTO aligned with the spec:
  - `current` always present
  - `previous` nullable
  - `diff` nullable except for `ready`
  - `limits.max_compare_bytes` present for all responses
  - `messages.title` and `messages.detail` present for all responses
  - navigation URLs stable
  - current and previous metadata include `artifact_id`, `run_id`, `task_id`, `kind`, `content_type`, and `created_at`

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/app/manageragent -run ArtifactCompare`
Expected: PASS

Run: `go test ./internal/adapters/http -run ArtifactCompare`
Expected: PASS

Run: `go test ./internal/bootstrap -run ArtifactCompare`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/manageragent/types.go internal/app/manageragent/service.go internal/app/manageragent/service_test.go internal/adapters/http/dto.go internal/adapters/http/manager_handlers.go internal/adapters/http/manager_handlers_test.go internal/adapters/http/router.go internal/bootstrap/app.go internal/bootstrap/app_test.go
git commit -m "feat: expose artifact compare endpoint"
```

## Task 3: Build The Compare Page And Artifact-Workbench Navigation

**Files:**
- Create: `web/board/artifact-compare.html`
- Create: `web/board/artifact-compare.js`
- Create: `web/board/artifact-compare.test.js`
- Modify: `web/board/artifact-workbench.js`
- Modify: `web/board/artifact-workbench.test.js`
- Modify: `web/board/styles.css`
- Modify: `internal/adapters/http/board_handlers_test.go`

- [ ] **Step 1: Write the failing board asset and Node tests**

```go
func TestArtifactComparePageServes(t *testing.T) {}
func TestArtifactCompareHTMLLoadsCompareAsset(t *testing.T) {}
func TestArtifactWorkbenchJavaScriptLinksToArtifactCompare(t *testing.T) {}
```

Add executable Node tests for:

- `ready` state rendering with diff content
- `no_previous` empty state
- `unsupported` state
- `too_large` state
- navigation links back to current artifact workbench and run workbench
- refresh behavior using the current `artifact_id`

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/adapters/http -run ArtifactCompare`
Expected: FAIL because the compare page route and asset assertions do not exist yet

Run: `node --test web/board/artifact-compare.test.js`
Expected: FAIL because the compare page implementation does not exist yet

Run: `node --test web/board/artifact-workbench.test.js`
Expected: FAIL because artifact workbench does not expose compare navigation yet

- [ ] **Step 3: Implement the minimal compare page**

Rules:

- add `/board/artifacts/compare?artifact_id=<artifact-id>`
- keep the page read-only
- use the current `artifact_id` as the only page identity
- render the four business states from the API without inventing new client-side states
- keep the three-part page structure from the spec:
  - current metadata panel
  - center compare result panel
  - previous metadata panel
- show unified diff text in the center when compare is `ready`
- keep compare-page navigation limited to:
  - `Back to current artifact`
  - `Back to run workbench`
  - `Refresh compare`
- add one `Compare with previous` action to artifact workbench
- do not add manual history selection or side-by-side raw content panes

- [ ] **Step 4: Run tests to verify they pass**

Run: `node --check web/board/artifact-compare.js`
Expected: PASS

Run: `node --test web/board/artifact-compare.test.js`
Expected: PASS

Run: `node --test web/board/artifact-workbench.test.js`
Expected: PASS

Run: `go test ./internal/adapters/http -run ArtifactCompare`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/board/artifact-compare.html web/board/artifact-compare.js web/board/artifact-compare.test.js web/board/artifact-workbench.js web/board/artifact-workbench.test.js web/board/styles.css internal/adapters/http/board_handlers_test.go
git commit -m "feat: add artifact compare workbench"
```

## Task 4: Update Documentation And Smoke Guidance

**Files:**
- Modify: `README.md`
- Modify: `INSTALL.md`
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Write the failing docs checklist**

Create a short checklist in your working notes:

```text
- README mentions the compare page as a read-only follow-on to artifact workbench
- INSTALL includes compare API and optional browser smoke steps
- CHANGELOG records the artifact compare slice
```

- [ ] **Step 2: Verify the docs are incomplete**

Run: `rg -n "artifact compare|compare with previous|artifacts/:id/compare|/board/artifacts/compare" README.md INSTALL.md CHANGELOG.md`
Expected: missing or incomplete matches

- [ ] **Step 3: Implement the docs update**

Rules:

- document compare as a read-only page below artifact workbench
- document the current-vs-previous selection rule in user-facing terms
- document the API and board route
- keep browser smoke optional and focused on existing comparable text artifacts
- do not imply image or binary compare support

- [ ] **Step 4: Run verification**

Run: `rg -n "artifact compare|compare with previous|artifacts/:id/compare|/board/artifacts/compare" README.md INSTALL.md CHANGELOG.md`
Expected: matches present where intended

- [ ] **Step 5: Commit**

```bash
git add README.md INSTALL.md CHANGELOG.md
git commit -m "docs: add artifact compare guidance"
```

## Milestone Verification

Run these after all tasks complete:

```bash
go test ./internal/app/query -run ArtifactCompare
go test ./internal/app/manageragent -run ArtifactCompare
go test ./internal/adapters/http -run ArtifactCompare
go test ./internal/bootstrap -run ArtifactCompare
go test ./...
node --check web/board/artifact-compare.js
node --test web/board/artifact-compare.test.js
node --test web/board/artifact-workbench.test.js
```

Manual smoke:

- create or identify a task with at least two text artifacts of the same `kind` across different runs
- open `/board/artifacts/workbench?artifact_id=<current-artifact-id>`
- follow `Compare with previous`
- verify:
  - compare page loads at `/board/artifacts/compare?artifact_id=<current-artifact-id>`
  - a unified diff appears for the previous same-task same-kind artifact
  - `Back to current artifact` and `Back to run workbench` work
  - when no previous comparable artifact exists, the page shows the stable empty state instead of failing
