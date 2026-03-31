# Foreman

Foreman is a local embedded control plane for manager agents such as OpenClaw, Nanobot, and ZeroClaw.

Its job is to keep project truth, task state, approvals, leases, artifacts, and board views in one local system while upstream manager agents coordinate downstream workers such as Codex and Claude.

## Current State

This repository is now a standalone `Foreman` codebase.

What remains here is only Foreman-related content:

- Go bootstrap code under [`cmd/foreman`](/root/link/repo/cmd/foreman) and [`internal`](/root/link/repo/internal)
- Foreman architecture and implementation docs under [`docs/superpowers/specs`](/root/link/repo/docs/superpowers/specs) and [`docs/superpowers/plans`](/root/link/repo/docs/superpowers/plans)

The currently implemented slice now includes:

- Go module and binary bootstrap
- config/bootstrap/runtime seam
- `cobra` root command and `serve`
- `zerolog` setup
- domain aggregates and strict approval policy
- SQLite schema, repositories, and artifact filesystem store
- command and query handlers
- OpenClaw gateway normalization
- Codex runner adapter
- HTTP board router, dynamic board assets, and endpoint tests
- runtime wiring for `serve` with SQLite-backed board and gateway flow
- CLI project/module/task commands wired to real handlers
- a Foreman-native manager-agent API for upstream integrations
- deterministic manager-facing task status reconstruction from persisted task/run/approval state
- retry-safe dispatch and approval handling with tx-bound SQLite persistence
- approval workbench queue/detail/action endpoints, including `approved_pending_dispatch` retry-dispatch recovery
- task-detail workbench detail/action endpoints for per-task operator flow, including direct task dispatch, retry, cancel, and reprioritize controls
- run-detail workbench detail flow, including the canonical `/board/runs/workbench?run_id=<run-id>` route and legacy `/board/runs/:id` compatibility redirect
- artifact workbench detail flow, including run-workbench deep links to `/board/artifacts/workbench?artifact_id=<artifact-id>`, legacy run-page anchor fallback, safe raw-content streaming headers, renderer polish for JSON / Markdown / diff previews, and long-text ergonomics inside the existing artifact workbench

Phase 1 is now validated end-to-end, including a live smoke run against the real `codex` CLI.

Work beyond Phase 1 is now concentrated in:

- richer board controls and polish
- additional runner and gateway adapters

## Architecture Constraints

Foreman is designed around:

- `DDD Lite`
- light `CQRS`
- `Clean Architecture`
- `DIP`

Preferred packages when needed:

- `cobra`
- `viper`
- `zerolog`
- `gin`

These stay in outer layers. Domain and application code should not depend on framework packages.

## Repository Layout

- [`cmd/foreman`](/root/link/repo/cmd/foreman): binary entrypoint
- [`internal/bootstrap`](/root/link/repo/internal/bootstrap): config, runtime, app wiring
- [`internal/adapters`](/root/link/repo/internal/adapters): CLI / HTTP / gateway / runner adapters
- [`internal/infrastructure`](/root/link/repo/internal/infrastructure): logging and future store implementations
- [`docs/superpowers/specs/2026-03-27-foreman-go-design.md`](/root/link/repo/docs/superpowers/specs/2026-03-27-foreman-go-design.md): current approved design
- [`docs/superpowers/specs/2026-03-28-foreman-phase-2-boundary.md`](/root/link/repo/docs/superpowers/specs/2026-03-28-foreman-phase-2-boundary.md): Phase 2 architecture boundary
- [`docs/superpowers/specs/2026-03-28-foreman-approval-workbench-design.md`](/root/link/repo/docs/superpowers/specs/2026-03-28-foreman-approval-workbench-design.md): approval workbench design
- [`docs/superpowers/specs/2026-03-29-foreman-task-detail-workbench-design.md`](/root/link/repo/docs/superpowers/specs/2026-03-29-foreman-task-detail-workbench-design.md): task-detail workbench design
- [`docs/superpowers/specs/2026-03-30-foreman-run-detail-workbench-design.md`](/root/link/repo/docs/superpowers/specs/2026-03-30-foreman-run-detail-workbench-design.md): run-detail workbench design
- [`docs/superpowers/specs/2026-03-31-foreman-artifact-workbench-design.md`](/root/link/repo/docs/superpowers/specs/2026-03-31-foreman-artifact-workbench-design.md): artifact workbench design
- [`docs/superpowers/specs/2026-03-31-foreman-artifact-renderer-polish-design.md`](/root/link/repo/docs/superpowers/specs/2026-03-31-foreman-artifact-renderer-polish-design.md): artifact renderer polish design
- [`docs/superpowers/specs/2026-03-31-foreman-artifact-log-ergonomics-design.md`](/root/link/repo/docs/superpowers/specs/2026-03-31-foreman-artifact-log-ergonomics-design.md): artifact log ergonomics design
- [`docs/superpowers/plans/2026-03-27-foreman-go-phase-1.md`](/root/link/repo/docs/superpowers/plans/2026-03-27-foreman-go-phase-1.md): current implementation plan
- [`docs/superpowers/plans/2026-03-28-foreman-phase-2-manager-contract.md`](/root/link/repo/docs/superpowers/plans/2026-03-28-foreman-phase-2-manager-contract.md): first Phase 2 execution plan
- [`docs/superpowers/plans/2026-03-28-foreman-phase-2-control-plane-hardening.md`](/root/link/repo/docs/superpowers/plans/2026-03-28-foreman-phase-2-control-plane-hardening.md): second Phase 2 execution plan
- [`docs/superpowers/plans/2026-03-28-foreman-phase-2-approval-workbench.md`](/root/link/repo/docs/superpowers/plans/2026-03-28-foreman-phase-2-approval-workbench.md): third Phase 2 execution plan
- [`docs/superpowers/plans/2026-03-29-foreman-phase-2-task-detail-workbench.md`](/root/link/repo/docs/superpowers/plans/2026-03-29-foreman-phase-2-task-detail-workbench.md): fourth Phase 2 execution plan
- [`docs/superpowers/plans/2026-03-30-foreman-phase-2-run-detail-workbench.md`](/root/link/repo/docs/superpowers/plans/2026-03-30-foreman-phase-2-run-detail-workbench.md): fifth Phase 2 execution plan
- [`docs/superpowers/plans/2026-03-31-foreman-phase-2-artifact-workbench.md`](/root/link/repo/docs/superpowers/plans/2026-03-31-foreman-phase-2-artifact-workbench.md): sixth Phase 2 execution plan
- [`docs/superpowers/plans/2026-03-31-foreman-phase-2-artifact-renderer-polish.md`](/root/link/repo/docs/superpowers/plans/2026-03-31-foreman-phase-2-artifact-renderer-polish.md): seventh Phase 2 execution plan

