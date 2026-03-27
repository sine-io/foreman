# Foreman Go Design

Date: 2026-03-27
Status: approved in interactive design review

## Summary

Foreman is a local embedded control plane that lets upstream manager agents such as OpenClaw, Nanobot, and ZeroClaw coordinate downstream CLI workers such as Codex and Claude. The manager agent remains the primary user-facing PM. Foreman owns project truth, task state, approvals, leases, artifacts, and board views.

Foreman is implemented as a single Go binary with strong internal boundaries:

- `DDD Lite`
- light `CQRS`
- `Clean Architecture`
- `DIP`

Preferred Go packages when needed:

- `gin`
- `zerolog`
- `cobra`
- `viper`

## Naming

Project name:

- `Foreman`

Rationale:

- captures the PM / crew-chief / work-coordination role
- better reflects project intent than generic names such as `ai-orch`
- fits the control-plane + runner + board model

## Goals

- Let manager agents drive Codex, Claude, and future runners through a stable local control plane
- Keep project truth outside chat history
- Support users with only Codex, only Claude, or both
- Support customized runners with skills, MCPs, and local extensions
- Provide a persistent board that starts in brainstorming and continues through implementation
- Keep approvals, retries, cancellation, and reprioritization as first-class commands
- Start as a single local binary without blocking later extraction into multiple processes

## Non-Goals

- Full multi-user RBAC in v1
- Event sourcing in v1
- Separate write and read databases in v1
- Treating Telegram or other chat transports as Foreman-native first-class entrypoints
- Making the board a full admin console in v1

## Top-Level Shape

Foreman is not the human-facing PM. It is the local embedded control plane called by upstream manager agents.

### Primary upstream entrypoints

- OpenClaw
- Nanobot
- ZeroClaw
- similar PM-style manager agents

### Secondary entrypoints

- local CLI
- local board UI

### Explicitly not first-class entrypoints

- Telegram
- Discord
- WhatsApp

Those are treated as transport capabilities of upstream manager agents, not Foreman-native channels.

## Architecture

### One Go binary, strong internal boundaries

Foreman starts as a single Go executable for ease of self-use, open-source distribution, and local deployment. Internally it is split into clear layers so the binary can be decomposed later if needed.

### Embedded control plane

The binary contains:

- command handlers
- query handlers
- orchestrator logic
- board HTTP surface
- local API surface
- manager-agent gateway adapters
- runner adapters

## Required Design Constraints

### DDD Lite

Use explicit aggregates and invariants, but avoid heavy tactical DDD overhead.

Recommended aggregates:

- `Project`
- `Module`
- `Task`
- `Approval`
- `Lease`

`Run` and `Artifact` are important records, but they do not need to become heavyweight aggregates unless later behavior proves it necessary.

### Light CQRS

Separate command and query flows, but keep one database at first.

Command side is responsible for:

- create project
- create module
- create task
- split task
- dispatch task
- approve task
- retry task
- cancel task
- reprioritize task

Query side is responsible for:

- module board view
- task board view
- run detail view
- approval queue
- artifact summaries

CQRS in v1 means separate handlers and read models, not separate storage engines.

### Clean Architecture

Dependency direction always points inward.

- adapters depend on application/domain abstractions
- domain depends on no framework packages
- infrastructure implements ports
- UI / transport layers do not contain business rules

### DIP

The application layer depends on interfaces such as:

- `ProjectRepository`
- `ModuleRepository`
- `TaskRepository`
- `LeaseRepository`
- `ApprovalRepository`
- `RunnerPort`
- `ManagerAgentGateway`
- `ArtifactStore`

Adapters implement those interfaces.

## Preferred Package Layout

```text
cmd/foreman/
internal/bootstrap/
internal/domain/
  project/
  module/
  task/
  approval/
  lease/
  policy/
internal/app/command/
internal/app/query/
internal/ports/
internal/adapters/http/
internal/adapters/cli/
internal/adapters/gateway/
  manageragent/
  openclaw/
  nanobot/
  zeroclaw/
internal/adapters/runner/
  codex/
  claude/
internal/infrastructure/store/sqlite/
internal/infrastructure/store/artifactfs/
internal/infrastructure/logging/
web/board/
```

### Package placement rules

- `cobra` only in CLI adapters
- `gin` only in HTTP / board adapters
- `viper` only in bootstrap/config adapters
- `zerolog` only in logging adapters

These packages must not leak into domain or application logic.

## Core Domain Model

### Project

Owns:

- repo identity
- default policy profile
- upstream manager-agent profile
- module tree

### Module

Represents a board-visible implementation area. A module is not just a task bucket; it is the main progress rollup unit for the board.

Owns:

- name
- description
- module board state
- completion criteria
- task rollup summary

### Task

Executable work unit.

Owns:

- task type
- acceptance criteria
- priority
- write scope
- current state
- linked module
- linked run summary

### Approval

Tracks risk-gated actions requiring human or manager-agent decision.

### Lease

Enforces the one-writer rule for repo/worktree/write-scope ownership.

### Write scope model for v1

To keep Phase 1 planning and schema design bounded, Foreman v1 uses a small canonical write-scope model:

