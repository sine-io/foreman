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
- HTTP board router and endpoint tests

The remaining work for Phase 1 is concentrated in:

- runtime wiring so `serve` hosts the board and gateway end-to-end
- richer CLI command integration on top of handlers
- final polish of the board UI and full runtime smoke checks

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
- [`docs/superpowers/plans/2026-03-27-foreman-go-phase-1.md`](/root/link/repo/docs/superpowers/plans/2026-03-27-foreman-go-phase-1.md): current implementation plan

## Quick Start

Prerequisites:

- Go `1.26+`

Current verification:

```bash
go test ./...
go run ./cmd/foreman --help
```

When the `serve` path grows beyond the bootstrap stub:

```bash
go run ./cmd/foreman serve
```

## Status Notes

- This repo no longer contains the legacy shell-runtime, hook, or skill-packaging line.
- Foreman should call native downstream CLIs through dedicated Go adapters instead of inheriting the old repository wrapper scripts.

## See Also

- [INSTALL.md](/root/link/repo/INSTALL.md)
- [CHANGELOG.md](/root/link/repo/CHANGELOG.md)
