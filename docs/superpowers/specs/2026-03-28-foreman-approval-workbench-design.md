# Foreman Approval Workbench Design

Date: 2026-03-28
Status: approved from interactive design review

## Summary

Phase 2 should add a dedicated approval workbench for operators.

The workbench should not replace the existing board overview. It should sit beside it as a focused operator surface for reviewing risky tasks, making approval decisions, and immediately seeing the resulting execution state.

This design stays inside Foreman's control-plane boundary:

- it improves approval handling, task governance, and operator usability
- it does not add ACP, session-routing, or gateway-PM behavior
- it does not turn the board into a full admin console

## Problem

Foreman now has stronger approval correctness and retry-safe control-plane behavior, but the operator experience is still shallow:

- approvals are only visible in a light queue on the board
- there is no dedicated review surface for one approval at a time
- risk context, task context, and recent execution context are not assembled into one decision view
- approval actions exist, but not as a deliberate operator workflow

Phase 2 should turn approvals into a real operator flow rather than a thin endpoint surface.

## Goals

- Provide a dedicated approval workbench linked from the existing board
- Let an operator scan the queue quickly and review one approval deeply
- Put risk and approval reason at the top of the decision surface
- Support `Approve` and `Reject` with clear, deterministic control-plane semantics
- Continue dispatch automatically after approval
- Preserve auditability by keeping approval records and explicit rejection reasons
- Show enough recent run and artifact context to support a decision without leaving the page

## Non-Goals

This design does not include:

- batch approve or batch reject
- full board redesign
- websocket push or live collaborative presence
- multi-user auth or RBAC
- comments, threaded discussion, or chat UX
- approval analytics or historical reporting screens
- complex saved filters or custom operator views

## Product Position

The approval workbench is a dedicated operator surface under Foreman's control-plane layer.

It is not:

- a replacement for upstream manager agents
- a gateway or session-management feature
- a new primary product mode

The board overview remains the entry-level operational homepage. The approval workbench is a focused sub-surface for one high-value control-plane workflow.

## Entry Points And Page Structure

### Existing Board

Keep the existing `/board` page as the overview surface.

The approvals area on `/board` should continue to exist, but its responsibility should narrow to:

- showing the number of pending approvals
- showing a short preview list of pending approvals
- linking into the approval workbench

The board overview should not become approval-first.

### Approval Workbench

Add a dedicated approval workbench page:

- `/board/approvals/workbench?project_id=<project-id>`

The selected approval should be reflected in the URL:

- `/board/approvals/workbench?project_id=<project-id>&approval_id=<approval-id>`

This gives the workbench:

- stable deep links
- refresh-safe state
- direct navigation to one approval without reconstructing state from local UI memory

## UI Layout

The approved direction is:

- left-side approval queue
- right-side full review panel

This hybrid layout is preferred because it balances:

- fast queue throughput
- deep review context for a single decision

### Left Queue

The left pane is a dedicated pending-approval queue.

Each row should show compact operator scan information:

- task summary
- risk indicator
- module or project context
- lightweight identifier such as task id

The selected item should load the full review panel on the right.

### Right Review Panel

The right pane is the operator review surface for one approval.

It should be organized into four sections:

1. `Risk And Approval Reason`
   - risk level
   - policy or rule that triggered approval
   - explicit reason why approval is required

2. `Task Context`
   - project id
   - module id
   - task id
   - summary
   - write scope
   - priority
   - current task state

3. `Recent Execution Context`
   - latest run id
   - latest run state
   - assistant summary preview
   - artifact list with links

4. `Actions`
   - `Approve`
   - `Reject`
   - rejection reason input
   - recent result or action feedback

The review panel should stay summary-first. It should not become a full trace explorer in v1.

## Queue Ordering

The workbench queue should be ordered by:

1. risk level
2. task priority
3. approval creation time

Rationale:

- risk-first keeps the highest-impact approvals in front of the operator
- priority keeps business urgency visible within the same risk band
- creation-time ordering avoids starving older approvals

The sorting should be performed server-side so all clients see the same queue order.

For v1, `risk level` must come from a persisted approval field rather than being re-derived at render time.

Existing approvals should keep the risk metadata they were created with, even if policy rules change later.

## Action Semantics

### Approve

`Approve` should be one click with no required note.

Semantics:

- the approval record becomes `approved`
- Foreman immediately continues dispatch for the task
- the operator should not need to click dispatch separately

Result handling:

- if execution finishes quickly, the UI may show `completed`
- if execution starts and continues, the UI should show `running`
- if dispatch fails after approval, the approval remains `approved` and the UI should show the failure result explicitly
- repeated `approve` against an already-approved approval should be a no-op success that returns current authoritative state and must not trigger a second dispatch
- `approve` against an already-rejected approval should return `409 Conflict` with current approval and task state

### Reject

`Reject` must require a reason.

Semantics:

- the approval record becomes `rejected`
- the rejection reason is persisted
- the task returns to `ready`

This means the task is not destroyed. It is sent back for revision or later resubmission, while preserving an auditable rejection record.

Additional reject semantics:

- repeated `reject` against an already-rejected approval should be a no-op success that returns current authoritative state
- `reject` against an already-approved approval should return `409 Conflict` with current approval and task state

## Interaction Flow

The approved v1 flow is:

1. operator lands in the approval workbench
2. left queue shows pending approvals ordered by risk, priority, and age
3. operator selects one approval
4. right panel loads risk context, task context, and recent execution context
5. operator chooses one action

On `Approve`:

- approval is marked approved
- dispatch is attempted immediately
- the right panel updates to show the post-approval result
- the left queue advances to the next pending approval if one exists

On `Reject`:

- operator must provide a reason
- approval is marked rejected
- task is returned to `ready`
- the right panel shows the rejection outcome
- the left queue advances to the next pending approval if one exists

If no next item exists, the workbench should show an empty state instead of stale details.

Processed approvals should not remain in the pending queue, but direct links by `approval_id` should still allow historical review.

## API And Backend Shape

Keep the lightweight overview endpoint:

- `GET /board/approvals?project_id=<project-id>`

Add workbench-specific manager-facing endpoints:

- `GET /api/manager/projects/:id/approvals`
  - returns the ordered pending approval queue for the workbench

- `GET /api/manager/approvals/:id`
  - returns one approval review view
  - includes risk explanation, task context, latest run context, assistant summary preview, artifacts, approval status, and `rejection_reason` when present

- `POST /api/manager/approvals/:id/approve`
  - approves and immediately continues dispatch

- `POST /api/manager/approvals/:id/reject`
  - requires a rejection reason in the request body
  - rejects and moves the task back to `ready`

The workbench should operate primarily on `approval_id`, not `task_id`, because the object under review is the approval decision itself.

Action responses should return the authoritative resulting approval and task state so the UI can update deterministically after success, retries, or conflicts.

## Data And State Requirements

The workbench relies on Foreman remaining the source of truth for:

- task state
- approval state
- latest run state
- artifacts

The UI should not infer approval state from board columns.

For the workbench view, the server should assemble a unified approval-review read model from persisted control-plane state.

That read model should include:

- current approval record
- task snapshot
- latest run snapshot
- relevant artifact metadata
- policy/risk explanation

## Approval Metadata Model

The workbench needs a canonical approval-review record that is stable for ordering, history, and explainability.

For v1, the approval record should carry these approval-specific metadata fields:

- `risk_level`
- `policy_rule`
- `approval_reason`
- `rejection_reason`

### Persisted Fields

The following fields should be persisted on the approval record at approval-creation time:

- `risk_level`
- `policy_rule`
- `approval_reason`

`rejection_reason` should be persisted on that same approval record when a rejection occurs.

Rationale:

- queue ordering should not depend on re-running policy code later
- historical review by `approval_id` must preserve the original reason the approval existed
- the workbench should render stable review data even if policy rules evolve after approval creation

### Derived Or Joined Fields

The following fields can be assembled at read time from other control-plane records:

- task context
- latest run context
- assistant summary preview
- artifact list

### Historical Review Requirement

`GET /api/manager/approvals/:id` must work for:

- pending approvals
- already-approved approvals
- already-rejected approvals

For rejected approvals, the detail view must return the persisted `rejection_reason`.

## Error Handling

The workbench should treat action results as explicit operator outcomes, not silent failures.

### Approve Errors

If approval succeeds but dispatch fails:

- keep the approval as `approved`
- return the dispatch failure clearly
- do not reopen the approval automatically

If an action request targets a non-pending approval:

- repeated same-direction action should return the current authoritative state without creating new side effects
- opposite-direction action should return `409 Conflict`

### Reject Errors

If reject validation fails because the reason is missing:

- reject the request
- keep the approval pending
- keep the operator on the same item

### Missing Or Stale Approval

If the selected `approval_id` no longer exists or is no longer pending:

- the workbench should show a clear non-pending or not-found state
- the queue should refresh
- the operator should not be left viewing misleading stale pending data

## Planning Anchor In The Existing Runtime

This sub-project should extend the current approval vertical slice instead of introducing a second approval stack.

The current runtime anchor is:

- board queue read path through `internal/adapters/http/router.go`, `internal/adapters/http/board_handlers.go`, `internal/app/query/approval_queue.go`, and `internal/infrastructure/store/sqlite/board_query_repo.go`
- approval action path through `internal/bootstrap/app.go` into `internal/app/command/approve_task.go`
- existing task and approval reconstruction through `internal/app/query/task_status.go`

Implementation planning should stay anchored to that path and extend it with:

- workbench-specific read models
- approval-centered HTTP handlers and DTOs
- workbench UI under `web/board/*`

## V1 Scope Boundary

V1 must include:

- independent approval workbench page
- pending queue
- full single-approval review panel
- approve-and-dispatch flow
- reject-with-reason flow
- next-item queue advancement
- latest run and artifact summary context
- approval-centered manager API endpoints

V1 must not include:

- batch actions
- approval history analytics page
- saved filters
- real-time push updates
- full trace/timeline explorer
- RBAC

## Testing Expectations For Planning

Implementation planning should cover:

- queue ordering tests
- approval detail query tests
- approve action tests
- reject action tests
- UI tests for queue selection and post-action advancement
- regression tests for:
  - approval remains approved even when post-approval dispatch fails
  - reject requires a reason
  - processed approvals leave the pending queue
  - approval workbench deep links load the expected approval

## Implementation Notes

This sub-project should stay focused on operator UX around approvals.

Follow-on Phase 2 work may later cover:

- task-detail workbench
- run-detail and artifact drill-down UX
- richer board controls

Those should remain separate follow-on efforts rather than being absorbed into this approval workbench scope.
