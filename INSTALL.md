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

## Repository Purpose

This repository is now Foreman-only.

It intentionally excludes the previous shell-runtime and skill-packaging line. If you are looking for the earlier Codex tmux/runtime wrapper flow, that is no longer part of this codebase.

## Design and Plan

- Spec: [docs/superpowers/specs/2026-03-27-foreman-go-design.md](/root/link/repo/docs/superpowers/specs/2026-03-27-foreman-go-design.md)
- Plan: [docs/superpowers/plans/2026-03-27-foreman-go-phase-1.md](/root/link/repo/docs/superpowers/plans/2026-03-27-foreman-go-phase-1.md)