## Quick Start

Prerequisites:

- Go `1.26+`

Current verification:

```bash
go test ./...
go run ./cmd/foreman --help
```

To run the local control plane:

```bash
go run ./cmd/foreman serve
```

To call the normalized manager-agent API directly:

```bash
curl -X POST http://localhost:8080/api/manager/commands \
  -H 'Content-Type: application/json' \
  -d '{"kind":"create_task","summary":"Summarize current project status"}'
```

To inspect persisted manager-facing task state:

```bash
curl http://localhost:8080/api/manager/tasks/<task-id>?project_id=demo
curl http://localhost:8080/api/manager/projects/demo/board
```

To re-dispatch an existing task without creating duplicate runs or approvals:

```bash
curl -X POST http://localhost:8080/api/manager/commands \
  -H 'Content-Type: application/json' \
  -d '{"kind":"dispatch_task","project_id":"demo","task_id":"<task-id>"}'
```

## Approval Workbench

Approval workbench spec and execution plan:

- [`docs/superpowers/specs/2026-03-28-foreman-approval-workbench-design.md`](/root/link/repo/docs/superpowers/specs/2026-03-28-foreman-approval-workbench-design.md)
- [`docs/superpowers/plans/2026-03-28-foreman-phase-2-approval-workbench.md`](/root/link/repo/docs/superpowers/plans/2026-03-28-foreman-phase-2-approval-workbench.md)

Approval workbench entry flow:

- Board entry: [`/board/approvals/workbench?project_id=demo`](/root/link/repo/board/approvals/workbench?project_id=demo)
- Manager API queue entry: `GET /api/manager/projects/demo/approvals`
- Risky `create_task` or `dispatch_task` requests return `kind: "approval_needed"` and leave the task in `waiting_approval`
- Load approval detail with `GET /api/manager/approvals/<approval-id>`
- Resolve the approval with `POST /api/manager/approvals/<approval-id>/approve` or `POST /api/manager/approvals/<approval-id>/reject`

Approval workbench recovery flow:

- `approved_pending_dispatch` means the approval is already granted, but Foreman still has to resume or recover dispatch safely
- Use `POST /api/manager/approvals/<approval-id>/retry-dispatch` to continue from that persisted recovery point instead of creating a new approval
- `retry-dispatch` is the recovery path for approved work that could not move directly from approval resolution back into runner execution

## Task-Detail Workbench

Task-detail workbench spec and execution plan:

- [`docs/superpowers/specs/2026-03-29-foreman-task-detail-workbench-design.md`](/root/link/repo/docs/superpowers/specs/2026-03-29-foreman-task-detail-workbench-design.md)
- [`docs/superpowers/plans/2026-03-29-foreman-phase-2-task-detail-workbench.md`](/root/link/repo/docs/superpowers/plans/2026-03-29-foreman-phase-2-task-detail-workbench.md)

