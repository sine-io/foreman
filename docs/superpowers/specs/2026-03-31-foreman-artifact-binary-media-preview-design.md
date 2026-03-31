# Foreman Artifact Binary/Media Preview Design

Date: 2026-03-31
Status: approved from interactive design review

## Summary

Phase 2 should improve the existing artifact workbench by adding lightweight binary/media preview behavior.

This slice should not introduce a new page or a new manager API. It should extend the current artifact workbench so operators can preview common image artifacts in place and get a clearer fallback for other binary artifacts.

The approved direction is:

- keep everything inside the current artifact workbench
- reuse the existing artifact workbench detail endpoint
- reuse the existing raw artifact content endpoint
- add inline preview only for a small set of image formats
- keep non-image binary artifacts on a metadata-plus-download path

The first version should focus on:

- inline preview for `png`, `jpg`, `jpeg`, `gif`, `webp`, and `svg`
- metadata card plus download/open behavior for other binary artifacts

## Problem

Foreman now has:

- artifact workbench
- raw artifact content streaming
- structured text renderer polish
- long-text/log ergonomics

But binary and media artifacts still behave like opaque files.

For operators, the current artifact workbench is strong for text, but weak for common visual artifacts such as screenshots or other image outputs. The page needs a minimal binary/media story so operators can see obvious visual artifacts directly and understand when an artifact is download-only.

## Goals

- Add inline preview for common image artifacts inside the existing artifact workbench
- Improve the fallback experience for non-image binary artifacts
- Preserve the existing route and manager API shape
- Keep raw artifact access on the current raw content endpoint

## Non-Goals

This design does not include:

- a new artifact page
- a dedicated media preview API
- PDF preview
- video/audio preview
- image lightbox / overlay viewer
- editing or annotation
- renderer plugin framework

## Product Position

This is binary/media preview polish inside the existing artifact workbench.

It should:

- make common visual artifacts easier to inspect
- keep the current page and data flow
- preserve the current control-plane boundary

It should not:

- become a general media browser
- become a document viewer platform
- add a second binary-focused surface below artifact workbench

## Primary Role

The approved primary role for this slice is:

- `lightweight image preview + binary fallback`

That means the page should prioritize:

- direct inspection of common images
- clear metadata and download behavior for everything else

It should not try to solve every media type in v1.

## Supported Types In V1

The approved previewable types are:

- `image/png`
- `image/jpeg`
- `image/gif`
- `image/webp`
- `image/svg+xml`

All other binary/media artifacts should stay on the fallback path:

- metadata card
- download/open raw artifact
- explicit “not previewed inline” messaging

## Route And API Shape

This slice does not add new routes.

The existing surfaces remain:

- page: `/board/artifacts/workbench?artifact_id=<artifact-id>`
- detail API: `GET /api/manager/artifacts/:id/workbench`
- raw content API: `GET /api/manager/artifacts/:id/content`

No dedicated media endpoint is added.

## Client / Server Boundary

The approved responsibility split is:

- server keeps returning existing artifact detail fields
- server continues to stream raw artifact bytes from the existing content endpoint
- client decides whether the artifact is previewable inline

This means:

- no new media-specific manager DTO
- no new image-specific route
- no server-side HTML rendering for previews

## Preview Rules

### Image Preview

If the artifact is one of the approved previewable image types, the artifact workbench should:

- render the image inline inside the current artifact detail area
- keep the existing metadata and raw artifact actions alongside it
- preserve the current route and navigation model

The preview is page-local only. No lightbox or second-stage viewer is added in v1.

### Non-Image Binary Fallback

If the artifact is not one of the approved previewable image types, the artifact workbench should:

- show the existing metadata card
- show a clear “not previewed inline” explanation
- keep the raw artifact download/open action visible

This is the approved v1 binary fallback path.

## SVG Handling

SVG is included in the previewable set, but must still follow Foreman's existing safety restrictions.

That means:

- the client may treat SVG as a previewable image
- the page must not inject raw SVG markup directly into the DOM
- the preview should use the same raw content endpoint path as other image types
- if the current raw-content safety policy or browser behavior prevents safe inline SVG display in a given case, the page must fall back to the non-image binary metadata/download path

This keeps the behavior aligned with the existing content-safety boundary.

## Raw Content Contract

This slice reuses:

- `GET /api/manager/artifacts/:id/content`

The raw content endpoint remains the source of truth for download/open behavior.

For this slice, the implementation may extend the existing inline-safe behavior of the raw content endpoint to support the approved image types, but it must not weaken the current safety guarantees for active or untrusted content.

## Error Handling

Binary/media preview must never be required for page success.

If inline preview cannot be shown:

- the artifact workbench page must remain usable
- the artifact must still expose metadata
- the raw artifact action must still work when available
- the page must clearly fall back to the non-image binary path

This applies to:

- unsupported media types
- image load failure
- SVG preview rejection under current safety restrictions

## Testing Expectations

Implementation should cover:

- image-preview detection for the approved types
- non-image binary fallback behavior
- SVG preview path vs fallback behavior
- raw artifact action visibility for previewable and non-previewable artifacts
- no route or API regressions

## Result

If implemented as designed, Foreman will keep the current artifact workbench and manager API shape while giving operators a better experience for common visual outputs:

- images preview directly in the page
- non-image binary artifacts remain explicit download-oriented artifacts
- existing control-plane boundaries stay intact

That improves the artifact workbench without turning it into a heavier media subsystem.
