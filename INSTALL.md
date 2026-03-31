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

7. In the board UI, follow the task path from board to task workbench to run workbench, and confirm the task page now links to the canonical run workbench route while still linking to approval workbench when approval history exists.

8. For a task with no approvals, confirm the task workbench still shows the approval-workbench link in a disabled state with reason `No approval history`.

Expected outcomes:

- `GET /api/manager/tasks/<task-id>/workbench?project_id=demo` returns the task-detail workbench view for that task
- `POST /dispatch?project_id=demo` starts or resumes task execution through the task workbench action path
- `POST /retry?project_id=demo` reissues work only from a retryable task state
- `POST /cancel?project_id=demo` cancels work only when the task is still cancellable
- `POST /reprioritize?project_id=demo` accepts a JSON body and persists the new task priority
- the task page links operators to approval workbench and the canonical run workbench route when those destinations exist
- the no-approval state keeps the approval-workbench link visible but disabled with reason `No approval history`

## Run Workbench Smoke

With `foreman serve` running, verify the run workbench detail and compatibility routes:

1. Use a run created during the task workbench smoke and copy its `run_id` from the task workbench response or board UI.

2. Read the manager run workbench detail.

```bash
curl http://localhost:8080/api/manager/runs/<run-id>/workbench
```

3. Load the canonical board run workbench page.

```bash
curl http://localhost:8080/board/runs/workbench?run_id=<run-id>
```

4. Check the legacy board route compatibility redirect.

```bash
curl -sS -D - -o /dev/null http://localhost:8080/board/runs/<run-id>
```

Expected outcomes:

- `GET /api/manager/runs/<run-id>/workbench` returns the run-detail workbench view for that run
- `GET /board/runs/workbench?run_id=<run-id>` serves the canonical run workbench page
- `GET /board/runs/<run-id>` redirects to `/board/runs/workbench?run_id=<run-id>`
- the task workbench now links to the canonical run workbench route instead of the legacy run page

## Artifact Workbench Smoke

With `foreman serve` running, verify the artifact workbench detail, raw-content, and board routes:

1. Use a run from the run workbench smoke and copy one artifact ID from the manager run workbench response `artifacts[].id` field.

2. Read the manager artifact workbench detail.

```bash
curl http://localhost:8080/api/manager/artifacts/<artifact-id>/workbench
```

3. Stream the raw artifact content and inspect the response headers.

```bash
curl -i http://localhost:8080/api/manager/artifacts/<artifact-id>/content
```

4. Load the canonical board artifact workbench page.

```bash
curl http://localhost:8080/board/artifacts/workbench?artifact_id=<artifact-id>
```

5. In the board UI, follow the run workbench artifact link and confirm it lands on the canonical artifact workbench route.

Expected outcomes:

- `GET /api/manager/artifacts/<artifact-id>/workbench` returns the artifact workbench view for that artifact
- `GET /api/manager/artifacts/<artifact-id>/content` streams the raw artifact content with safe response headers
- `GET /board/artifacts/workbench?artifact_id=<artifact-id>` serves the canonical artifact workbench page
- the run workbench links linked artifacts to the canonical artifact workbench route

Legacy `#artifact-...` run-page anchors remain as a compatibility fallback for older unlinked artifacts, but that behavior is not part of the normal reproducible smoke because newly created artifacts are linked.

## Artifact Renderer Polish Smoke (Optional Browser Check)

With `foreman serve` running, you can manually verify renderer polish inside the existing artifact workbench if you already have a suitable artifact. The documented clean-repo smoke above does not guarantee creation of a JSON, Markdown, or diff / patch artifact.

1. If you already have an artifact whose `content_type`, `kind`, or `path` maps to JSON, Markdown, or diff / patch, note its `artifact_id`. Otherwise, skip this optional check.

2. Read the manager artifact workbench detail and note the preview metadata that drives renderer selection.

```bash
curl http://localhost:8080/api/manager/artifacts/<artifact-id>/workbench
```

3. Open `http://localhost:8080/board/artifacts/workbench?artifact_id=<artifact-id>` in a web browser and inspect the preview there. The renderer polish is client-side, so `curl` against the board route only verifies the static shell, not the rendered preview.

Expected outcomes:

- renderer polish stays inside `/board/artifacts/workbench?artifact_id=<artifact-id>` and does not add a new route
- in the browser, JSON artifacts pretty-print, Markdown artifacts render safely as inert formatted content, and diff / patch artifacts use a structured diff-oriented preview
- the smoke artifact may be selected by `content_type`, artifact `kind`, or filename `path`
- unsupported content, and renderer fallback cases such as JSON parse failures, still use the generic text preview
- truncation warnings remain visible when the preview is partial

## Artifact Binary/Media Preview Smoke (Optional Browser Check)

With `foreman serve` running, you can manually verify browser-only image preview and binary fallback behavior inside the existing artifact workbench if you already have a suitable artifact. The documented clean-repo smoke above does not guarantee creation of an approved image artifact or another binary artifact.

1. If you want to confirm the successful inline-preview path, start with an artifact whose `content_type` is `image/png`, `image/jpeg`, `image/gif`, or `image/webp`, and note its `artifact_id`. If you also want to confirm the fallback paths, use one non-image binary artifact plus, optionally, one `image/svg+xml` artifact for the current best-effort SVG behavior. Otherwise, skip this optional check.

