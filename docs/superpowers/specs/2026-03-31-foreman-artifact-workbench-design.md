# Foreman Artifact Workbench Design

Date: 2026-03-31
Status: approved from interactive design review

## Summary

Phase 2 should add a dedicated artifact drill-down workbench for operators.

This page should sit beneath the run-detail workbench and act as the artifact-level detail surface for one selected artifact. Its purpose is to help an operator quickly understand:

- what this artifact is
- which run it belongs to
- whether it has a readable preview
- what sibling artifacts exist for the same run
- where to navigate next

The artifact workbench remains inside Foreman's control-plane boundary:

- it improves execution observability and operator usability
- it does not become a general file browser
- it does not replace the run-detail workbench
- it does not expose raw filesystem access to the browser

## Problem

Foreman now exposes:

- board overview
- approval workbench
- task-detail workbench
- run-detail workbench

But artifact navigation still stops one level too early.

The run-detail workbench can explain the run outcome and list artifact summaries, but it does not yet give operators a dedicated artifact-level page for:

- reading a bounded text preview
- switching between sibling artifacts from the same run
- opening the raw content safely through Foreman
- reconnecting that artifact to the surrounding run context

For Phase 2, Foreman needs a dedicated artifact drill-down workbench that answers:

- what artifact am I looking at?
- what is the best short explanation of it?
- can Foreman show a safe preview directly?
- what other artifacts belong to the same run?
- how do I get back to the run-level troubleshooting view?

## Goals

- Add a dedicated artifact workbench beneath the run-detail workbench
- Make `artifact_id` the stable identity for the page
- Show bounded preview content for text-like artifacts
- Keep sibling artifact switching scoped to the same run
- Provide a safe raw-content endpoint through Foreman
- Preserve a direct navigation path back to the related run workbench

## Run Linkage Source Of Truth

This design requires one explicit source-of-truth decision:

- artifact drill-down is based on a durable `artifact -> run` linkage

The persisted artifact model is currently task-scoped, but same-run sibling navigation and `run_workbench_url` require Foreman to resolve one artifact to one exact run without guessing.

For this slice, the durable source of truth should be:

- add persisted `run_id` linkage to artifact records
- new artifact writes must persist `run_id`
- same-run sibling lookup must use persisted `run_id`, not timestamp inference

### Legacy Artifacts

Existing task-scoped artifacts without reliable run linkage should remain legacy artifacts.

Foreman should not guess run ownership for those rows from timestamps alone.

That means:

- linked artifacts participate fully in artifact workbench
- legacy unlinked artifacts remain compatible through run-workbench in-page behavior until rewritten by newer runs or explicitly backfilled later

This avoids ambiguous historical reconstruction and gives the artifact workbench a stable durable model.

## Non-Goals

This design does not include:

- cross-run artifact comparison
- artifact editing, deletion, or regeneration
- full binary or media rendering
- a full log viewer
- direct browser access to filesystem paths
- board-wide redesign

## Product Position

The artifact workbench is an artifact-level drill-down page.

It is not:

- a replacement for the run-detail workbench
- a generic file explorer
- a cross-run diff tool
- a general-purpose download manager

It should instead connect:

- run-detail workbench
- artifact-level understanding
- safe preview and raw-content access

## Primary Role

The approved primary role for this page is:

- `artifact viewer with same-run drill-down`

That means the page should prioritize:

- the current artifact
- a readable preview when possible
- same-run sibling navigation
- a clear route back to run workbench

It should not start as a richer comparison or media experience.

## Entry And URL Shape

The artifact workbench should be a dedicated standalone page:

- `/board/artifacts/workbench?artifact_id=<artifact-id>`

The page identity should be based on `artifact_id` only.

Run and task context should be derived server-side from the artifact record and associated task/run records.

The artifact workbench should be reachable from:

1. run-detail workbench artifact rows
2. direct URL entry

From the artifact workbench the operator should be able to:

- refresh the same artifact
- open raw content
- go back to the related run workbench

## Navigation Relationships

The approved navigation path is:

- `Run Workbench -> Artifact Workbench -> Back to Run Workbench`

Board overview and task-detail workbench do not need direct artifact entry in v1.

This keeps artifact drill-down clearly below run-level troubleshooting rather than turning it into a first-class top-level operator surface.

Artifact workbench entry from run workbench is guaranteed only for linked artifacts with durable `run_id`.

Legacy unlinked artifacts should continue to use compatibility behavior on the run page rather than pretending to belong to one exact run.

## Layout

The recommended v1 layout is a dedicated three-column page:

1. `Sibling Artifacts`
2. `Current Artifact`
3. `Metadata And Navigation`

### Left Column

The left column should show only sibling artifacts from the same run as the selected artifact.

Each row should show:

- `kind`
- summary when available
- selection state

This is a same-run switching surface, not a task-wide history surface.

### Center Column

The center column should show:

- artifact kind
- summary
- bounded preview
- fallback guidance for non-text artifacts

This is the primary reading surface.

### Right Column

The right column should show:

- `artifact_id`
- `run_id`
- path
- content type when known
- link back to run workbench

This is the operator context and navigation column.

## Actions

The first version should expose only:

- `Refresh artifact`
- `Back to run workbench`
- `Open raw artifact endpoint`

The artifact workbench should not directly expose:

- edit
- delete
- regenerate
- compare
- approval actions
- task actions

Those remain on adjacent workbenches or future slices.

## Content Strategy

### Detail Endpoint

Add a dedicated artifact workbench detail endpoint:

- `GET /api/manager/artifacts/:id/workbench`

This endpoint should return the full artifact workbench view, including:

- current artifact metadata
- run/task linkage
- bounded preview
- sibling artifacts from the same run
- navigation URLs

The workbench detail response should include preview content directly so the page can render the first view without an immediate second roundtrip for content.

The workbench detail response applies to linked artifacts. For legacy unlinked artifacts, the endpoint may reject the request rather than inventing same-run context.

### Raw Content Endpoint

Add a dedicated raw artifact content endpoint:

- `GET /api/manager/artifacts/:id/content`

This endpoint should return raw artifact bytes directly, with the server controlling the response `Content-Type`.

The browser should never infer raw content access from the artifact path alone.

## Preview Rule

The first version should show:

- summary
- metadata
- bounded text preview for text-like artifacts

It should not:

- stream arbitrarily large artifact bodies inline
- inline binary payloads
- become a full log viewer

### Bounded Preview

The approved preview strategy is:

- text-like artifacts may include a bounded `preview`
- the preview should be truncated at a fixed upper bound, such as `64 KB`
- the view should indicate whether the preview is truncated

This keeps the page useful for operators while staying within the control-plane UI boundary.

### Non-Text Fallback

For non-text artifacts, v1 should show:

- summary
- path
- content type when known
- raw-content/open entry

It should not force inline rendering for binary or media types.

## Sibling Scope Rule

Sibling artifact navigation should be limited to artifacts from the same persisted `run_id` as the selected artifact.

It should not include:

- all task artifacts across runs
- cross-run comparisons
- task-wide artifact history

This keeps the page aligned with the run-detail workbench and avoids prematurely turning it into a broader artifact explorer.

## Page Data Model

The artifact workbench needs a dedicated read model.

The view should include:

- `artifact_id`
- `run_id`
- `task_id`
- `project_id`
- `module_id`
- `kind`
- `summary`
- `path`
- `content_type`
- `preview`
- `preview_truncated`
- `run_workbench_url`
- `raw_content_url`
- `siblings`

The sibling rows should include:

- `artifact_id`
- `kind`
- summary when available
- current selection state or enough data for the client to derive it

The server should compute navigation URLs rather than forcing the client to infer them.

The `path` field in the workbench view should be a sanitized display path relative to Foreman's artifact root, never a raw absolute filesystem path.

## Error Handling

### Missing Artifact

If the artifact does not exist:

- return `404`
- do not render stale artifact data

### Legacy Unlinked Artifact

If the artifact row exists but does not have durable run linkage yet:

- return `409`
- explain that the artifact is not yet linked to one exact run
- do not synthesize same-run sibling context from timestamps alone

### Broken Artifact Linkage

If the artifact exists but required run/task linkage is broken:

- return `500`
- do not return `200` with embedded error-state payloads

### Preview Unavailable

If a preview cannot be shown safely:

- still return the artifact workbench view
- show metadata and raw-content access
- do not treat missing preview as a hard failure

### Raw Content Missing Or Unreadable

If the artifact row exists but the backing file is missing:

- return `410`

If the backing file exists but Foreman cannot read it safely:

- return `500`

## Control-Plane Boundary

The artifact workbench should stay inside Foreman's local control-plane role.

That means:

- preview and content access should flow through Foreman HTTP endpoints
- the browser should not touch raw filesystem paths directly
- the page should remain operator-focused, not repo-browser-focused

### Raw Content Safety

`GET /api/manager/artifacts/:id/content` should:

- always set `X-Content-Type-Options: nosniff`
- return only server-chosen `Content-Type`
- force `Content-Disposition: attachment` for active or untrusted content types such as HTML, SVG, JavaScript, or unknown types
- allow inline rendering only for explicitly safe text-like content types

The workbench page should rely on bounded preview for first-class reading and treat raw content as a controlled escape hatch, not as the default inline surface.

This keeps Foreman aligned with its Phase 2 boundary and avoids reimplementing a general artifact storage UI.

## Testing Expectations

Implementation should cover:

- artifact workbench detail query
- missing artifact `404`
- broken artifact linkage `500`
- same-run sibling scoping
- bounded preview truncation
- non-text fallback behavior
- raw content endpoint behavior
- page route and JavaScript URL-state behavior
- run-workbench to artifact-workbench navigation

## Result

If implemented as designed, Foreman will have a clean operator ladder:

- `board`
- `task-detail workbench`
- `run-detail workbench`
- `artifact workbench`

Each page will have one clear role, with artifact drill-down handled as a dedicated final detail surface rather than as a generalized file browser.
