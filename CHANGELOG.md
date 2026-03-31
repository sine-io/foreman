# Changelog

## 2026-03-27

### Repository Transition

- Renamed the active project direction to `Foreman`
- Split the repository away from the older shell-runtime/tooling line
- Removed the old Codex shell-runtime packaging direction from active development

### Design and Planning

- Added the approved Go design spec:
  - [docs/superpowers/specs/2026-03-27-foreman-go-design.md](/root/link/repo/docs/superpowers/specs/2026-03-27-foreman-go-design.md)
- Added the approved Go Phase 1 plan:
  - [docs/superpowers/plans/2026-03-27-foreman-go-phase-1.md](/root/link/repo/docs/superpowers/plans/2026-03-27-foreman-go-phase-1.md)

### Go Bootstrap

- Added `Foreman` Go module bootstrap
- Added config/bootstrap/runtime seam
- Added `cobra` root command and `serve`
- Added `zerolog` setup

### Phase 1 Progress

- Added domain aggregates and strict approval policy
- Added SQLite-backed repositories and artifact storage
- Added command handlers, query models, OpenClaw gateway, and Codex runner adapter
- Added HTTP board routes and end-to-end HTTP tests
- Wired `serve` to the real SQLite-backed app runtime and board endpoints
- Added bootstrap integration tests covering OpenClaw-to-board flow
- Wired CLI project/module/task commands to the real application handlers
- Wired the board UI to live module/task/approval HTTP data
- Validated the Phase 1 flow against a live `codex` CLI and persisted completed run artifacts

### Phase 2 Progress

- Added an application-level manager-agent contract and service
- Moved OpenClaw onto the manager-agent service path
- Added a Foreman-native manager HTTP API under `/api/manager/*`
- Wired the bootstrap/runtime path to expose the manager service

## 2026-03-28

### Phase 2 Approval Workbench

- Added approval workbench queue and detail guidance for manager review of risky tasks
- Added approval workbench smoke coverage for `GET /api/manager/projects/:id/approvals` and `GET /api/manager/approvals/:id`
- Documented `POST /approve`, `POST /reject`, and `POST /retry-dispatch` manager actions
- Recorded `approved_pending_dispatch` as the approval workbench recovery state for approved work that still needs dispatch recovery
- Linked the approval workbench spec and execution plan from the main docs

### Phase 2 Control-Plane Hardening

- Replaced ad-hoc SQLite bootstrap with ordered, idempotent migrations
- Added explicit `created_at` ordering metadata for runs, approvals, and artifacts
- Made latest run and approval lookups deterministic instead of relying on implicit SQLite row order
- Added a dedicated manager task-status query model backed by persisted task/run/approval state
- Added tx-bound SQLite repositories plus a transaction runner for command-side atomicity
- Made dispatch and approval transitions retry-safe under duplicate dispatch and approval retries
- Hardened completed-run retries so lease cleanup is retried instead of leaving write scopes stranded
- Documented repeated-dispatch and approval-gated smoke flows for the manager API

## 2026-03-29

### Phase 2 Task Workbench

- Linked the task-detail workbench spec and execution plan from the main docs
- Documented the board -> task workbench -> run detail operator flow
- Added task workbench smoke coverage for `GET /api/manager/tasks/:id/workbench`
- Documented task workbench action routes for `POST /dispatch`, `POST /retry`, `POST /cancel`, and `POST /reprioritize`
- Recorded that the task page links to approval workbench and run detail
- Recorded the no-approval task workbench state with disabled approval-workbench link reason `No approval history`

## 2026-03-31

### Phase 2 Artifact Workbench

- Linked the artifact workbench spec and execution plan from the main docs
- Documented the run workbench -> artifact workbench operator flow
- Added artifact workbench smoke coverage for `GET /api/manager/artifacts/:id/workbench`, raw-content coverage for `GET /api/manager/artifacts/:id/content`, and the board route at `/board/artifacts/workbench?artifact_id=<artifact-id>`
- Recorded that run workbench deep-links linked artifacts into artifact workbench while legacy run-page artifact anchors remain as a fallback
- Recorded that raw artifact content now streams with safe response headers

### Phase 2 Artifact Renderer Polish

- Documented renderer polish for JSON pretty-print, safe Markdown render, and diff / patch previews inside the existing artifact workbench
- Added a renderer-polish smoke path that reuses artifacts selected by `content_type`, `kind`, or `path`
- Recorded that malformed or unsupported preview content still falls back to the generic text preview

## 2026-03-30

### Phase 2 Run Workbench

- Linked the run-detail workbench spec and execution plan from the main docs
- Documented the task workbench -> run workbench operator flow
- Added run workbench smoke coverage for `GET /api/manager/runs/:id/workbench`
- Documented the canonical board run workbench route at `/board/runs/workbench?run_id=<run-id>`
- Recorded that legacy `/board/runs/:id` now redirects to `/board/runs/workbench?run_id=<run-id>`
