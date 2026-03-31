# Foreman Phase 2 Artifact Compare History Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend the existing artifact compare page so operators can choose from a bounded recent-history list and deep-link a selected historical compare target through `previous_artifact_id`.

**Architecture:** Keep compare on the current compare route and API. First extend the compare repository/query path so it can return a bounded recent-history list and validate an optional `previous_artifact_id` against that list. Then plumb the optional compare target through manageragent, HTTP, and bootstrap, add right-column history selection to the compare page with URL-driven refetch, and finish with docs and smoke guidance.

**Tech Stack:** Go, SQLite-backed board query repository, existing `artifactfs` bounded reads, existing manager-agent service, `gin`, existing `web/board` assets, `node --test`

---

## Scope Check

This plan intentionally covers only the next Phase 2 sub-project:

- optional `previous_artifact_id` support on the existing compare API and page
- bounded recent-history list of at most five artifacts
- server-side validation that explicit history targets belong to that bounded set
- compare-page history list rendering and URL-driven history selection
- docs and smoke guidance updates

Explicitly out of scope for this plan:

- full artifact history pagination
- arbitrary historical compare outside the bounded list
- image/media compare
- dedicated history endpoint
- search, filtering, or sorting controls
- manual artifact-id input

Follow-on Phase 2 plans should cover:

- richer history browsing beyond five items
- compare-side filtering and controls
- image/media compare

## File Structure

### Compare repository/query history support

- Modify: `internal/ports/repositories.go`
  Responsibility change: extend compare row types with bounded history items and support an explicit selected historical artifact without changing workbench contracts.
- Modify: `internal/infrastructure/store/sqlite/board_query_repo.go`
  Responsibility change: return bounded recent-history items, validate optional `previous_artifact_id` against the bounded set, and keep deterministic ordering rules centralized in SQLite.
- Modify: `internal/infrastructure/store/sqlite/board_query_repo_test.go`
  Responsibility change: verify history window size/order, default selected previous artifact, valid explicit selection, and invalid-selection rejection.
- Modify: `internal/app/query/artifact_compare.go`
  Responsibility change: accept optional `previousArtifactID`, expose `history[]`, keep business states unchanged, and turn invalid selected-history targets into a client error.
- Modify: `internal/app/query/artifact_compare_test.go`
  Responsibility change: verify `history[]`, selected history behavior, default-vs-explicit target selection, and invalid explicit target handling.
- Modify: `internal/app/query/board_query_test.go`
  Responsibility change: keep `ports.BoardQueryRepository` test doubles compiling after compare method signature/row changes.
- Modify: `internal/app/query/approval_workbench_test.go`
  Responsibility change: keep existing compare-related fake repo implementations compiling after compare row changes.

### Manager service, HTTP, and bootstrap parameter plumbing

- Modify: `internal/app/manageragent/types.go`
  Responsibility change: expose compare history item types and any compare-specific client error needed for invalid `previous_artifact_id`.
- Modify: `internal/app/manageragent/service.go`
  Responsibility change: accept optional `previousArtifactID` for compare, normalize invalid-selection errors to a client-facing compare error, and keep `404` / `500` behavior unchanged.
- Modify: `internal/app/manageragent/service_test.go`
  Responsibility change: cover explicit compare-target selection and invalid explicit target behavior.
- Modify: `internal/adapters/http/dto.go`
  Responsibility change: add compare `history[]` DTO items and preserve nullable/array semantics.
- Modify: `internal/adapters/http/manager_handlers.go`
  Responsibility change: pass `previous_artifact_id` through to compare and map invalid explicit target errors to `400`.
- Modify: `internal/adapters/http/manager_handlers_test.go`
  Responsibility change: verify `history[]`, explicit target selection, and `400` behavior for invalid `previous_artifact_id`.
- Modify: `internal/bootstrap/app.go`
  Responsibility change: plumb optional selected history target into the compare manager path without adding a new endpoint.
- Modify: `internal/bootstrap/app_test.go`
  Responsibility change: verify live compare requests with and without `previous_artifact_id`.

### Compare page selectable history

- Modify: `web/board/artifact-compare.js`
  Responsibility change: render the recent-history list, navigate via compare URLs, and stay URL-driven using `artifact_id` plus optional `previous_artifact_id`.
