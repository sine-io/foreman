# Foreman Phase 2 Artifact Log Ergonomics Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Improve troubleshooting readability for long text and log-like artifacts inside the existing artifact workbench by adding line numbers, a default collapsed first-screen teaser, expand-all behavior, and lightweight summary navigation while preserving current route/API shape.

**Architecture:** Keep all behavior inside the existing artifact workbench and keep the manager artifact API unchanged. Add browser-side helpers that apply only to the generic long-text path after structured renderer selection has already decided not to use JSON/Markdown/diff rendering, then integrate those helpers into the current preview area and finish with docs/smoke guidance updates.

**Tech Stack:** Existing `web/board` static assets, browser-side JavaScript, `node --test`, Go HTTP asset tests

---

## Scope Check

This plan intentionally covers only the next Phase 2 sub-project:

- line numbers for long text / log-like artifact previews
- default collapsed first-screen teaser for long text / log-like artifact previews
- explicit expand-all behavior for bounded previews
- lightweight summary navigation derived from existing summary / preview text
- docs and smoke guidance updates

Explicitly out of scope for this plan:

- new routes
- new manager API fields
- search / filtering
- syntax-aware log analysis
- cross-artifact comparison
- image or binary rendering

Follow-on Phase 2 plans should cover:

- full log search/filter/highlight
- richer binary/media preview
- compare-oriented artifact experiences

## File Structure

### Long-text ergonomics helpers

- Create: `web/board/artifact-log-ergonomics.js`
  Responsibility: expose a stable global helper API for long-text detection, line-number rendering, collapsed teaser slicing, expand-all state transitions, and lightweight summary-anchor extraction.
- Create: `web/board/artifact-log-ergonomics.test.js`
  Responsibility: execute helper behavior under `node --test`, covering line numbering, collapse/expand, summary-anchor extraction, structured-renderer precedence, and fallback behavior.
- Modify: `internal/adapters/http/board_handlers_test.go`
  Responsibility change: verify the new helper asset is served and that the expected long-text ergonomics logic is present.

### Artifact workbench integration

- Modify: `web/board/artifact-workbench.html`
  Responsibility change: load the new log-ergonomics helper before the existing workbench script.
- Modify: `web/board/artifact-workbench.js`
  Responsibility change: integrate long-text ergonomics into the generic preview path only, preserve structured-renderer precedence, and expose a pure preview-composition helper that can be executed under Node tests.
- Create: `web/board/artifact-workbench-log.test.js`
  Responsibility: execute page-level long-text preview behavior under `node --test`, including collapse reset on sibling change, expand-all behavior, and truncated preview warning persistence.
- Modify: `web/board/styles.css`
  Responsibility change: add styles for line numbers, collapsed teaser presentation, expand controls, and summary-anchor navigation without changing the page route or core layout.
- Modify: `internal/adapters/http/board_handlers_test.go`
  Responsibility change: verify the artifact workbench page loads the helper asset and keeps the expected long-text ergonomics integration strings.

### Documentation

- Modify: `README.md`
  Responsibility change: mention artifact log ergonomics for long text / log-like artifacts.
- Modify: `INSTALL.md`
  Responsibility change: add an optional browser-only smoke path for long text/log ergonomics.
- Modify: `CHANGELOG.md`
  Responsibility change: record the log-ergonomics slice.

## Runtime Path and Ownership

The existing runtime path this plan extends is:

- `web/board/artifact-workbench.html`
- `web/board/artifact-workbench.js`
- `web/board/styles.css`
- `internal/adapters/http/board_handlers_test.go`

Primary ownership by package:

- `web/board`: long-text ergonomics helpers, workbench integration, and styling
- `internal/adapters/http`: board asset coverage

## Task 1: Add Long-Text Ergonomics Helpers

**Files:**
- Create: `web/board/artifact-log-ergonomics.js`
- Create: `web/board/artifact-log-ergonomics.test.js`
- Modify: `internal/adapters/http/board_handlers_test.go`

- [ ] **Step 1: Write the failing asset and Node tests**

