# Foreman Installation

## Prerequisites

- Go `1.26+`

Optional, for later Phase 1 work:

- `codex` CLI
- upstream manager agent such as OpenClaw

## Local Setup

From the repository root:

```bash
go test ./...
go run ./cmd/foreman --help
```

Current verification and adapter-level checks:

```bash
go test ./...
go test ./internal/adapters/cli ./internal/adapters/http ./test
go test ./internal/adapters/http -run Manager
go test ./internal/bootstrap -run Serve
```

The `serve` command now wires the SQLite-backed board and OpenClaw gateway flow, the CLI command surface can create projects/modules/tasks plus run task actions, and the board UI reads real module/task/approval data from the HTTP endpoints. This Phase 1 slice has also been smoke-tested against a live `codex` CLI, with completed runs, released leases, and persisted assistant-summary artifacts.

## Manager API Smoke

With `foreman serve` running:

```bash
curl -X POST http://localhost:8080/api/manager/commands \
  -H 'Content-Type: application/json' \
  -d '{"kind":"create_task","summary":"Bootstrap board"}'

curl http://localhost:8080/api/manager/tasks/<task-id>?project_id=demo
curl http://localhost:8080/api/manager/projects/demo/board
```

These routes expose the normalized manager-agent contract directly from Foreman without ACP or channel/gateway concerns.

## Approval Workbench API Smoke

With `foreman serve` running, verify the approval workbench queue, detail, action, and recovery routes:

1. Create a risky task and confirm the response kind is `approval_needed`.

```bash
curl -X POST http://localhost:8080/api/manager/commands \
  -H 'Content-Type: application/json' \
  -d '{"kind":"create_task","summary":"git push origin main"}'
```

2. List the approval workbench queue and copy the `approval_id`.

```bash
curl http://localhost:8080/api/manager/projects/demo/approvals
```

3. Read the approval detail and confirm the approval is pending before action.

```bash
curl http://localhost:8080/api/manager/approvals/<approval-id>
```

4. Approve the request.

```bash
curl -X POST http://localhost:8080/api/manager/approvals/<approval-id>/approve \
  -H 'Content-Type: application/json' \
  -d '{}'
```

5. For the rejection path, create another risky task, list the queue again, and reject the new `approval_id` with a reason.

```bash
curl -X POST http://localhost:8080/api/manager/approvals/<approval-id>/reject \
  -H 'Content-Type: application/json' \
  -d '{"rejection_reason":"missing rollback plan"}'
```

6. If the detail or action response shows `approval_state: "approved"` with `task_state: "approved_pending_dispatch"`, use retry-dispatch recovery instead of creating a new approval.

```bash
curl -X POST http://localhost:8080/api/manager/approvals/<approval-id>/retry-dispatch \
  -H 'Content-Type: application/json' \
  -d '{}'
```

Expected outcomes:

- `curl http://localhost:8080/api/manager/projects/demo/approvals` returns the approval workbench queue for project `demo`
- `curl http://localhost:8080/api/manager/approvals/<approval-id>` remains the source of truth for approval detail before and after actions
- `POST /approve` returns an approved approval state together with the persisted task/run state
- `POST /reject` persists and returns the rejection reason
- `approved_pending_dispatch` means approval already succeeded and only dispatch recovery remains
- `POST /retry-dispatch` is the recovery route for the `approved_pending_dispatch` state

## Task Workbench API Smoke

With `foreman serve` running, verify the task workbench detail and action routes:

1. Create a normal task and copy the returned `task_id`.

```bash
curl -X POST http://localhost:8080/api/manager/commands \
  -H 'Content-Type: application/json' \
  -d '{"kind":"create_task","summary":"Summarize current project status"}'
```

2. Read the task workbench detail and confirm it returns the task operator view.

```bash
curl http://localhost:8080/api/manager/tasks/<task-id>/workbench?project_id=demo
```

3. Dispatch the task from the task workbench action route.

```bash
curl -X POST http://localhost:8080/api/manager/tasks/<task-id>/dispatch?project_id=demo
```

4. Retry the task from the task workbench action route after the task reaches a retryable state.

```bash
curl -X POST http://localhost:8080/api/manager/tasks/<task-id>/retry?project_id=demo
```

5. Cancel the task from the task workbench action route while it is still cancellable.

```bash
curl -X POST http://localhost:8080/api/manager/tasks/<task-id>/cancel?project_id=demo
```

6. Reprioritize the task through the task workbench action route.

