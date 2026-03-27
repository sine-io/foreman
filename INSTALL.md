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

At the current bootstrap stage, the `serve` command exists but still represents only the earliest control-plane seam:

```bash
go run ./cmd/foreman serve
```

## Repository Purpose

This repository is now Foreman-only.

It intentionally excludes the previous shell-runtime and skill-packaging line. If you are looking for the earlier Codex tmux/runtime wrapper flow, that is no longer part of this codebase.

## Design and Plan

- Spec: [docs/superpowers/specs/2026-03-27-foreman-go-design.md](/root/link/repo/docs/superpowers/specs/2026-03-27-foreman-go-design.md)
- Plan: [docs/superpowers/plans/2026-03-27-foreman-go-phase-1.md](/root/link/repo/docs/superpowers/plans/2026-03-27-foreman-go-phase-1.md)