Task workbench operator flow:

- Start on the board overview and open a task from its task card into the task-detail workbench.
- The task page acts as the operator hub between the board and deeper execution detail.
- From the task-detail workbench, open the latest run workbench when execution context or artifacts need deeper troubleshooting.
- That task-to-run hop now uses the canonical `/board/runs/workbench?run_id=<run-id>` route.
- The task page also links to the approval workbench when approval history exists.
- If no approval exists, the approval-workbench link stays visible but disabled with reason `No approval history`.

Task workbench manager endpoints:

- Detail: `GET /api/manager/tasks/<task-id>/workbench?project_id=demo`
- Dispatch: `POST /api/manager/tasks/<task-id>/dispatch?project_id=demo`
- Retry: `POST /api/manager/tasks/<task-id>/retry?project_id=demo`
- Cancel: `POST /api/manager/tasks/<task-id>/cancel?project_id=demo`
- Reprioritize: `POST /api/manager/tasks/<task-id>/reprioritize?project_id=demo`

## Run-Detail Workbench

Run-detail workbench spec and execution plan:

- [`docs/superpowers/specs/2026-03-30-foreman-run-detail-workbench-design.md`](/root/link/repo/docs/superpowers/specs/2026-03-30-foreman-run-detail-workbench-design.md)
- [`docs/superpowers/plans/2026-03-30-foreman-phase-2-run-detail-workbench.md`](/root/link/repo/docs/superpowers/plans/2026-03-30-foreman-phase-2-run-detail-workbench.md)

Run workbench operator flow:

- The run-detail workbench is the run-level troubleshooting page that sits beneath the task-detail workbench.
- Read the manager view with `GET /api/manager/runs/<run-id>/workbench`.
- Open the canonical board route at `/board/runs/workbench?run_id=<run-id>`.
- Legacy `/board/runs/<run-id>` remains available as a compatibility redirect to `/board/runs/workbench?run_id=<run-id>`.

## Artifact Workbench

Artifact workbench spec and execution plan:

- [`docs/superpowers/specs/2026-03-31-foreman-artifact-workbench-design.md`](/root/link/repo/docs/superpowers/specs/2026-03-31-foreman-artifact-workbench-design.md)
- [`docs/superpowers/plans/2026-03-31-foreman-phase-2-artifact-workbench.md`](/root/link/repo/docs/superpowers/plans/2026-03-31-foreman-phase-2-artifact-workbench.md)

Artifact workbench operator flow:

- The artifact workbench is the artifact-level inspection page reached from the run-detail workbench.
- From the run workbench, linked artifacts now deep-link to `/board/artifacts/workbench?artifact_id=<artifact-id>`.
- Older run-page artifact anchors remain available as a compatibility fallback for legacy links.
- Read the manager view with `GET /api/manager/artifacts/<artifact-id>/workbench`.
- Stream raw artifact bytes with `GET /api/manager/artifacts/<artifact-id>/content`, which now returns safe response headers for direct download or preview.
- Renderer polish stays inside this existing artifact workbench page; it does not add a new route.
- Artifacts whose `content_type`, `kind`, or `path` maps to JSON, Markdown, or diff / patch now get a more readable structured preview.
- Long text and log-like artifacts that stay on the generic long-text path now add line numbers, a collapsed first-screen teaser with `Expand all`, and lightweight summary navigation derived from existing summary and preview text.
- Unsupported text-like content, along with JSON previews that fail to parse, still fall back to the generic text preview.
- Short or otherwise simple text content stays on the simpler preview path without the heavier long-text ergonomics UI.

## Control-Plane Guarantees

- Manager task status is reconstructed from persisted task, run, and approval records instead of ad-hoc board columns.
- Latest run and approval lookups use explicit ordering metadata, so repeated reads do not depend on SQLite `rowid`.
- Re-dispatch is retry-safe: if an authoritative run already exists, Foreman returns the persisted state instead of re-invoking the runner.
- Approval creation and approval resolution are atomic. Repeated risky dispatches reuse the existing pending approval instead of creating duplicates.
- Approved work can pause in `approved_pending_dispatch` and resume later through the approval workbench `retry-dispatch` path.
- Completed-run retries also retry lease cleanup, so a transient release failure does not leave the write scope stranded permanently.

## Status Notes

- This repo no longer contains the legacy shell-runtime, hook, or skill-packaging line.
- Foreman should call native downstream CLIs through dedicated Go adapters instead of inheriting the old repository wrapper scripts.
- OpenClaw now routes through the same application-level manager-agent service used by the new Foreman-native manager API.

## See Also

- [INSTALL.md](/root/link/repo/INSTALL.md)
- [CHANGELOG.md](/root/link/repo/CHANGELOG.md)
