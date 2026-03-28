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
- [`docs/superpowers/plans/2026-03-27-foreman-go-phase-1.md`](/root/link/repo/docs/superpowers/plans/2026-03-27-foreman-go-phase-1.md): current implementation plan
- [`docs/superpowers/plans/2026-03-28-foreman-phase-2-manager-contract.md`](/root/link/repo/docs/superpowers/plans/2026-03-28-foreman-phase-2-manager-contract.md): first Phase 2 execution plan
- [`docs/superpowers/plans/2026-03-28-foreman-phase-2-control-plane-hardening.md`](/root/link/repo/docs/superpowers/plans/2026-03-28-foreman-phase-2-control-plane-hardening.md): second Phase 2 execution plan

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

## Control-Plane Guarantees

- Manager task status is reconstructed from persisted task, run, and approval records instead of ad-hoc board columns.
- Latest run and approval lookups use explicit ordering metadata, so repeated reads do not depend on SQLite `rowid`.
- Re-dispatch is retry-safe: if an authoritative run already exists, Foreman returns the persisted state instead of re-invoking the runner.
- Approval creation and approval resolution are atomic. Repeated risky dispatches reuse the existing pending approval instead of creating duplicates.
- Completed-run retries also retry lease cleanup, so a transient release failure does not leave the write scope stranded permanently.

## Status Notes

- This repo no longer contains the legacy shell-runtime, hook, or skill-packaging line.
- Foreman should call native downstream CLIs through dedicated Go adapters instead of inheriting the old repository wrapper scripts.
- OpenClaw now routes through the same application-level manager-agent service used by the new Foreman-native manager API.

## See Also

- [INSTALL.md](/root/link/repo/INSTALL.md)
- [CHANGELOG.md](/root/link/repo/CHANGELOG.md)
