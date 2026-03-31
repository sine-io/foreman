# Foreman Artifact Log Ergonomics Design

Date: 2026-03-31
Status: approved from interactive design review

## Summary

Phase 2 should improve the reading experience for long text and log-like artifacts inside the existing artifact workbench.

This slice should not add a new page or a new API. It should make large text artifacts easier to inspect during troubleshooting by improving scanability rather than introducing a full log viewer.

The approved direction is:

- keep everything inside the current artifact workbench
- prioritize troubleshooting readability
- add only a small set of ergonomic improvements

The first version should focus on:

- line numbers
- default collapsed first-screen preview for long text
- explicit expand-all behavior
- lightweight summary navigation derived from existing summary/preview text

## Problem

Foreman now has:

- artifact workbench
- same-run sibling navigation
- bounded preview
- renderer polish for JSON, Markdown, and diff / patch

But long plain-text and log-like artifacts are still tiring to read.

For troubleshooting, operators often need to:

- orient themselves quickly inside a long artifact
- scan for rough structure before reading in detail
- expand the whole content only when necessary

Today, long text-like artifacts still behave mostly like one large preview block. That is usable, but not ergonomic.

## Goals

- Improve readability for long text and log-like artifacts inside the existing artifact workbench
- Preserve the current route and manager API shape
- Make long previews easier to scan before full expansion
- Add lightweight navigation derived from existing text

## Non-Goals

This design does not include:

- new routes
- new manager API fields
- full text search
- filtering
- syntax-aware log analysis
- timeline views
- cross-artifact comparison
- image or binary rendering

## Product Position

This is a troubleshooting reading improvement.

It should:

- help operators read long text artifacts faster
- stay inside the current artifact workbench
- keep the current preview model intact

It should not:

- become a full log viewer product
- become a search/filter console
- add a second artifact reading surface

## Primary Role

The approved primary role for this slice is:

- `troubleshooting reading ergonomics`

That means the work should prioritize:

- quick orientation
- scanability
- controlled expansion

It should not prioritize advanced querying or heavy analysis.

## Scope Of Artifact Types

The first version should apply only to long text and log-like artifacts, including:

- `run_log`
- `command_result`
- general `text/plain` long text

This slice does not change the renderer-polish direction for:

- JSON
- Markdown
- diff / patch

Those remain handled by the structured renderer path.

### Renderer Precedence

Structured renderer selection still runs first.

That means:

- JSON stays on the JSON renderer path
- Markdown stays on the Markdown renderer path
- diff / patch stays on the structured diff renderer path

Artifact log ergonomics apply only when the artifact preview is on the generic long-text path.

This includes:

- native long text / log-like artifacts
- structured artifacts that explicitly fall back to generic text preview because their specialized renderer declines or safely falls back

It does not apply on top of the normal structured-renderer success path.

## Route And API Shape

This slice does not add new routes.

The existing surfaces remain:

- page: `/board/artifacts/workbench?artifact_id=<artifact-id>`
- detail API: `GET /api/manager/artifacts/:id/workbench`
- raw content API: `GET /api/manager/artifacts/:id/content`

The server does not need new outline or navigation fields for v1.

## Client / Server Boundary

The approved responsibility split is:

- server keeps returning the current summary / preview / metadata
- client derives lightweight navigation and expansion behavior from those existing fields

That means:

- no new server-side outline field
- no new search endpoints
- no new pagination API

This keeps the change lightweight and contained to the existing artifact workbench UI.

## Ergonomics Features

### Line Numbers

Long text and log-like previews should display line numbers.

This is intended only to improve orientation and discussion, not to create a fully addressable log protocol.

### Default Collapsed First Screen

For long text and log-like artifacts, the preview should default to a collapsed first-screen view.

The approved behavior is:

- show an initial visible slice
- make it obvious that more content exists
- allow one-click expansion to the full bounded preview

For v1, collapsed mode means:

- show a clipped first-screen teaser of the bounded preview
- do not rely on internal scrolling inside collapsed mode
- require explicit expansion before the operator reads the rest of the bounded preview

The page should not default to fully expanded long text.

Selecting a different sibling or artifact should reset that newly selected artifact back to collapsed mode by default.

### Expand All

The first version should include a simple explicit expansion action:

- `Expand all`

This expands the current bounded preview content only. It does not fetch additional content beyond the existing preview contract.

### Summary Navigation

The first version should add lightweight summary navigation derived from existing text.

The client may use:

- `summary`
- existing preview text

to extract a few rough navigation anchors.

This should remain intentionally lightweight. It is not a semantic parser and does not need to be perfect.

## Summary Navigation Rules

Summary navigation should be:

- cheap to derive
- useful for rough orientation
- safe to ignore if the artifact has no meaningful structure

The extraction can be heuristic and format-specific, but it should not require new backend fields.

Likely useful anchors include:

- obvious section starts
- hunk markers
- error-ish lines
- command boundaries

If no clear anchors exist, the navigation area may remain empty.

Summary navigation is derived only from the currently available bounded preview text and existing summary fields. It must not imply access to content beyond the bounded preview.

If an operator clicks a derived navigation anchor and that target is outside the currently visible collapsed slice, the page may auto-expand the current bounded preview to reveal it.

The truncation warning must remain visible in both collapsed and expanded states so `Expand all` is not mistaken for “show the full artifact.”

## Long-Text Detection

This slice should apply ergonomics only when artifacts are both:

- text-like
- long enough to need them

Short text should remain simple and direct.

The exact length threshold is an implementation detail, but the design requires that the page not add heavy ergonomics UI to small artifacts unnecessarily.

## Interaction Rules

The ergonomics features should not disrupt the existing artifact workbench responsibilities:

- same-run sibling switching still works the same way
- back-to-run navigation still works the same way
- raw artifact access still works the same way

This slice only improves how long text-like previews are presented.

## Error Handling

Ergonomics enhancement must never be required for page success.

If line-numbering, collapse logic, or summary-anchor extraction fails:

- the page must remain usable
- the preview must still render as generic text
- the user should not lose access to the artifact

## Testing Expectations

Implementation should cover:

- long-text detection
- line-number rendering for long text/log artifacts
- default collapsed state for long text
- explicit expand-all path
- summary-anchor extraction from existing summary/preview
- no ergonomics UI for short text artifacts
- fallback to generic text when heuristics produce nothing useful
- no route or API regressions

## Result

If implemented as designed, Foreman will keep the same artifact workbench and API shape while making long text and log-like artifacts easier to inspect:

- faster first scan
- clearer orientation
- better incremental reading

That improves troubleshooting ergonomics without turning artifact workbench into a separate log-viewer product.
