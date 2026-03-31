# Foreman Phase 2 Artifact Renderer Polish Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Improve readability inside the existing artifact workbench by adding client-side JSON pretty-print, safe Markdown render, and diff/patch structure display while preserving generic text fallback and current API shape.

**Architecture:** Keep the server thin and leave the manager artifact API untouched except for consuming the existing `preview`, `content_type`, `kind`, `path`, and `preview_truncated` fields already returned today. Add a focused client-side renderer helper for structured text, integrate it into the artifact workbench preview block, and extend board asset tests plus docs without creating any new routes or nested pages.

**Tech Stack:** Go test suite for asset-string coverage, existing `web/board` static assets, browser-side JavaScript/CSS only, no new runtime dependencies

---

## Scope Check

This plan intentionally covers only the next Phase 2 sub-project:

- artifact workbench renderer polish inside the existing page
- JSON pretty-print
- safe Markdown render
- diff / patch structure display
- generic fallback preservation
- docs and smoke guidance updates

Explicitly out of scope for this plan:

- new artifact routes or new manager API endpoints
- image/binary rendering
- cross-run compare
- full log search/folding/filtering
- renderer plugin framework
- server-side HTML rendering

Follow-on Phase 2 plans should cover:

- richer renderer polish for logs
- image/binary/media preview
- compare-oriented artifact experiences

## File Structure

### Renderer helpers and artifact workbench integration

- Create: `web/board/artifact-renderers.js`
  Responsibility: hold the small client-side renderer helpers for JSON, Markdown, diff/patch detection, safe fallback behavior, and truncation-aware rendering decisions.
- Create: `web/board/artifact-renderers.test.js`
  Responsibility: execute renderer helper behavior under `node --test`, covering JSON/Markdown/diff success and fallback cases instead of relying only on string assertions.
- Modify: `web/board/artifact-workbench.html`
  Responsibility change: load the new renderer helper before the artifact workbench page script.
- Modify: `web/board/artifact-workbench.js`
  Responsibility change: replace the generic preview-only block with renderer-aware display that still respects `preview_truncated`, inert Markdown requirements, and generic fallback behavior.
- Create: `web/board/artifact-workbench.test.js`
  Responsibility: execute the page-level preview composition logic under `node --test`, including renderer-error fallback behavior.
- Modify: `web/board/styles.css`
  Responsibility change: add renderer-specific styles for formatted JSON, Markdown blocks, and diff sections while preserving the existing artifact workbench layout.

### Asset and board route verification

- Modify: `internal/adapters/http/board_handlers_test.go`
  Responsibility change: verify renderer asset serving and the presence of JSON/Markdown/diff logic, truncation handling, and inert Markdown safeguards.

### Documentation

- Modify: `README.md`
  Responsibility change: record that the artifact workbench now has structured text renderer polish.
- Modify: `INSTALL.md`
  Responsibility change: add a short smoke path for JSON / Markdown / diff artifact rendering.
- Modify: `CHANGELOG.md`
  Responsibility change: record the renderer polish slice.

## Runtime Path and Ownership

The existing runtime path this plan extends is:

- `web/board/artifact-workbench.html`
- `web/board/artifact-workbench.js`
- `web/board/styles.css`
- `internal/adapters/http/board_handlers_test.go`

Primary ownership by package:

- `web/board`: renderer helpers, artifact workbench preview rendering, and visual presentation
- `internal/adapters/http`: board asset coverage and regression tests

## Task 1: Add Structured Text Renderer Helpers

**Files:**
- Create: `web/board/artifact-renderers.js`
- Create: `web/board/artifact-renderers.test.js`
- Modify: `internal/adapters/http/board_handlers_test.go`

- [ ] **Step 1: Write the failing asset tests**

```go
func TestArtifactRendererHelpersAssetServes(t *testing.T) {}
func TestArtifactRendererHelpersIncludeJSONPrettyPrintPath(t *testing.T) {}
func TestArtifactRendererHelpersIncludeMarkdownRenderPath(t *testing.T) {}
func TestArtifactRendererHelpersIncludeDiffDetectionByKindOrPath(t *testing.T) {}
func TestArtifactRendererHelpersKeepMarkdownInert(t *testing.T) {}
func TestArtifactRendererHelpersRespectPreviewTruncation(t *testing.T) {}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/adapters/http -run ArtifactRenderer`
Expected: FAIL because the helper asset and renderer logic do not exist yet

Run: `node --test web/board/artifact-renderers.test.js`
Expected: FAIL because the renderer helper API does not exist yet

- [ ] **Step 3: Implement the minimal renderer helper asset**

Rules:

- expose a stable global API, e.g. `window.ForemanArtifactRenderers`, so Task 2 can integrate it cleanly without a module loader
- the helper API must be pure and non-throwing from the caller's perspective:
  - given artifact detail metadata + preview text, it returns a render result object
  - on parser/render failure, it returns a generic fallback result instead of throwing
- renderer selection may use `content_type`, artifact `kind`, and artifact `path`
- JSON renderer should pretty-print parsed preview text and fall back if parsing fails
- Markdown renderer must remain inert/text-only:
  - no remote image loading
  - no iframe/embed rendering
  - no raw HTML passthrough