```go
func TestArtifactLogErgonomicsAssetServes(t *testing.T) {}
func TestArtifactLogErgonomicsIncludesLineNumbers(t *testing.T) {}
func TestArtifactLogErgonomicsIncludesCollapsedTeaserLogic(t *testing.T) {}
func TestArtifactLogErgonomicsIncludesSummaryAnchorExtraction(t *testing.T) {}
func TestArtifactLogErgonomicsHonorsStructuredRendererPrecedence(t *testing.T) {}
```

Add executable Node tests for:

- long-text detection
- line-number rendering
- collapsed teaser slicing
- expand-all state transition helpers
- summary-anchor extraction from existing summary / preview
- structured-renderer precedence (JSON/Markdown/diff success paths should not go through log ergonomics)
- structured artifacts that already fell back to generic text becoming eligible for log ergonomics

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/adapters/http -run ArtifactLogErgonomics`
Expected: FAIL because the helper asset and assertions do not exist yet

Run: `node --test web/board/artifact-log-ergonomics.test.js`
Expected: FAIL because the helper API does not exist yet

- [ ] **Step 3: Implement the minimal helper API**

Rules:

- expose a stable global API such as `window.ForemanArtifactLogErgonomics`
- the helper API must be pure and non-throwing from the caller’s perspective
- structured-renderer precedence runs first:
  - JSON / Markdown / diff success paths must not use log ergonomics
  - generic long-text ergonomics apply only to the generic text path, including structured artifacts that already fell back
- long-text detection applies only to text-like artifacts long enough to need ergonomics
- helpers must support:
  - line numbers
  - collapsed first-screen teaser
  - expand-all
  - summary-anchor extraction from current `summary` plus bounded `preview`
- summary anchors may be derived from the current `summary` plus the currently available bounded preview text, but must not imply access to content beyond that bounded preview
- if heuristics produce nothing useful, the helper must return an empty navigation result without failing the page

- [ ] **Step 4: Run tests to verify they pass**

Run: `node --check web/board/artifact-log-ergonomics.js`
Expected: PASS

Run: `node --test web/board/artifact-log-ergonomics.test.js`
Expected: PASS

Run: `go test ./internal/adapters/http -run ArtifactLogErgonomics`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/board/artifact-log-ergonomics.js web/board/artifact-log-ergonomics.test.js internal/adapters/http/board_handlers_test.go
git commit -m "feat: add artifact log ergonomics helpers"
```

## Task 2: Integrate Long-Text Ergonomics Into Artifact Workbench

**Files:**
- Modify: `web/board/artifact-workbench.html`
- Modify: `web/board/artifact-workbench.js`
- Create: `web/board/artifact-workbench-log.test.js`
- Modify: `web/board/styles.css`
- Modify: `internal/adapters/http/board_handlers_test.go`

- [ ] **Step 1: Write the failing artifact workbench asset and Node tests**

```go
func TestArtifactWorkbenchHTMLLoadsLogErgonomicsHelpers(t *testing.T) {}
func TestArtifactWorkbenchJavaScriptUsesLineNumberRenderingForLongText(t *testing.T) {}
func TestArtifactWorkbenchJavaScriptUsesCollapsedTeaserForLongText(t *testing.T) {}
func TestArtifactWorkbenchJavaScriptShowsExpandAllControl(t *testing.T) {}
func TestArtifactWorkbenchJavaScriptShowsSummaryNavigationForLongText(t *testing.T) {}
func TestArtifactWorkbenchJavaScriptResetsCollapsedStateOnSiblingChange(t *testing.T) {}
func TestArtifactWorkbenchJavaScriptKeepsStructuredRendererPrecedence(t *testing.T) {}
func TestArtifactWorkbenchJavaScriptKeepsTruncationWarningVisibleWhenExpanded(t *testing.T) {}
```

Add executable Node tests for:

- collapsed teaser vs expanded state
- expand-all behavior
- line-number output
- summary-anchor rendering
- sibling/artifact switch resetting to collapsed
- structured-renderer precedence staying intact
- generic long-text fallback behavior
- truncation warning remaining visible in both collapsed and expanded states

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/adapters/http -run ArtifactWorkbench`
Expected: FAIL because the page does not yet load or use the log-ergonomics helper

Run: `node --test web/board/artifact-workbench-log.test.js`
Expected: FAIL because the page-level long-text ergonomics helper behavior does not exist yet

- [ ] **Step 3: Implement the long-text ergonomics UI**

Rules:

- keep the current artifact workbench route and page shape
- load `artifact-log-ergonomics.js` before `artifact-workbench.js`
- integrate ergonomics only into the generic long-text path
- use the existing `ForemanArtifactRenderers.renderPreview()` result contract when deciding whether the current artifact is on a structured-renderer success path or a generic/fallback path
- structured-renderer success paths remain untouched even when their `output` is still `"text"` (for example pretty-printed JSON)
- collapsed mode means:
  - clipped first-screen teaser
  - no internal scrolling as a substitute for expansion
  - explicit `Expand all` to reveal the full bounded preview
- selecting a different sibling/artifact resets that artifact to collapsed mode by default
- summary navigation may auto-expand if needed to reveal a target
- truncation warning must remain visible in both collapsed and expanded states
- if log ergonomics fail, fall back to the current generic text rendering
- no new route or API field is introduced

- [ ] **Step 4: Run tests to verify they pass**

Run: `node --check web/board/artifact-log-ergonomics.js`
Expected: PASS

Run: `node --check web/board/artifact-workbench.js`
Expected: PASS

Run: `node --test web/board/artifact-workbench-log.test.js`
Expected: PASS

Run: `go test ./internal/adapters/http -run ArtifactWorkbench`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/board/artifact-workbench.html web/board/artifact-workbench.js web/board/artifact-workbench-log.test.js web/board/styles.css internal/adapters/http/board_handlers_test.go
git commit -m "feat: add artifact log ergonomics"
```

## Task 3: Update Docs And Smoke Guidance

**Files:**
- Modify: `README.md`
- Modify: `INSTALL.md`
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Write the failing docs checklist**

Create a short checklist in your working notes:

```text
- README mentions line numbers / collapse / summary navigation for long text artifacts
- INSTALL includes an optional browser-only long-text ergonomics smoke path
- CHANGELOG records the log-ergonomics slice
```

- [ ] **Step 2: Verify the docs are incomplete**

Run: `rg -n "line numbers|Expand all|summary navigation|log ergonomics|long text" README.md INSTALL.md CHANGELOG.md`
Expected: missing or incomplete matches

- [ ] **Step 3: Implement the docs update**

Rules:

- document that log ergonomics stay inside the existing artifact workbench
- document that the smoke is optional/browser-only and assumes a suitable long text/log-like artifact exists
- document that structured-renderer success paths remain separate from log ergonomics
- document that unsupported or short content continues to use the simpler preview path

- [ ] **Step 4: Run verification**

Run: `rg -n "line numbers|Expand all|summary navigation|log ergonomics|long text" README.md INSTALL.md CHANGELOG.md`
Expected: matches present where intended

- [ ] **Step 5: Commit**

```bash
git add README.md INSTALL.md CHANGELOG.md
git commit -m "docs: add artifact log ergonomics guidance"
```

## Milestone Verification

Run these after all tasks complete:

```bash
go test ./internal/adapters/http -run 'ArtifactLogErgonomics|ArtifactWorkbench'
go test ./...
node --check web/board/artifact-log-ergonomics.js
node --check web/board/artifact-workbench.js
node --check web/board/artifact-renderers.js
node --test web/board/artifact-renderers.test.js
node --test web/board/artifact-workbench.test.js
node --test web/board/artifact-log-ergonomics.test.js
node --test web/board/artifact-workbench-log.test.js
```

Manual smoke:

- use an existing long text/log-like artifact such as `run_log`, `command_result`, or a long `text/plain` artifact
- open `/board/artifacts/workbench?artifact_id=<artifact-id>` in a browser
- verify line numbers, collapsed first-screen teaser, explicit expand-all, and summary navigation behavior