- `repo:<project_id>`
- `module:<module_id>`
- `task:<task_id>`

Rules:

- the default writable scope in v1 is `repo:<project_id>`
- helper/read-only work does not acquire a write lease
- a task may request a narrower scope later, but Phase 1 should assume repo-level leasing unless a task is explicitly marked otherwise
- only one active writable lease may exist for a given scope key at a time

This intentionally avoids path-level locking in v1.

## Board Model

Foreman has one underlying state model and two board projections.

### Module board

Recommended columns:

- `Backlog`
- `Designed`
- `Implementing`
- `Review`
- `Done`

Purpose:

- answer “what parts exist?”
- answer “which implementation areas are blocked or progressing?”
- connect brainstorming outputs to implementation state

### Task board

Recommended columns:

- `Ready`
- `Running`
- `Waiting Approval`
- `Review / Verify`
- `Done`

Purpose:

- answer “what is executing right now?”
- expose runnable, blocked, and completed work
- provide light interaction

### Board interactions

Supported in v1:

- approve
- retry
- cancel
- reprioritize

Not supported in v1:

- full project creation/editing inside the board
- deep admin operations

## Brainstorming Board Integration

The brainstorming companion and the runtime system should use one conceptual model.

### During brainstorming

The board is a design companion:

- module placeholders
- design status
- scope visibility

### During implementation

The Go program becomes the source of truth:

- persisted modules
- persisted tasks
- persisted runs
- persisted approvals

The visual structure stays recognizable across both phases.

## Data Model

Recommended persistent records:

- `projects`
- `modules`
- `tasks`
- `runs`
- `approvals`
- `leases`
- `artifacts`
- `worker_capabilities`
- board-oriented read models or projections

### Structured truth

Use `SQLite` for:

- domain records
- state transitions
- run metadata
- approval queue
- leases
- read model projections

### Raw evidence

Use filesystem storage for:

- logs
- assistant summaries
- diffs
- test outputs
- screenshots
- reports

SQLite stores references and summaries, not large raw blobs.

## Manager-Agent Integration Flow

Recommended flow:

1. Upstream manager agent sends a goal or action
2. Gateway adapter normalizes it into a Foreman command DTO
3. Command handler validates domain rules and policy
4. Foreman mutates state
5. Orchestrator dispatches work or pauses for approval
6. Query side exposes updated state back to board and upstream manager agent

### Important rule

Manager-agent adapters translate protocol and identity metadata only. They do not own scheduling, policy, or domain rules.

## Runner Model

Foreman routes work to downstream runners through interfaces rather than vendor-specific logic.

Recommended runner adapters:

- Codex adapter
- Claude adapter
- future custom adapters

Worker selection is capability-based, not vendor-name-based.

## Policy and Execution Model

### Default approval mode

- strict

### Phase 1 approval trigger matrix

In Phase 1, strict mode must pause execution for these categories:

- destructive shell actions
  - examples: broad deletion, `rm -rf`, forceful cleanup of unknown paths
- outward side effects
  - examples: `git push`, release, deploy, publishing
- non-read-only network actions initiated by a runner
- escalation from read-only helper work to writable execution

In Phase 1, strict mode does not need to pause for these categories if they stay local:

- reading files
- local search
- local build/test/lint commands
- local code edits inside the already leased writable scope

This matrix is intentionally narrow so the approval state machine and board behavior can be planned precisely.

### Override scopes

- global
- project
- task
- session

### Core execution rule

Many read-only helpers may run in parallel, but only one writable worker may hold the active lease for a given write scope at a time.

### Phase 1 persisted system truth

Phase 1 persistence is not limited to projects/modules/tasks alone. The first vertical slice must persist:

- projects
- modules
- tasks
- runs
- approvals
- leases
- artifact index records
- board-oriented read models or projections

Large raw logs and files still live in the filesystem, but their indexed metadata is part of persisted system truth.

## Phase 1 Recommendation

The first Go slice should be intentionally narrow:

1. one Go binary
2. SQLite-backed project/module/task/run/approval/lease store
3. artifact index plus board read model
4. one manager-agent gateway, starting with OpenClaw
5. one writable runner adapter, starting with Codex
6. strict approval flow using the Phase 1 trigger matrix above
7. light board interactions

### Exact Phase 1 execution path

The first end-to-end slice should be:

1. OpenClaw sends a command to Foreman
2. Foreman gateway normalizes it into a command DTO
3. command side creates or updates project/module/task state
4. orchestrator allocates a writable lease if dispatch is allowed
5. Codex adapter starts a run
6. Foreman persists run state, approval records, lease state, and artifact metadata
7. query side updates board views and status payloads
8. OpenClaw receives summary / approval-needed / completion responses from Foreman

Deferred:

- Claude adapter
- Nanobot adapter
- ZeroClaw adapter
- richer board controls
- process splitting

## Why This Is Better Than the Previous Python Line

The earlier Python implementation path proved the control-plane shape, but the project constraints have now changed:

- implementation language must be Go
- architecture must follow DDD Lite + CQRS + Clean Arch + DIP
- board integration is now a formal product requirement
- manager-agent integration is first-class

Therefore the previous Python worktree should be treated as a disposable exploration branch, not the current implementation baseline.
