# Foreman Artifact Compare Design

Date: 2026-03-31
Status: approved from interactive design review

## Summary

Phase 2 should add a minimal artifact compare slice for operators.

This slice should introduce a dedicated compare page that answers one focused question:

- how does the current artifact differ from the previous artifact of the same kind for the same task?

The approved direction is:

- add a dedicated compare page
- use only the current `artifact_id` as the entry identity
- compare against the immediately previous artifact with the same `task_id` and `kind`
- support only text and structured-text artifacts in v1
- generate the compare result on the server
- keep compare read-only

The first version should focus on:

- one current artifact
- one previous same-task same-kind artifact
- one unified text diff
- stable empty and fallback states

## Problem

Foreman now has:

- artifact workbench
- structured renderer polish
- long-text ergonomics
- binary/media preview

But once an operator understands one artifact, there is still no direct answer to:

- what changed since the previous artifact?
- is this output different in a meaningful way?
- did the same artifact kind regress or improve?

Today an operator can inspect one artifact at a time, but cannot directly compare it with the previous version in the same task flow.

For Phase 2, Foreman needs a narrow compare surface that solves the most common comparison question without turning artifact handling into a large history browser.

## Goals

- Add a dedicated compare page beneath the artifact workbench
- Make the current `artifact_id` the only v1 entry identity
- Compare the current artifact with the immediately previous artifact that shares the same `task_id` and `kind`
- Keep compare read-only
- Support text and structured-text compare in v1
- Return stable compare states for “no previous”, “unsupported”, and “too large”

## Non-Goals

This design does not include:

- arbitrary artifact-to-artifact comparison
- manual selection of older historical artifacts
- same-run side-by-side artifact comparison
- image comparison
- binary/media comparison
- approve/retry/task actions from the compare page
- search, filtering, or folding controls inside compare

## Product Position

The artifact compare page is a follow-on operator surface beneath the artifact workbench.

It is not:

- a replacement for artifact workbench
- a replacement for run workbench
- a general history browser
- a multi-format compare platform

It should instead provide the smallest useful compare flow:

- inspect one current artifact
- compare it with the previous comparable version
- navigate back to the surrounding artifact and run context

## Relationship To Existing Artifact Workbench Design

This spec follows:

- `docs/superpowers/specs/2026-03-31-foreman-artifact-workbench-design.md`

That earlier spec explicitly kept compare out of the first artifact workbench slice.

After this amendment:

- artifact workbench remains the page for one selected artifact
- artifact compare becomes a separate read-only page
- artifact workbench may add a navigation action to open compare

This spec does not change raw-content behavior or the responsibilities of the existing artifact workbench detail page.

## Primary Role

The approved primary role for this slice is:

- `previous-version troubleshooting compare`

That means the page should prioritize:

- current vs previous
- same-task same-kind continuity
- an immediately readable diff

It should not start as a generic history explorer.

## Comparison Scope

The approved v1 comparison rule is:

- current artifact is selected by `artifact_id`
- previous artifact is the latest earlier artifact with the same `task_id` and `kind`

“Earlier” must be deterministic.

The ordering rule for compare selection should be:

- current artifact is identified first
- candidate historical artifacts must share the same `task_id` and `kind`
- candidates must satisfy:
  - lower `created_at` than the current artifact, or
  - same `created_at` and lower stable `artifact_id`
- the chosen previous artifact is the nearest candidate ordered by:
  - `created_at DESC`
  - `artifact_id DESC`

No alternative selector is added in v1.

If no such earlier artifact exists, the page still loads and returns a stable business state rather than failing the route.

## Entry And URL Shape

The compare page should be a dedicated standalone page:

- `/board/artifacts/compare?artifact_id=<artifact-id>`

The page identity should be based on the current `artifact_id` only.

The compare page should be reachable from:

1. artifact workbench
2. direct URL entry

The artifact workbench should expose one new navigation action:

- `Compare with previous`

## Navigation Relationships

The approved navigation path is:

- `Run Workbench -> Artifact Workbench -> Artifact Compare -> Back`

From the compare page, the operator should be able to:

- go back to the current artifact workbench
- go back to the related run workbench
- refresh the compare page

The compare page should not become a new top-level board surface in v1.

## Layout

The recommended v1 layout is a dedicated three-part page:

1. `Current Artifact Metadata`
2. `Compare Result`
3. `Previous Artifact Metadata`

### Top Navigation

The page header should provide:

- title: `Artifact Compare`
- `Back to current artifact`
- `Back to run workbench`
- `Refresh compare`

### Left Metadata

The left side should show the current artifact:

- `artifact_id`
- `run_id`
- `task_id`
- `kind`
- `content_type`
- `created_at`

### Center Compare Area

The center area should be the primary reading surface.

It should show:

- unified text diff when compare is ready
- stable empty or fallback state when compare is not available

