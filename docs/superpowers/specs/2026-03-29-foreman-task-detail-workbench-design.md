# Foreman Task-Detail Workbench Design

Date: 2026-03-29
Status: approved from interactive design review

## Summary

Phase 2 should add a dedicated task-detail workbench for operators.

This page should sit between the board overview, the approval workbench, and the existing run-detail page. Its job is to make one task operable from a single surface: current task state, current action availability, latest run summary, approval status, artifact summary, and task metadata should all be visible without forcing the operator to jump between multiple pages.

The task-detail workbench remains inside Foreman's control-plane boundary:

- it improves task governance and operator usability
- it does not add PM, gateway, or protocol-layer behavior
- it does not replace the approval workbench
- it does not turn the board into a full admin console

## Problem

Foreman now has:

- a board overview
- an approval workbench
- a run-detail page

But it still lacks a dedicated task-level operator page.

Right now the operator has to reconstruct a task's actual situation from multiple places:

- board columns for broad status
- approval workbench for approval context
- run detail for execution context

That is workable, but it is not an operator hub. A task-detail workbench should become the page where the operator can answer:

- what is this task's current state?
- what can I do next?
- what happened in the latest run?
- is approval involved?
- where should I go next if I need more depth?

## Goals

- Add a dedicated task-detail workbench as an operator hub
- Make task actions directly available from the page
- Show latest run summary without forcing immediate drill-down
- Show approval status and provide a clean jump to the approval workbench
- Show artifact summary without turning the page into an artifact explorer
- Keep state refresh-safe with a stable URL using `project_id` and `task_id`

## Non-Goals

This design does not include:

- full run history on the task page
- inline artifact viewer
- approval decision actions on the task page
- event timeline / audit timeline page
- board-wide redesign
- websocket push
- multi-user collaboration or RBAC

## Product Position

The task-detail workbench is an operator hub page.

It is not:

- a replacement for the board overview
- a replacement for the approval workbench
- a replacement for run detail

It should instead connect those surfaces:

- board overview can jump into it
- approval workbench can jump into it
- it can jump into run detail

## Primary Role

The approved primary role for this page is:

- `operation hub`

That means the page should prioritize:

- current task state
- what actions are available right now
- latest run summary
- approval status
- the next useful navigation path

It should not start as a passive status report or a full timeline page.

## Entry And URL Shape

The task-detail workbench should be a dedicated standalone page:

- `/board/tasks/workbench?project_id=<project-id>&task_id=<task-id>`

This URL shape is required so the page is:

- refresh-safe
- deep-linkable
- reachable from multiple entrypoints
- stable for future richer operator flows

## Entry Relationships

The page should be reachable from:

1. board task cards
2. approval workbench task references
3. direct URL entry

And from the task-detail workbench the operator should be able to:

- open the latest run detail page
- open the approval workbench when approval context is relevant

This makes the task page the middle surface between:

- board overview
- approval workbench
- run detail

The approval-workbench-to-task-workbench link should appear in the approval workbench right-side detail panel, not just in the queue.

## Layout

The page should be organized from top to bottom in this order:

1. `Task State And Primary Actions`
2. `Latest Run Summary`
3. `Approval Status`
4. `Artifact Summary`
5. `Task Metadata`

Rationale:

- the operator should see current state and available actions first
- latest run context is usually the next thing needed
- approval belongs on the page, but as status and navigation, not as a competing primary flow
- metadata should remain visible but secondary

## Actions

The first version should expose these task actions:

- `Dispatch`
- `Retry`
- `Cancel`
- `Reprioritize`
- `Open latest run`

### Explicit Boundary

The task-detail workbench should not directly expose:

- `Approve`
- `Reject`

Approval decisions remain in the approval workbench.

This page should only:

- display approval state
- display latest approval context
- link to the approval workbench

### Reprioritize Interaction

`Reprioritize` should use a lightweight numeric input on the task-detail page.