- Modify: `web/board/artifact-compare.html`
  Responsibility change: reserve stable right-column structure for selected-history metadata plus recent-history list without adding editable inputs.
- Modify: `web/board/artifact-compare.test.js`
  Responsibility change: cover history rendering, selected-item highlighting, URL-driven target changes, and popstate behavior with explicit selected-history targets.
- Modify: `web/board/styles.css`
  Responsibility change: style the recent-history list and selected-history state inside the existing compare layout.
- Modify: `internal/adapters/http/board_handlers_test.go`
  Responsibility change: verify compare page assets still expose the compare route shape and history-selection strings.

### Documentation

- Modify: `README.md`
  Responsibility change: explain bounded recent-history selection on the compare page.
- Modify: `INSTALL.md`
  Responsibility change: add compare-history smoke commands for `previous_artifact_id`.
- Modify: `CHANGELOG.md`
  Responsibility change: record the compare-history slice.

## Runtime Path and Ownership

The existing runtime path this plan extends is:

- `internal/infrastructure/store/sqlite/board_query_repo.go`
- `internal/app/query/artifact_compare.go`
- `internal/app/manageragent/service.go`
- `internal/adapters/http/manager_handlers.go`
- `internal/adapters/http/dto.go`
- `internal/bootstrap/app.go`
- `web/board/artifact-compare.js`

Primary ownership by package:

- `internal/infrastructure/store/sqlite`: bounded recent-history selection and validation
- `internal/app/query`: compare history DTO, selected-target behavior, and compare-state assembly
- `internal/app/manageragent`: compare target exposure and invalid-selection normalization
- `internal/adapters/http`: compare query-param transport and HTTP status mapping
- `web/board`: history list rendering and URL-driven selection

## Task 1: Extend Compare Repository And Query With Bounded History

**Files:**
- Modify: `internal/ports/repositories.go`
- Modify: `internal/infrastructure/store/sqlite/board_query_repo.go`
- Modify: `internal/infrastructure/store/sqlite/board_query_repo_test.go`
- Modify: `internal/app/query/artifact_compare.go`
- Modify: `internal/app/query/artifact_compare_test.go`
- Modify: `internal/app/query/board_query_test.go`
- Modify: `internal/app/query/approval_workbench_test.go`

- [ ] **Step 1: Write the failing repository and query tests**

```go
func TestArtifactCompareBoardQueryRepositoryReturnsRecentHistoryWindow(t *testing.T) {}
func TestArtifactCompareBoardQueryRepositorySelectsExplicitPreviousArtifactWithinHistoryWindow(t *testing.T) {}
func TestArtifactCompareBoardQueryRepositoryRejectsExplicitPreviousArtifactOutsideHistoryWindow(t *testing.T) {}
func TestArtifactCompareBoardQueryRepositoryReturnsHistoryItemSummaryFields(t *testing.T) {}
func TestArtifactCompareReturnsHistoryItemsWithSelectedCompareURL(t *testing.T) {}
func TestArtifactCompareUsesExplicitPreviousArtifactWhenProvided(t *testing.T) {}
func TestArtifactCompareReturnsClientErrorForInvalidExplicitPreviousArtifact(t *testing.T) {}
func TestArtifactCompareKeepsHistoryArrayForNoPreviousState(t *testing.T) {}
func TestArtifactCompareKeepsHistoryArrayForUnsupportedState(t *testing.T) {}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/infrastructure/store/sqlite -run ArtifactCompare`
Expected: FAIL because the repository does not yet return bounded history or validate explicit selected targets

Run: `go test ./internal/app/query -run ArtifactCompare`
Expected: FAIL because the query does not yet accept explicit selected history or return `history[]`

- [ ] **Step 3: Implement the minimal history-aware compare query path**

Rules:

- keep the current compare API shape conceptually intact: one current artifact, one selected previous artifact, one compare response
- add a bounded recent-history list of at most five items
- each history item must include:
  - `artifact_id`
  - `run_id`
  - `created_at`
  - `summary`
  - `selected`
  - `compare_url`
- history selection remains deterministic:
  - same `task_id`
  - same `kind`
  - earlier than the current artifact
  - ordered by `created_at desc, artifact_id desc`