```bash
curl -X POST http://localhost:8080/api/manager/tasks/<task-id>/reprioritize?project_id=demo \
  -H 'Content-Type: application/json' \
  -d '{"priority":42}'
```

7. In the board UI, follow the task path from board to task workbench to run detail, and confirm the task page also links to approval workbench when approval history exists.

8. For a task with no approvals, confirm the task workbench still shows the approval-workbench link in a disabled state with reason `No approval history`.

Expected outcomes:

- `GET /api/manager/tasks/<task-id>/workbench?project_id=demo` returns the task-detail workbench view for that task
- `POST /dispatch?project_id=demo` starts or resumes task execution through the task workbench action path
- `POST /retry?project_id=demo` reissues work only from a retryable task state
- `POST /cancel?project_id=demo` cancels work only when the task is still cancellable
- `POST /reprioritize?project_id=demo` accepts a JSON body and persists the new task priority
- the task page links operators to approval workbench and run detail when those destinations exist
- the no-approval state keeps the approval-workbench link visible but disabled with reason `No approval history`

## Control-Plane Hardening Smoke

With `foreman serve` running, verify repeated dispatch and persisted task-status reconstruction:

1. Create a normal task and note the returned `task_id`.

```bash
curl -X POST http://localhost:8080/api/manager/commands \
  -H 'Content-Type: application/json' \
  -d '{"kind":"create_task","summary":"Summarize current project status"}'
```

2. Read the authoritative persisted status.

```bash
curl http://localhost:8080/api/manager/tasks/<task-id>?project_id=demo
```

3. Dispatch the same task again.

```bash
curl -X POST http://localhost:8080/api/manager/commands \
  -H 'Content-Type: application/json' \
  -d '{"kind":"dispatch_task","project_id":"demo","task_id":"<task-id>"}'
```

4. Read task status again and confirm Foreman returns the persisted run/task state instead of creating a duplicate run.

```bash
curl http://localhost:8080/api/manager/tasks/<task-id>?project_id=demo
```

5. Create a risky task and note the returned `task_id`.

```bash
curl -X POST http://localhost:8080/api/manager/commands \
  -H 'Content-Type: application/json' \
  -d '{"kind":"create_task","summary":"git push origin main"}'
```

6. Confirm the task is approval-gated, then repeat dispatch and verify the same pending approval is reused.

```bash
curl http://localhost:8080/api/manager/tasks/<task-id>?project_id=demo
curl -X POST http://localhost:8080/api/manager/commands \
  -H 'Content-Type: application/json' \
  -d '{"kind":"dispatch_task","project_id":"demo","task_id":"<task-id>"}'
curl http://localhost:8080/api/manager/tasks/<task-id>?project_id=demo
```

Expected outcomes:

- repeated `dispatch_task` returns authoritative persisted state instead of re-running completed work
- approval-gated retries do not create duplicate pending approvals
- task status reports latest run and latest approval consistently through `/api/manager/tasks/:id`

## Repository Purpose

This repository is now Foreman-only.

It intentionally excludes the previous shell-runtime and skill-packaging line. If you are looking for the earlier Codex tmux/runtime wrapper flow, that is no longer part of this codebase.

## Design and Plan

- Spec: [docs/superpowers/specs/2026-03-27-foreman-go-design.md](/root/link/repo/docs/superpowers/specs/2026-03-27-foreman-go-design.md)
- Plan: [docs/superpowers/plans/2026-03-27-foreman-go-phase-1.md](/root/link/repo/docs/superpowers/plans/2026-03-27-foreman-go-phase-1.md)
- Approval workbench spec: [docs/superpowers/specs/2026-03-28-foreman-approval-workbench-design.md](/root/link/repo/docs/superpowers/specs/2026-03-28-foreman-approval-workbench-design.md)
- Approval workbench plan: [docs/superpowers/plans/2026-03-28-foreman-phase-2-approval-workbench.md](/root/link/repo/docs/superpowers/plans/2026-03-28-foreman-phase-2-approval-workbench.md)
- Task-detail workbench spec: [docs/superpowers/specs/2026-03-29-foreman-task-detail-workbench-design.md](/root/link/repo/docs/superpowers/specs/2026-03-29-foreman-task-detail-workbench-design.md)
- Task-detail workbench plan: [docs/superpowers/plans/2026-03-29-foreman-phase-2-task-detail-workbench.md](/root/link/repo/docs/superpowers/plans/2026-03-29-foreman-phase-2-task-detail-workbench.md)
