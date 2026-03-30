# Foreman Run-Detail Workbench Design

Date: 2026-03-30
Status: approved from interactive design review

## Summary

Phase 2 should add a dedicated run-detail workbench for operators.

This page should serve as the run-level troubleshooting hub that sits beneath the task-detail workbench. Its purpose is to help an operator quickly understand:

- what happened in one run
- whether it succeeded, failed, or is still running
- what the most useful artifacts are
- which task this run belongs to
- where to navigate next

The run-detail workbench remains inside Foreman's control-plane boundary:

- it improves execution observability and operator usability
- it does not become a full log explorer
- it does not replace the task-detail workbench
- it does not expand into general PM or gateway behavior

## Problem

Foreman already exposes:

- board overview
- approval workbench
- task-detail workbench
- a basic run-detail page

But the existing run-detail page is still shallow. It exposes state and artifact rows, but it is not yet a real operator troubleshooting surface.

For Phase 2, Foreman needs a dedicated run-level workbench that answers:

- what is the current run outcome?
- what is the best short explanation of that outcome?
- which artifacts matter most?
- which task does this run belong to?
- what page should the operator go to next?

## Goals

- Add a dedicated run-detail workbench as a troubleshooting hub
- Make run state and primary conclusion visible at the top
- Show a concise artifact list with summaries
- Show the related task context and a direct jump back to task-detail workbench
- Keep the page refresh-safe through a stable URL

## Non-Goals

This design does not include:

- a full raw log viewer
- inline artifact content viewer
- stop / rerun / approval actions directly on the run page
- event timeline / audit timeline view
- websocket push
- board-wide redesign

## Product Position

The run-detail workbench is a troubleshooting page.

It is not:

- a replacement for task-detail workbench
- an artifact explorer
- a generic trace UI

It should instead connect:

- task-detail workbench
- run-level result understanding
- artifact navigation

## Primary Role

The approved primary role for this page is:

- `troubleshooting hub`

That means the page should prioritize:

- run state
- the most useful explanation of that state
- the artifacts that matter most
- the next useful navigation path

It should not start as a full execution audit page.

## Entry And URL Shape

The run-detail workbench should be a dedicated standalone page:

- `/board/runs/workbench?run_id=<run-id>`

The page identity should be based on `run_id` only.

Task and project context should be derived server-side from the run record and associated task/module/project records.

The existing `/board/runs/:id` route should remain as a compatibility entrypoint in v1, but it should issue a `302` or `303` redirect to `/board/runs/workbench?run_id=<run-id>`. The workbench URL is the canonical target for new navigation.

## Entry Relationships

The page should be reachable from:

1. task-detail workbench latest-run section
2. direct URL entry

From the run-detail workbench the operator should be able to:

- open the related task-detail workbench
- open the related artifact target

This makes the page the run-level troubleshooting surface under the task-detail workbench.

## Layout

The page should be organized from top to bottom in this order:

1. `Run State And Primary Conclusion`
2. `Key Failure Or Result Summary`
3. `Artifact List`
4. `Related Task Context`
5. `Supplemental Run Metadata`

Rationale:

- operators should first understand the run outcome
- then see the best concise explanation
- then inspect which artifacts matter
- then reconnect that run to the surrounding task context

## Actions

The first version should expose only:

- `Refresh run`
- `Open task workbench`
- `Open artifact target`

The run-detail workbench should not directly expose:

- `Stop run`
- `Retry task`
- `Dispatch task`
- `Approve`
- `Reject`

Those remain on adjacent workbenches.

## Summary Source Rule

The page should show a concise summary in this order:

1. prefer an `assistant_summary` artifact summary
2. if unavailable, fall back to run state plus the most useful artifact summaries
3. do not inline raw log content in v1

This keeps the page focused on troubleshooting instead of turning it into a log viewer.

## Artifact Depth

The first version should show only:

- artifact list
- each artifact's `kind`
- summary or path

It should not inline artifact contents.

Artifact interactions should open the artifact target or surrounding run context rather than expanding content inline.

In v1, `artifact target` means an internal anchor or selection target inside the run-detail workbench itself. Artifact rows are primarily summary rows, not navigation into a separate artifact surface. It does not imply:

- raw filesystem paths
- a dedicated artifact-detail page
- inline artifact contents

## Page Data Model

The run-detail workbench needs a dedicated read model rather than reusing the current basic `RunDetailView`.

The view should include:

- `run_id`
- `task_id`
- `project_id`
- `module_id`
- `task_summary`
- `run_state`
- `runner_kind`
- `primary_summary`
- `artifacts`
- `task_workbench_url`
- `artifact_target_urls`
- key metadata such as timestamps when available

The server should compute navigation URLs rather than forcing the client to infer them.

## API Shape

Add a dedicated run workbench detail endpoint:

- `GET /api/manager/runs/:id/workbench`

This endpoint should return the full run-detail workbench view, including:

- run state
- primary summary
- artifact list with summaries
- related task context
- navigation URLs

No new write actions are required for v1.

The API should return normal view payloads only for successful lookups. It should not use `200` with embedded error-state payloads.

## Error Handling

### Missing Run

If the run does not exist:

- return `404`
- do not render stale run data

### Missing Summary

If no `assistant_summary` artifact exists:

- show a fallback summary derived from run state and artifact summaries

### Missing Artifacts

If the run has no artifacts:

- show an explicit empty artifact state
- keep the page usable

### Missing Task Linkage

If run-to-task linkage is broken:

- return `500`
- do not silently hide task context

## V1 Scope Boundary

V1 must include:

- dedicated run-detail workbench page
- stable `run_id`-based URL
- task-detail to run-detail navigation
- run state and primary summary
- artifact list with summaries
- related task context and task-workbench navigation
- dedicated run workbench detail endpoint

V1 must not include:

- inline log viewer
- inline artifact viewer
- run stop / rerun controls
- timeline mode
- websocket push

## Testing Expectations For Planning

Implementation planning should cover:

- run workbench query tests
- summary fallback tests
- artifact list rendering tests
- HTTP endpoint tests
- UI tests for:
  - `run_id` URL state
  - run-detail link usage
  - task-workbench link usage

It should also include regression tests for:

- missing run
- missing `assistant_summary`
- no artifacts
- broken run-to-task linkage

## Follow-On Relationship

This page is the correct precursor to:

- richer artifact drill-down
- deeper run diagnostics
- optional future trace/log viewing

Those should remain follow-on work, not part of this first run-detail workbench slice.