- explicit `previous_artifact_id` is valid only if it is a member of that bounded recent-history list
- invalid explicit `previous_artifact_id` must surface as a client error, not a compare business state
- `history` must always be present as an array in successful compare responses
- `history[].compare_url` must include both:
  - `artifact_id`
  - `previous_artifact_id`
- existing business states remain unchanged:
  - `ready`
  - `no_previous`
  - `unsupported`
  - `too_large`
- update fake `BoardQueryRepository` implementations so packages continue compiling

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/infrastructure/store/sqlite -run ArtifactCompare`
Expected: PASS

Run: `go test ./internal/app/query -run ArtifactCompare`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ports/repositories.go internal/infrastructure/store/sqlite/board_query_repo.go internal/infrastructure/store/sqlite/board_query_repo_test.go internal/app/query/artifact_compare.go internal/app/query/artifact_compare_test.go internal/app/query/board_query_test.go internal/app/query/approval_workbench_test.go
git commit -m "feat: add artifact compare history query"
```

## Task 2: Plumb `previous_artifact_id` Through Manager Service, HTTP, And Bootstrap

**Files:**
- Modify: `internal/app/manageragent/types.go`
- Modify: `internal/app/manageragent/service.go`
- Modify: `internal/app/manageragent/service_test.go`
- Modify: `internal/adapters/http/dto.go`
- Modify: `internal/adapters/http/manager_handlers.go`
- Modify: `internal/adapters/http/manager_handlers_test.go`
- Modify: `internal/bootstrap/app.go`
- Modify: `internal/bootstrap/app_test.go`

- [ ] **Step 1: Write the failing service, HTTP, and live-wiring tests**

```go
func TestManagerArtifactCompareReturnsHistoryItems(t *testing.T) {}
func TestManagerArtifactCompareUsesExplicitPreviousArtifactWhenProvided(t *testing.T) {}
func TestManagerArtifactCompareMapsInvalidPreviousArtifactTo400(t *testing.T) {}
func TestManagerArtifactCompareKeepsHistoryArrayForUnsupportedState(t *testing.T) {}
func TestServeArtifactCompareSupportsExplicitPreviousArtifactQueryParam(t *testing.T) {}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/app/manageragent -run ArtifactCompare`
Expected: FAIL because manageragent does not yet accept explicit compare targets or expose `history[]`

Run: `go test ./internal/adapters/http -run ArtifactCompare`
Expected: FAIL because the handler does not yet pass `previous_artifact_id` or map invalid selection to `400`

Run: `go test ./internal/bootstrap -run ArtifactCompare`
Expected: FAIL because live compare requests do not yet support optional explicit selected history targets

- [ ] **Step 3: Implement the minimal service and HTTP plumbing**

Rules:

- keep the compare endpoint:
  - `GET /api/manager/artifacts/:id/compare`
- accept optional query param:
  - `previous_artifact_id`
- preserve existing transport semantics:
  - `404` for missing current artifact
  - `500` for broken linkage
- add `400` for invalid explicit `previous_artifact_id`
- keep `history` as a stable array in successful compare responses
- keep per-item history fields aligned with the spec:
  - `artifact_id`
  - `run_id`
  - `created_at`
  - `summary`
  - `selected`
  - `compare_url`
- preserve nullable `previous` and `diff`
- do not add new write actions or a new history endpoint

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/app/manageragent -run ArtifactCompare`
Expected: PASS

Run: `go test ./internal/adapters/http -run ArtifactCompare`
Expected: PASS

Run: `go test ./internal/bootstrap -run ArtifactCompare`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/manageragent/types.go internal/app/manageragent/service.go internal/app/manageragent/service_test.go internal/adapters/http/dto.go internal/adapters/http/manager_handlers.go internal/adapters/http/manager_handlers_test.go internal/bootstrap/app.go internal/bootstrap/app_test.go
git commit -m "feat: plumb artifact compare history selection"
```

## Task 3: Add Selectable History To The Compare Page

**Files:**
- Modify: `web/board/artifact-compare.js`
- Modify: `web/board/artifact-compare.html`
- Modify: `web/board/artifact-compare.test.js`
- Modify: `web/board/styles.css`
- Modify: `internal/adapters/http/board_handlers_test.go`

