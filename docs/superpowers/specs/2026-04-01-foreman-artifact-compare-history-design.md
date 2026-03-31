# Foreman Artifact Compare History Design

Date: 2026-04-01
Status: approved from interactive design review

## Summary

Phase 2 should extend the existing artifact compare page with a small, selectable history surface.

This slice should not replace the compare page or turn it into a general history browser. It should extend the current compare contract so operators can move beyond “current vs previous” and compare the current artifact against a small number of recent historical artifacts from the same task and kind.

The approved direction is:

- keep compare on the existing compare page
- keep the current artifact as the primary page identity
- add a small history list inside the compare page
- allow an optional `previous_artifact_id` in the compare URL and API
- keep selection rules and validation on the server

The first version should focus on:

- the current artifact
- one selected historical compare target
- a recent history list limited to five items
- deep-linkable compare target selection

## Problem

Foreman now supports:

- artifact workbench
- artifact compare against the immediately previous same-task same-kind text artifact

That is enough for the first comparison question, but not enough when an operator needs to answer:

- did this change start one run ago or three runs ago?
- which earlier version should I compare against?
- how does the current artifact differ from a specific recent historical version?

Today the compare page is fixed to one default history target. That keeps the first version simple, but it prevents short-range operator investigation inside the compare flow.

For Phase 2, Foreman needs a minimal history-selection extension that keeps compare focused while allowing a small amount of controlled historical exploration.

## Goals

- Add a recent-history list to the compare page
- Keep the current `artifact_id` as the primary page identity
- Allow a selected compare target through optional `previous_artifact_id`
- Make history target selection deep-linkable and refresh-safe
- Keep selection validation and compare target resolution on the server
- Limit the history surface to a small recent window instead of building a full history browser

## Non-Goals

This design does not include:

- full artifact history pagination
- arbitrary artifact-to-artifact compare outside the same task and kind
- image or media compare
- standalone artifact history page
- manual artifact-id text entry
- search, filtering, or sorting controls

## Product Position

This slice is a focused enhancement to the current artifact compare page.

It is not:

- a replacement for the compare page
- a replacement for artifact workbench
- a complete artifact history system

It should instead provide:

- one current artifact
- one selected historical target
- one bounded recent history list

## Relationship To Existing Compare Design

This spec follows:

- `docs/superpowers/specs/2026-03-31-foreman-artifact-compare-design.md`

That earlier spec fixed compare to:

- current artifact
- default previous artifact
- no manual history selection

After this amendment:

- compare remains read-only
- compare still centers on one current artifact
- compare may use either:
  - the default previous artifact
  - an explicitly selected recent historical artifact

This spec does not change the existing compare business states:

- `ready`
- `no_previous`
- `unsupported`
- `too_large`

## Primary Role

The approved primary role for this slice is:

- `bounded historical target selection`

That means the page should prioritize:

- quick compare target switching
- stable deep links
- recent history only

It should not become a full history explorer.

## URL Shape

The compare page remains:

- `/board/artifacts/compare?artifact_id=<current-artifact-id>`

This slice adds one optional query parameter:

- `previous_artifact_id=<historical-artifact-id>`

Rules:

- `artifact_id` stays required
- `previous_artifact_id` is optional
- if `previous_artifact_id` is absent, the server uses the existing default previous-artifact rule
- if `previous_artifact_id` is present, the server validates and uses it as the compare target

The current artifact remains the primary identity of the page.

## Compare API Contract

The compare API remains:

- `GET /api/manager/artifacts/:id/compare`

This slice adds support for:

- `GET /api/manager/artifacts/:id/compare?previous_artifact_id=<artifact-id>`

The same compare endpoint should now do two jobs:

- return the compare result for the selected target
- return the bounded recent-history list for the page

No dedicated history endpoint is added.

## History Selection Rules

The recent-history list should contain at most five artifacts.

The list must include only artifacts that:

- are earlier than the current artifact
- share the same `task_id`
- share the same `kind`