- diff renderer may detect `.diff` / `.patch` by path suffix or kind even when `content_type` is generic text
- enhanced renderers must preserve visible truncation warnings and may fall back when truncated partial content would be misleading
- everything else must preserve generic text fallback

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/adapters/http -run ArtifactRenderer`
Expected: PASS

Run: `node --check web/board/artifact-renderers.js`
Expected: PASS

Run: `node --test web/board/artifact-renderers.test.js`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/board/artifact-renderers.js web/board/artifact-renderers.test.js internal/adapters/http/board_handlers_test.go
git commit -m "feat: add artifact renderer helpers"
```

## Task 2: Integrate Renderer Helpers Into Artifact Workbench

**Files:**
- Modify: `web/board/artifact-workbench.html`
- Modify: `web/board/artifact-workbench.js`
- Create: `web/board/artifact-workbench.test.js`
- Modify: `web/board/styles.css`
- Modify: `internal/adapters/http/board_handlers_test.go`

- [ ] **Step 1: Write the failing artifact workbench asset tests**

```go
func TestArtifactWorkbenchHTMLLoadsRendererHelpers(t *testing.T) {}
func TestArtifactWorkbenchJavaScriptUsesRendererHelpersForJSON(t *testing.T) {}
func TestArtifactWorkbenchJavaScriptUsesRendererHelpersForMarkdown(t *testing.T) {}
func TestArtifactWorkbenchJavaScriptUsesRendererHelpersForDiffArtifacts(t *testing.T) {}
func TestArtifactWorkbenchJavaScriptKeepsGenericFallback(t *testing.T) {}
func TestArtifactWorkbenchJavaScriptKeepsTruncationNoticeVisible(t *testing.T) {}
func TestArtifactWorkbenchJavaScriptKeepsPageUsableWhenRendererFallsBack(t *testing.T) {}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/adapters/http -run ArtifactWorkbench`
Expected: FAIL because the artifact workbench still renders only the generic preview block

Run: `node --test web/board/artifact-workbench.test.js`
Expected: FAIL because the page-level preview composition helper does not exist yet

- [ ] **Step 3: Implement the renderer-aware workbench UI**

Rules:

- keep the current artifact workbench route and page shape
- load `artifact-renderers.js` before `artifact-workbench.js`
- expose a small pure preview-composition helper from `artifact-workbench.js` so page-level renderer fallback can be executed under Node tests
- artifact workbench should use specialized rendering only inside the existing preview area
- JSON, Markdown, and diff/patch artifacts should render in improved forms
- unsupported or malformed cases must fall back to generic text preview
- empty text preview remains a valid preview, not a non-text fallback
- truncation warning stays visible even under enhanced renderers
- if a specialized renderer fails, the page must stay usable by showing the generic preview block instead of surfacing a page-level error
- no new page or nested route is introduced

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/adapters/http -run ArtifactWorkbench`
Expected: PASS

Run: `node --check web/board/artifact-renderers.js`
Expected: PASS

Run: `node --check web/board/artifact-workbench.js`
Expected: PASS

Run: `node --test web/board/artifact-workbench.test.js`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/board/artifact-workbench.html web/board/artifact-workbench.js web/board/artifact-workbench.test.js web/board/styles.css internal/adapters/http/board_handlers_test.go
git commit -m "feat: polish artifact renderers"
```

## Task 3: Update Docs And Smoke Guidance

**Files:**
- Modify: `README.md`
- Modify: `INSTALL.md`
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Write the failing docs checklist**

Create a short checklist in your working notes:

```text
- README mentions renderer polish for JSON / Markdown / diff
- INSTALL includes a renderer-polish smoke path
- CHANGELOG records the renderer polish slice
```

- [ ] **Step 2: Verify the docs are incomplete**

Run: `rg -n "renderer polish|JSON pretty-print|Markdown render|diff / patch|structured text" README.md INSTALL.md CHANGELOG.md`
Expected: missing or incomplete matches

- [ ] **Step 3: Implement the docs update**

Add smoke guidance such as:

```bash
curl http://localhost:8080/api/manager/artifacts/<artifact-id>/workbench
```

and note that the smoke artifact should be one whose `content_type`, `kind`, or `path` maps to:

- JSON
- Markdown
- diff / patch

- [ ] **Step 4: Run verification**

Run: `rg -n "renderer polish|JSON pretty-print|Markdown render|diff / patch|structured text" README.md INSTALL.md CHANGELOG.md`
Expected: matches present where intended

- [ ] **Step 5: Commit**

```bash
git add README.md INSTALL.md CHANGELOG.md
git commit -m "docs: add artifact renderer guidance"
```

## Milestone Verification

Run these after all tasks complete:

```bash
go test ./internal/adapters/http -run 'ArtifactRenderer|ArtifactWorkbench'
go test ./...
node --check web/board/artifact-renderers.js
node --check web/board/artifact-workbench.js
node --test web/board/artifact-renderers.test.js
node --test web/board/artifact-workbench.test.js
```

Manual smoke:

```bash
go run ./cmd/foreman serve
curl http://localhost:8080/api/manager/artifacts/<artifact-id>/workbench
```

Then open:

```text
/board/artifacts/workbench?artifact_id=<artifact-id>
```

and verify:

- JSON artifacts pretty-print
- Markdown artifacts render safely as inert formatted content
- diff / patch artifacts use structured diff-oriented display
- malformed or unsupported content falls back to generic text preview
- truncation notice remains visible when `preview_truncated=true`