- [ ] **Step 1: Write the failing board asset and Node tests**

```go
func TestArtifactComparePageServesHistorySelectionCopy(t *testing.T) {}
func TestArtifactCompareJavaScriptUsesPreviousArtifactIDURLState(t *testing.T) {}
```

Add executable Node tests for:

- rendering a recent-history list with up to five items
- highlighting the selected history item
- default compare target when `previous_artifact_id` is absent
- switching to a different compare target via `history[].compare_url`
- preserving `artifact_id` while changing `previous_artifact_id`
- handling `400` invalid-selection errors without introducing compare-side write actions or editable artifact-id inputs

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/adapters/http -run ArtifactCompare`
Expected: FAIL because the compare page assets do not yet expose history-selection strings

Run: `node --test web/board/artifact-compare.test.js`
Expected: FAIL because the compare page does not yet render history or selected-target URL transitions

- [ ] **Step 3: Implement the minimal compare-history UI**

Rules:

- keep the compare page read-only
- do not reintroduce editable artifact-id inputs
- keep `artifact_id` as the primary page identity
- allow optional `previous_artifact_id` in the URL
- use server-provided `history[].compare_url` for selection instead of inventing client-side compare URLs
- keep the three-column layout, but expand the right column into:
  - selected history artifact metadata
  - recent-history list
- do not add pagination, filters, or manual artifact-id entry

- [ ] **Step 4: Run tests to verify they pass**

Run: `node --check web/board/artifact-compare.js`
Expected: PASS

Run: `node --test web/board/artifact-compare.test.js`
Expected: PASS

Run: `go test ./internal/adapters/http -run ArtifactCompare`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/board/artifact-compare.js web/board/artifact-compare.html web/board/artifact-compare.test.js web/board/styles.css internal/adapters/http/board_handlers_test.go
git commit -m "feat: add compare history selection"
```

## Task 4: Update Docs And Smoke Guidance

**Files:**
- Modify: `README.md`
- Modify: `INSTALL.md`
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Write the failing docs checklist**

Create a short checklist in your working notes:

```text
- README explains bounded compare history selection and optional previous_artifact_id
- INSTALL includes compare-history smoke commands
- CHANGELOG records the compare-history slice
```

- [ ] **Step 2: Verify the docs are incomplete**

Run: `rg -n "previous_artifact_id|compare history|recent history|history\\[\\]|bounded history" README.md INSTALL.md CHANGELOG.md`
Expected: missing or incomplete matches

- [ ] **Step 3: Implement the docs update**

Rules:

- document the bounded five-item history window
- document optional `previous_artifact_id`
- document `400` semantics for invalid explicit history targets
- keep compare-history guidance clearly separate from image/media compare or full history browsing

- [ ] **Step 4: Run verification**

Run: `rg -n "previous_artifact_id|compare history|recent history|history\\[\\]|bounded history" README.md INSTALL.md CHANGELOG.md`
Expected: matches present where intended

- [ ] **Step 5: Commit**

```bash
git add README.md INSTALL.md CHANGELOG.md
git commit -m "docs: add compare history guidance"
```

## Milestone Verification

Run these after all tasks complete:

```bash
go test ./internal/infrastructure/store/sqlite -run ArtifactCompare
go test ./internal/app/query -run ArtifactCompare
go test ./internal/app/manageragent -run ArtifactCompare
go test ./internal/adapters/http -run ArtifactCompare
go test ./internal/bootstrap -run ArtifactCompare
go test ./...
node --check web/board/artifact-compare.js
node --test web/board/artifact-compare.test.js
```

Manual smoke:

- identify a current text artifact with at least two earlier same-task same-kind artifacts
- open `/board/artifacts/compare?artifact_id=<current-artifact-id>`
- verify the page shows a recent-history list of at most five items
- click a non-default history item and confirm the URL becomes:
  - `/board/artifacts/compare?artifact_id=<current-artifact-id>&previous_artifact_id=<selected-artifact-id>`
- verify refresh and browser back/forward keep the selected compare target
- verify an invalid `previous_artifact_id` produces a client error instead of silently falling back