Ordering should be:

- `created_at DESC`
- `artifact_id DESC`

That means the first list item remains the default compare target when no explicit `previous_artifact_id` is provided.

The recent-history list is also the authoritative selection window for this slice.

That means:

- explicit `previous_artifact_id` selection is only valid when the target artifact is inside this bounded recent-history set
- older artifacts outside the recent five-item window are out of scope for this slice, even if they otherwise match `task_id`, `kind`, and earlier-ordering rules

## Explicit `previous_artifact_id` Validation

When `previous_artifact_id` is provided, the server must validate that it:

- exists
- belongs to the same `task_id`
- belongs to the same `kind`
- is earlier than the current artifact under the existing compare ordering rules

If validation fails, the server should return:

- `400`

This is a client error, not a compare business state.

The server must not silently ignore an invalid `previous_artifact_id` and fall back to the default previous artifact.

## Response Shape

The compare response should keep the existing sections:

- `current`
- `previous`
- `status`
- `diff`
- `limits`
- `messages`
- `navigation`

This slice adds one new top-level section:

- `history`

Each `history` item should contain:

- `artifact_id`
- `run_id`
- `created_at`
- `summary`
- `selected`
- `compare_url`

`selected` means:

- this history item is the active compare target for the current page

`compare_url` means:

- the exact compare page URL for selecting this item, including both `artifact_id` and `previous_artifact_id`

`history` should always be present in successful compare responses.

Rules:

- `history` is an array in all successful responses
- `history` is empty when there is no recent history to show
- `history` remains present for `ready`, `no_previous`, `unsupported`, and `too_large`

## Page Layout

The compare page keeps its three-column structure:

1. `Current Artifact Metadata`
2. `Compare Result`
3. `Selected History Artifact` plus `Recent History`

The right column is the only area that expands.

It should now contain:

- metadata for the currently selected historical artifact
- a bounded `Recent History` list below it

The history list should not appear as a separate page section or overlay.

## Page Interaction

Page entry rules:

- without `previous_artifact_id`, the server chooses the default previous artifact
- with `previous_artifact_id`, the server uses the validated explicit selection

Interaction rules:

- clicking a history row navigates to the corresponding `compare_url`
- the page then reloads compare data for that selected target
- browser refresh, back, and forward should preserve the selected compare target through the URL

The compare page should not support:

- manual artifact-id text entry
- in-page artifact-id editing
- multi-select compare
- arbitrary jump outside the five-item history window

## Navigation Rules

Compare-page actions remain:

- `Back to current artifact`
- `Back to run workbench`
- `Refresh compare`

History selection is not a new top-level action. It is part of the recent-history list.

No additional compare action buttons are added in this slice.

## Error Handling

Transport errors:

- `404`: current artifact does not exist
- `500`: current artifact task/run linkage is broken
- `400`: explicit `previous_artifact_id` is invalid

Business states still returned in successful compare responses:

- `ready`
- `no_previous`
- `unsupported`
- `too_large`

This means:

- invalid explicit history target is a client error
- absence of a default history target is still a normal compare business state

For the compare page, `400` should continue to use normal transport-error handling rather than introducing a fifth compare business state. The page should stay read-only and surface the invalid-selection error through its existing API-error path.

## Testing Expectations

Implementation should cover:

- default history list size and ordering
- default selected target equals the first valid history item
- explicit `previous_artifact_id` selection
- invalid `previous_artifact_id` returning `400`
- compare response `history[].selected`
- compare response `history[].compare_url`
- page rendering of the history list
- history click -> URL update -> refetch flow
- browser back/forward behavior for selected history targets
- continued read-only behavior for the compare page

## Result

If implemented as designed, Foreman will extend the current compare page from:

- current vs previous

to:

- current vs one selected recent historical artifact

without expanding into a full history system.

That gives operators a bounded but useful historical compare workflow while preserving the existing compare page, compare API, and read-only control-plane boundary.