2. Read the manager artifact workbench detail and note the authoritative `content_type` plus the existing metadata and raw-artifact actions.

```bash
curl http://localhost:8080/api/manager/artifacts/<artifact-id>/workbench
```

3. Open `http://localhost:8080/board/artifacts/workbench?artifact_id=<artifact-id>` in a web browser and inspect the preview there. This smoke is browser-only UI behavior inside the existing artifact workbench, so `curl` against the board route only verifies the static shell, not the rendered image preview or binary fallback.

Expected outcomes:

- binary/media preview stays inside `/board/artifacts/workbench?artifact_id=<artifact-id>` and does not add a new route or new manager API fields
- approved raster image preview renders inline for `image/png`, `image/jpeg`, `image/gif`, and `image/webp`
- `image/svg+xml` is best-effort under the current safety policy and may still fall back to the metadata/download binary fallback path
- non-image binary artifacts stay on the metadata/download binary fallback path with clear not-previewed-inline behavior and raw artifact actions still visible
- this smoke assumes a suitable existing artifact already exists and is optional/browser-only

## Artifact Long-Text Ergonomics Smoke (Optional Browser Check)

With `foreman serve` running, you can manually verify long-text ergonomics inside the existing artifact workbench if you already have a suitable artifact. This is browser-only UI behavior on the generic long-text path, so `curl` against the board route only verifies the shell, not the line-numbered collapsed preview.

1. If you already have an artifact that stays on the generic long-text path, note its `artifact_id`. Good candidates include `run_log`, `command_result`, or other long `text/plain` artifacts. If the artifact renders on the structured JSON, Markdown, or diff / patch path, or if the text is short, skip this optional check.

2. Read the manager artifact workbench detail and note the existing summary, preview, and truncation metadata that the browser uses for long-text ergonomics.

```bash
curl http://localhost:8080/api/manager/artifacts/<artifact-id>/workbench
```

3. Open `http://localhost:8080/board/artifacts/workbench?artifact_id=<artifact-id>` in a web browser and inspect the preview there.

Expected outcomes:

- long-text ergonomics stay inside `/board/artifacts/workbench?artifact_id=<artifact-id>` and do not add a new route or new manager API fields
- long text and log-like artifacts on the generic long-text path show line numbers, start in a collapsed first-screen teaser, and offer `Expand all` for the current bounded preview
- lightweight summary navigation is derived from existing summary and preview text, and following a hidden anchor may expand the bounded preview to reveal it
- JSON, Markdown, and diff / patch structured-renderer success paths remain separate from long-text ergonomics
- unsupported or short content still uses the simpler preview path
- truncation warnings remain visible in both collapsed and expanded states

## Artifact Compare Smoke (Optional Browser Check)

With `foreman serve` running, you can manually verify artifact compare if you already have two comparable text artifacts with the same `task_id` and `kind` across different runs. The earlier artifact workbench smokes do not guarantee that this history exists.

1. Identify a current text-like artifact whose immediately previous same-task same-kind artifact also exists. Note the current `artifact_id`. If the artifact is image/binary-only or has no earlier comparable history, this optional check will exercise `unsupported` or `no_previous` instead of the ready diff path.

2. Read the manager compare view.

```bash
curl http://localhost:8080/api/manager/artifacts/<artifact-id>/compare
```

3. Open the compare page in a browser.

```bash
curl http://localhost:8080/board/artifacts/compare?artifact_id=<artifact-id>
```

Expected outcomes:

- `GET /api/manager/artifacts/<artifact-id>/compare` returns the compare DTO with stable `current`, nullable `previous`, nullable `diff`, `limits`, `messages`, and `navigation`
- `GET /board/artifacts/compare?artifact_id=<artifact-id>` serves the dedicated compare page shell
- when a previous comparable text artifact exists, the browser shows a unified diff for the current artifact versus the immediately previous same-task same-kind artifact
- `Back to current artifact` and `Back to run workbench` stay available as the only compare-page navigation links besides `Refresh compare`
- if no previous comparable artifact exists, or the artifact type is unsupported, the compare page stays read-only and shows the corresponding manager-driven state instead of failing open or allowing manual history selection

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
- Run-detail workbench spec: [docs/superpowers/specs/2026-03-30-foreman-run-detail-workbench-design.md](/root/link/repo/docs/superpowers/specs/2026-03-30-foreman-run-detail-workbench-design.md)
- Run-detail workbench plan: [docs/superpowers/plans/2026-03-30-foreman-phase-2-run-detail-workbench.md](/root/link/repo/docs/superpowers/plans/2026-03-30-foreman-phase-2-run-detail-workbench.md)
- Artifact workbench spec: [docs/superpowers/specs/2026-03-31-foreman-artifact-workbench-design.md](/root/link/repo/docs/superpowers/specs/2026-03-31-foreman-artifact-workbench-design.md)
- Artifact workbench plan: [docs/superpowers/plans/2026-03-31-foreman-phase-2-artifact-workbench.md](/root/link/repo/docs/superpowers/plans/2026-03-31-foreman-phase-2-artifact-workbench.md)