### Right Metadata

The right side should show the previous artifact metadata when available:

- `artifact_id`
- `run_id`
- `kind`
- `content_type`
- `created_at`

If there is no previous artifact, the right side should stay present but display the corresponding empty-state explanation.

## Compare States

The compare DTO and page should use four stable business states:

- `ready`
- `no_previous`
- `unsupported`
- `too_large`

Meanings:

- `ready`: previous comparable artifact found and compare generated
- `no_previous`: no earlier artifact with the same `task_id` and `kind`
- `unsupported`: artifact is not in the text / structured-text compare scope
- `too_large`: content exceeds compare limits and compare is not generated

These are business states, not transport errors.

## Compare Query And API

Foreman should add a dedicated read model for compare, for example:

- `ArtifactCompareQuery`
- `GetArtifactCompare(ctx, artifactID)`

The manager-facing endpoint should be:

- `GET /api/manager/artifacts/:id/compare`

This endpoint should return:

- current artifact metadata
- previous artifact metadata when available
- compare `status`
- compare `diff`
- compare `messages`
- compare limits metadata
- navigation URLs

The endpoint is read-only and should not accept write actions.

## DTO Shape

The response should contain these top-level sections:

- `current`
- `previous`
- `status`
- `diff`
- `limits`
- `messages`
- `navigation`

Suggested contents:

- `current` and `previous`
  - `artifact_id`
  - `run_id`
  - `task_id`
  - `kind`
  - `content_type`
  - `created_at`
- `diff`
  - `format`
  - `content`
- `limits`
  - `max_compare_bytes`
- `messages`
  - `title`
  - `detail`
- `navigation`
  - `current_workbench_url`
  - `previous_workbench_url`
  - `back_to_run_url`

The first version should keep `diff.format` fixed at:

- `text/unified-diff`

Non-`ready` DTO contract should be explicit:

- `current` is always present
- `previous` is `null` for `no_previous`
- `previous` is still present for `unsupported` and `too_large` when the previous comparable artifact is found
- `diff` is present only for `ready`
- `diff` is `null` for `no_previous`, `unsupported`, and `too_large`
- `messages` is always present and should provide direct operator-facing text for the current state
- `navigation.current_workbench_url` and `navigation.back_to_run_url` are always present
- `navigation.previous_workbench_url` is present only when `previous` is present

## Supported Compare Content In V1

The compare page should support only:

- plain text
- JSON
- Markdown
- diff / patch text
- other text-like structured artifacts that already map onto text content

The first version should not attempt true compare for:

- images
- binary/media artifacts
- opaque non-text formats

Those artifacts should return:

- `unsupported`

Compare should become `ready` only when both the current artifact and the selected previous artifact resolve to supported text or structured-text content.

## Client / Server Boundary

The approved boundary is:

- server selects the previous artifact
- server decides compare status
- server generates unified diff text
- client renders the returned compare DTO

The client should not:

- select a different history target
- discover compare candidates on its own
- generate its own diff algorithm

This keeps compare rules in one place and makes later extension easier.

## Diff Strategy

The first version should generate a unified text diff on the server.

JSON and Markdown should still use the same compare surface in v1. They do not get separate structured compare renderers in this slice.

That means:

- one compare path
- one diff format
- one rendering surface

This keeps the initial slice narrow and predictable.

## Size Limits

Compare should be bounded.

If either side exceeds the configured compare limit, Foreman should not attempt an unbounded or truncated diff. Instead it should return:

- `too_large`

The DTO should include enough limit metadata and message text for the page to explain why compare is unavailable.

This slice should not turn compare into a large-file processor.

## Error Handling

Transport and business failures should be separated.

HTTP errors:

- `404`: current artifact does not exist
- `500`: required task/run linkage is broken

Business states returned inside a successful compare DTO:

- `no_previous`
- `unsupported`
- `too_large`

In other words:

- “the compare page is valid, but no compare can be shown” is not an HTTP error
- “the artifact graph is internally broken” is an HTTP error

## Testing Expectations

Implementation should cover:

- query logic for selecting the previous same-task same-kind artifact
- `no_previous` state
- `unsupported` state for non-text artifacts
- `too_large` state for over-limit content
- unified diff generation
- manager compare endpoint success and error semantics
- board compare page rendering for all four business states
- artifact workbench navigation into compare
- bootstrap/live wiring for the compare query

## Result

If implemented as designed, Foreman will add a narrow but useful compare surface without expanding artifact handling into a larger history system:

- operators can compare the current artifact with the immediately previous comparable version
- the compare contract stays server-authored and read-only
- artifact workbench keeps its single-artifact responsibilities
- future history selection can extend the query model without replacing the route or page identity

That gives Foreman a practical first compare workflow while preserving the product boundary established in Phase 2.
