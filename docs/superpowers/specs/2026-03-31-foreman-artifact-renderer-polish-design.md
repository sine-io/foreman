# Foreman Artifact Renderer Polish Design

Date: 2026-03-31
Status: approved from interactive design review

## Summary

Phase 2 should add renderer polish inside the existing artifact workbench.

This slice should improve operator readability for structured text artifacts without introducing a new page layer or turning Foreman into a full document viewer.

The approved direction is:

- keep rendering inside the current artifact workbench
- keep the server thin
- let the client enhance rendering based on `content_type`
- improve only a small set of high-value text formats in v1

The first version should focus on:

- JSON pretty-print
- Markdown render
- Diff / patch structure display

Everything else should continue to fall back to the current generic text preview.

## Problem

Foreman now has a working artifact workbench:

- artifact detail page
- same-run sibling navigation
- bounded preview
- raw content endpoint

But the reading experience is still generic.

Today, JSON, Markdown, and diff-like artifacts all render as plain text. That is usable, but it forces operators to mentally parse structured content that Foreman already knows is text-like and could present more clearly.

For Phase 2, Foreman should improve readability for common structured text artifacts while keeping the product boundary intact.

## Goals

- Improve readability for structured text artifacts inside the artifact workbench
- Keep the existing artifact workbench route and manager API shape
- Preserve the current generic preview as the fallback path
- Make JSON, Markdown, and diff-like artifacts easier to scan without changing control-plane boundaries

## Non-Goals

This design does not include:

- a new artifact page or nested renderer page
- cross-run comparison
- full log search / folding / filtering
- image or binary preview rendering
- server-side HTML rendering of artifact bodies
- renderer plugins or a full renderer registry framework

## Product Position

This is renderer polish, not a new artifact subsystem.

It should:

- improve artifact readability
- stay inside the artifact workbench
- keep the current data flow and navigation shape

It should not:

- create a second viewing surface below artifact workbench
- make the server responsible for presentation HTML
- expand into a generalized multi-format document platform

## Primary Role

The approved primary role for this slice is:

- `artifact workbench text renderer enhancement`

That means the page should still be the same artifact workbench, but its preview block should become smarter for a few structured text formats.

## Renderer Strategy

The approved v1 strategy is:

- `generic text preview + small type-specific enhancements`

This means:

- keep one common artifact workbench page
- keep one common preview area
- add limited enhancements for a small set of known text-like content types

This avoids overbuilding a renderer framework too early.

## Supported Renderer Enhancements In V1

The approved enhancement set is:

1. `JSON pretty-print`
2. `Markdown render`
3. `Diff / patch structure display`

All other text-like artifacts should continue to use the generic bounded preview.

## Route And API Shape

This slice does not add new routes.

The existing surfaces remain:

- page: `/board/artifacts/workbench?artifact_id=<artifact-id>`
- detail API: `GET /api/manager/artifacts/:id/workbench`
- raw content API: `GET /api/manager/artifacts/:id/content`

This slice improves rendering inside the artifact workbench only.

## Client / Server Boundary

The approved responsibility split is:

- server returns raw `preview`, `content_type`, and existing artifact metadata
- client decides how to render that preview

That means:

- no server-side HTML generation for Markdown or diff
- no pre-rendered rich content in the manager API
- no new rendering-specific write APIs

This keeps the server as a control-plane read model and keeps view formatting in the browser.

Renderer selection may use:

- `content_type`
- existing artifact `kind`
- existing artifact `path`

This matters for diff-like artifacts because the current server path may still classify `.diff` or `.patch` as generic text content types. The client may therefore recognize diff/patch artifacts by filename extension or artifact kind in addition to `content_type`.

## Rendering Rules

### Generic Text Fallback

Any text-like artifact that does not match a specialized renderer should still render as:

- bounded preview text
- monospaced or current generic preview style
- current truncation behavior

This remains the baseline behavior.

### Truncation Rule

The existing `preview_truncated` signal remains authoritative for enhanced renderers too.

That means:

- enhanced renderers may still render truncated preview content
- the truncation warning must remain visible
- if a structured renderer would become misleading on partial data, it should fall back to generic text preview instead of pretending the preview is complete

Renderer polish must never hide the fact that the preview is partial.

### JSON

If `content_type` indicates JSON:

- parse the preview as JSON when possible
- pretty-print it with indentation
- preserve fallback to generic text if parsing fails

The page should not fail just because one JSON preview is malformed.

### Markdown

If `content_type` indicates Markdown:

- render the preview as Markdown
- keep the result inside the current artifact preview area
- sanitize output so the page does not execute arbitrary markup or script content
- keep the rendered result inert and text-focused

For v1, Markdown enhancement must not introduce:

- remote image loading
- iframe or embed rendering
- script execution
- rich media expansion
- arbitrary raw HTML passthrough

If sanitization or parsing cannot produce a safe result, fall back to plain text preview.

### Diff / Patch

If the artifact is a diff-like text artifact:

- render a structured diff-oriented view
- make additions and removals visually distinct
- preserve a readable fallback if diff parsing is partial or malformed

The client may identify diff-like artifacts by:

- `content_type`
- artifact `kind`
- path suffix such as `.diff` or `.patch`

This is a readability aid, not a full code-review tool.

## Error Handling

Renderer enhancement must never be required for page success.

If a specialized renderer fails:

- keep the artifact workbench page usable
- fall back to the existing generic preview
- do not turn a renderer problem into a page-level failure

## Testing Expectations

Implementation should cover:

- JSON preview pretty-print path
- malformed JSON fallback path
- Markdown render path
- Markdown sanitization or safe fallback behavior
- diff renderer path
- malformed diff fallback path
- generic fallback still working for unsupported text-like content
- no route or API contract regressions

## Result

If implemented as designed, Foreman will keep the same artifact workbench and API shape while making three important structured text artifact types much easier for operators to read:

- JSON
- Markdown
- Diff / patch

That gives the artifact workbench a better reading experience without expanding scope into a larger viewer platform.
