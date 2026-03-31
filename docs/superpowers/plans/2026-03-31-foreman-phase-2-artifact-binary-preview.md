# Foreman Phase 2 Artifact Binary/Media Preview Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend the existing artifact workbench so approved image artifacts preview inline while non-image binary artifacts stay on the metadata/download path, all without introducing new routes or new manager API shapes.

**Architecture:** Keep the current artifact workbench detail endpoint and raw content endpoint as the only server surfaces. First adjust the existing raw-content safety contract so approved raster image types are inline-previewable and SVG remains best-effort under current safety policy, then add browser-side preview selection inside the existing artifact workbench, and finally update docs/smoke guidance.

**Tech Stack:** Existing Go HTTP handlers/tests, existing `web/board` static assets, browser-side JavaScript/CSS, `node --test`

---

## Scope Check

This plan intentionally covers only the next Phase 2 sub-project:

- inline preview for `png`, `jpg`, `jpeg`, `gif`, `webp`, and best-effort `svg`
- metadata/download fallback for all other binary artifacts
- reuse of the existing artifact workbench route and manager APIs
- docs and smoke guidance updates

Explicitly out of scope for this plan:

- new routes
- new manager API fields
- PDF preview
- video/audio preview
- image lightbox/overlay
- editing or annotation
- media-specific plugin frameworks

Follow-on Phase 2 plans should cover:

- richer media viewing
- PDF/video/audio support
- compare-oriented artifact experiences

## File Structure

### Raw-content contract and HTTP safety behavior

- Modify: `internal/app/query/artifact_workbench.go`
  Responsibility change: keep the normalized workbench `content_type` authoritative and verify the approved previewable image types are classified correctly in the real query path.
- Modify: `internal/app/query/artifact_workbench_test.go`
  Responsibility change: cover approved image content-type classification and non-image fallback classification in the real query path.
- Modify: `internal/bootstrap/app.go`
  Responsibility change: keep the live raw-content path aligned with the authoritative workbench `content_type`.
- Modify: `internal/bootstrap/app_test.go`
  Responsibility change: verify the live runtime preserves the expected image-preview/binary-fallback contract.
- Modify: `internal/adapters/http/manager_handlers.go`
  Responsibility change: extend the existing raw-content disposition logic so approved raster image types are inline-previewable while active or untrusted types remain attachment/download-oriented.
- Modify: `internal/adapters/http/manager_handlers_test.go`
  Responsibility change: verify the raw-content endpoint behavior for previewable image types, non-previewable binary types, and the explicit current SVG server behavior.

### Artifact workbench UI integration

- Modify: `web/board/artifact-workbench.js`
  Responsibility change: add inline image preview behavior for approved image types and keep binary fallback behavior for everything else.
- Create: `web/board/artifact-workbench-binary.test.js`
  Responsibility: execute page-level image-preview and binary-fallback behavior under `node --test`.
- Modify: `web/board/styles.css`
  Responsibility change: add image-preview and binary-fallback presentation styles inside the existing artifact workbench layout.
- Modify: `internal/adapters/http/board_handlers_test.go`
  Responsibility change: verify the artifact workbench asset strings for image-preview detection, binary fallback, and raw-content usage.

### Documentation

- Modify: `README.md`
  Responsibility change: mention binary/media preview for approved image types inside the existing artifact workbench.
- Modify: `INSTALL.md`
  Responsibility change: add an optional browser-only smoke path for image-preview and binary fallback behavior.
- Modify: `CHANGELOG.md`
  Responsibility change: record the binary/media preview slice.

## Runtime Path and Ownership

The existing runtime path this plan extends is:

- `internal/app/query/artifact_workbench.go`
- `internal/bootstrap/app.go`
- `internal/adapters/http/manager_handlers.go`
- `internal/adapters/http/manager_handlers_test.go`
- `web/board/artifact-workbench.js`
- `web/board/styles.css`
- `internal/adapters/http/board_handlers_test.go`