The UI behavior should be:

- show the current priority as the default value
- let the operator enter a new integer priority
- submit that value directly from the task-detail workbench

The API contract should be:

- `POST /api/manager/tasks/:id/reprioritize?project_id=<project-id>`
- request body: `{"priority": <int>}`

Validation rules:

- `priority` must be an integer
- `priority` must be `>= 1`
- invalid values should return `400`

## Action Visibility Rules

All primary task actions should remain visible even when they are currently unavailable.

Unavailable actions should:

- render disabled
- include a short reason explaining why they are disabled

Examples:

- `Waiting approval`
- `Already completed`
- `No latest run`
- `Task not failed`

This rule helps operators understand both:

- what the system can do
- why it cannot do it right now

### V1 Action Eligibility Matrix

The first version should use this server-side action matrix:

- `ready`
  - `Dispatch`: enabled
  - `Retry`: disabled, reason `Task not failed`
  - `Cancel`: enabled
  - `Reprioritize`: enabled

- `leased`
  - `Dispatch`: enabled
  - `Retry`: disabled, reason `Task not failed`
  - `Cancel`: enabled
  - `Reprioritize`: enabled

- `waiting_approval`
  - `Dispatch`: disabled, reason `Waiting approval`
  - `Retry`: disabled, reason `Task not failed`
  - `Cancel`: enabled
  - `Reprioritize`: enabled

- `approved_pending_dispatch`
  - `Dispatch`: disabled, reason `Use approval workbench retry-dispatch`
  - `Retry`: disabled, reason `Task not failed`
  - `Cancel`: enabled
  - `Reprioritize`: enabled

- `running`
  - `Dispatch`: disabled, reason `Already running`
  - `Retry`: disabled, reason `Task not failed`
  - `Cancel`: enabled
  - `Reprioritize`: enabled

- `failed`
  - `Dispatch`: disabled, reason `Use retry for failed tasks`
  - `Retry`: enabled
  - `Cancel`: enabled
  - `Reprioritize`: enabled

- `completed`
  - `Dispatch`: disabled, reason `Already completed`
  - `Retry`: disabled, reason `Task not failed`
  - `Cancel`: disabled, reason `Already completed`
  - `Reprioritize`: disabled, reason `Already completed`

- `canceled`
  - `Dispatch`: disabled, reason `Task canceled`
  - `Retry`: disabled, reason `Task canceled`
  - `Cancel`: disabled, reason `Task canceled`
  - `Reprioritize`: disabled, reason `Task canceled`

`Open latest run`:

- enabled when a latest run exists
- disabled with reason `No latest run` when none exists

## Latest Run Depth

The first version should show only the latest run.

It should not include:

- a full run history list
- a run timeline embedded into the task page

The latest run block should include:

- run id
- run state
- assistant summary preview
- link to `/board/runs/:id`

## Approval Status Depth

The task page should show approval as status and navigation only.

It should include:

- latest approval id
- latest approval state
- latest approval reason
- link to the approval workbench

It should not directly perform approval decisions.

When a latest approval exists, `approval_workbench_url` should deep-link to that approval:

- `/board/approvals/workbench?project_id=<project-id>&approval_id=<latest-approval-id>`

If no approval exists:

- the approval workbench link should remain visible but disabled
- the disabled reason should be `No approval history`

## Artifact Summary Depth

The first version should show only a concise artifact summary for the latest task artifacts.

This is intentionally a task-scoped approximation, not a new run-to-artifact linkage model.

Each artifact row should include:

- `kind`
- summary or path

Artifact interactions should not open artifact content inline.

Instead:

- the page should link the operator to the existing run-detail page
- run detail remains the deeper execution/artifact surface

## Page Data Model

The task-detail workbench needs a dedicated read model instead of reusing the current lightweight `TaskStatusView`.

The read model should include:

- `task_id`
- `project_id`
- `module_id`
- `summary`
- `task_state`
- `priority`
- `available_actions`
- `disabled_reasons`
- `latest_run_id`
- `latest_run_state`
- `latest_run_summary`
- `latest_approval_id`
- `latest_approval_state`
- `latest_approval_reason`
- `approval_workbench_url`
- `run_detail_url`
- `artifacts`
- metadata fields such as `write_scope`, `task_type`, and `acceptance`

The page should not infer action availability from UI-only rules. The server should compute the action model and return explicit availability plus disabled reasons.

`available_actions` should be an explicit per-action structure rather than parallel arrays or ad-hoc booleans. Each action entry should carry:

- action id
- enabled flag
- disabled reason when not enabled
- optional current value metadata when relevant, such as current priority for `reprioritize`

## API Shape

Add a dedicated task workbench detail endpoint:

- `GET /api/manager/tasks/:id/workbench?project_id=<project-id>`

This endpoint should return the full task-detail workbench view, including:

- task state
- action availability
- latest run summary
- approval status
- artifact summary
- metadata
- navigation URLs

Add dedicated task workbench action endpoints:

- `POST /api/manager/tasks/:id/dispatch?project_id=<project-id>`
- `POST /api/manager/tasks/:id/retry?project_id=<project-id>`
- `POST /api/manager/tasks/:id/cancel?project_id=<project-id>`
- `POST /api/manager/tasks/:id/reprioritize?project_id=<project-id>`

The task workbench should use these action endpoints directly instead of reusing board form actions or manager command envelopes.

All task workbench action endpoints should validate that the task belongs to the requested `project_id`. Cross-project mismatches should return a not-found style result instead of silently acting on the task.

## Action Response Contract

Task workbench action responses should return the authoritative resulting task state, not just `200 OK`.

The response should include at least:

- `task_id`
- `task_state`
- `latest_run_id` when relevant
- `latest_run_state` when relevant
- `latest_approval_id` when relevant
- `latest_approval_state` when relevant
- a short operator-facing message when useful

The action response should be a compact action result, not a fully refreshed workbench payload. After a successful action, the client should re-fetch the task-detail workbench view.

This keeps the page deterministic after actions and avoids additional guesswork in the client while keeping the action endpoints smaller than the detail endpoint.

## Error Handling

### Missing Task

If the task does not exist or does not belong to the requested project:

- return a not-found view
- do not render stale task data

### Ineligible Action

If the user triggers an action that is currently not allowed:

- return a conflict-style result
- keep the page on the same task
- preserve the current detail state

### Missing Run

If no latest run exists:

- latest run block should show an explicit empty state
- `Open latest run` should remain visible but disabled with reason

### Approval Linking

If no approval exists:

- approval block should show an explicit “no approval history” state
- approval workbench link should remain visible but disabled with reason `No approval history`

## V1 Scope Boundary

V1 must include:

- dedicated task-detail workbench page
- stable URL with `project_id` and `task_id`
- board entry into the page
- approval workbench entry into the page
- task action availability with disabled reasons
- latest run summary
- approval status summary
- artifact summary for latest run
- navigation to existing run detail
- dedicated task workbench detail and action endpoints

V1 must not include:

- full run history
- inline artifact content viewer
- approval decision actions on the task page
- timeline/audit mode
- websocket push

## Testing Expectations For Planning

Implementation planning should cover:

- task workbench detail query tests
- action-availability computation tests
- HTTP endpoint tests for detail and task actions
- UI tests for:
  - URL state with `project_id` and `task_id`
  - disabled action reasons
  - link to approval workbench
  - link to run detail

It should also include regression tests for:

- latest run absent
- approval absent
- cross-project task access rejection
- action response updates after dispatch/retry/cancel/reprioritize

## Follow-On Relationship

This page is the correct precursor to a later:

- richer run-detail workbench
- deeper artifact drill-down page

Those should come after this task-detail hub exists, not before.