Primary ownership by package:

- `internal/app/query`: authoritative workbench `content_type` classification for previewability
- `internal/bootstrap`: live runtime raw-content behavior aligned with workbench metadata
- `internal/adapters/http`: raw-content contract and previewability behavior at the HTTP boundary
- `web/board`: artifact workbench binary/media preview integration and styling

## Task 1: Extend Raw Content Behavior For Previewable Image Types

**Files:**
- Modify: `internal/app/query/artifact_workbench.go`
- Modify: `internal/app/query/artifact_workbench_test.go`
- Modify: `internal/bootstrap/app.go`
- Modify: `internal/bootstrap/app_test.go`
- Modify: `internal/adapters/http/manager_handlers.go`
- Modify: `internal/adapters/http/manager_handlers_test.go`

- [ ] **Step 1: Write the failing HTTP tests**

```go
func TestManagerArtifactContentEndpointKeepsRasterImagesInline(t *testing.T) {}
func TestManagerArtifactContentEndpointKeepsBinaryArtifactsAsAttachment(t *testing.T) {}
func TestManagerArtifactContentEndpointKeepsSVGAsAttachmentUnderCurrentPolicy(t *testing.T) {}
func TestArtifactWorkbenchQueryClassifiesApprovedImageTypes(t *testing.T) {}
func TestServeArtifactContentPreservesImagePreviewContract(t *testing.T) {}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/adapters/http -run ArtifactContent`
Expected: FAIL because the current raw-content contract still only treats text-like types as inline-safe

Run: `go test ./internal/app/query -run ArtifactWorkbench`
Expected: FAIL because the real query path does not yet verify approved image-type classification

Run: `go test ./internal/bootstrap -run Artifact`
Expected: FAIL because the live runtime path is not yet verified against the binary/media preview contract

- [ ] **Step 3: Implement the minimal raw-content contract change**

Rules:

- reuse `GET /api/manager/artifacts/:id/content`; do not add a new route
- `image/png`, `image/jpeg`, `image/gif`, and `image/webp` should be inline-previewable through the existing endpoint
- `image/svg+xml` should remain attachment/download under the current server safety policy in this slice
- SVG still remains best-effort at the page level because the browser-side preview path may attempt preview and fall back cleanly when the current server policy keeps it download-oriented
- non-image binary/media types must remain attachment/download oriented
- the normalized workbench `content_type` from the real query path is the authoritative previewability signal
- do not weaken the current safety guarantees for active or untrusted content

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/adapters/http -run ArtifactContent`
Expected: PASS

Run: `go test ./internal/app/query -run ArtifactWorkbench`
Expected: PASS

Run: `go test ./internal/bootstrap -run Artifact`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/query/artifact_workbench.go internal/app/query/artifact_workbench_test.go internal/bootstrap/app.go internal/bootstrap/app_test.go internal/adapters/http/manager_handlers.go internal/adapters/http/manager_handlers_test.go
git commit -m "feat: allow image artifact previews"
```

## Task 2: Add Image Preview And Binary Fallback To Artifact Workbench

**Files:**
- Modify: `web/board/artifact-workbench.js`
- Create: `web/board/artifact-workbench-binary.test.js`
- Modify: `web/board/styles.css`
- Modify: `internal/adapters/http/board_handlers_test.go`

- [ ] **Step 1: Write the failing asset and Node tests**

```go
func TestArtifactWorkbenchJavaScriptUsesInlineImagePreviewForSupportedImages(t *testing.T) {}
func TestArtifactWorkbenchJavaScriptKeepsBinaryArtifactsOnFallbackPath(t *testing.T) {}
func TestArtifactWorkbenchJavaScriptUsesRawContentURLForImagePreview(t *testing.T) {}
func TestArtifactWorkbenchJavaScriptKeepsSVGOnBestEffortPath(t *testing.T) {}
```

Add executable Node tests for:

- image-preview rendering for png/jpg/gif/webp
- SVG fallback behavior under the current server attachment/download policy
- non-image binary fallback rendering
- actual image load failure fallback behavior
- raw artifact action visibility in both previewable and non-previewable cases
- no disruption to existing text-renderer / log-ergonomics behavior

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/adapters/http -run ArtifactWorkbench`
Expected: FAIL because the page still has only text-oriented preview behavior

Run: `node --test web/board/artifact-workbench-binary.test.js`
Expected: FAIL because the binary/media preview composition behavior does not exist yet

- [ ] **Step 3: Implement the minimal artifact workbench preview changes**

Rules:

- keep the current artifact workbench route and page shape
- use the normalized server `content_type` as the authoritative previewability signal
- render approved image types inline inside the current artifact detail area
- keep metadata and raw artifact action visible alongside inline previews
- SVG remains best-effort under the current policy; because the server keeps SVG on attachment/download in this slice, the page must fall back cleanly to the binary metadata/download path
- non-image binary artifacts stay on the metadata/download fallback path
- if an image preview fails to load in the browser, fall back to the binary metadata/download path without breaking the page
- no new route or nested page is introduced
- existing text renderer polish and log ergonomics must keep working unchanged

- [ ] **Step 4: Run tests to verify they pass**

Run: `node --check web/board/artifact-workbench.js`
Expected: PASS

Run: `node --test web/board/artifact-workbench.test.js`
Expected: PASS

Run: `node --test web/board/artifact-workbench-log.test.js`
Expected: PASS

Run: `node --test web/board/artifact-workbench-binary.test.js`
Expected: PASS

Run: `go test ./internal/adapters/http -run ArtifactWorkbench`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/board/artifact-workbench.js web/board/artifact-workbench-binary.test.js web/board/styles.css internal/adapters/http/board_handlers_test.go
git commit -m "feat: preview binary artifacts"
```

## Task 3: Update Docs And Smoke Guidance

**Files:**
- Modify: `README.md`
- Modify: `INSTALL.md`
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Write the failing docs checklist**

Create a short checklist in your working notes:

```text
- README mentions approved image inline preview and binary fallback
- INSTALL includes an optional browser-only image-preview/binary-fallback smoke path
- CHANGELOG records the binary/media preview slice
```

- [ ] **Step 2: Verify the docs are incomplete**

Run: `rg -n "image preview|binary fallback|png|jpeg|gif|webp|svg|browser-only" README.md INSTALL.md CHANGELOG.md`
Expected: missing or incomplete matches

- [ ] **Step 3: Implement the docs update**

Rules:

- document that binary/media preview stays inside the existing artifact workbench
- document that the smoke is optional/browser-only and assumes a suitable artifact already exists
- document that approved image types preview inline, while non-image binary artifacts stay on metadata/download fallback
- document SVG as best-effort under the current safety policy

- [ ] **Step 4: Run verification**

Run: `rg -n "image preview|binary fallback|png|jpeg|gif|webp|svg|browser-only" README.md INSTALL.md CHANGELOG.md`
Expected: matches present where intended

- [ ] **Step 5: Commit**

```bash
git add README.md INSTALL.md CHANGELOG.md
git commit -m "docs: add artifact binary preview guidance"
```

## Milestone Verification

Run these after all tasks complete:

```bash
go test ./internal/adapters/http -run 'ArtifactContent|ArtifactWorkbench'
go test ./internal/app/query -run ArtifactWorkbench
go test ./internal/bootstrap -run Artifact
go test ./...
node --check web/board/artifact-workbench.js
node --test web/board/artifact-workbench.test.js
node --test web/board/artifact-workbench-log.test.js
node --test web/board/artifact-workbench-binary.test.js
```

Manual smoke:

- use an existing artifact whose `content_type` is one of the approved image types or another binary type
- open `/board/artifacts/workbench?artifact_id=<artifact-id>` in a browser
- verify inline image preview for approved image types, and metadata/download fallback for other binary artifacts
