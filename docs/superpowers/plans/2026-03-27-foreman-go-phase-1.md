# Foreman Go Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first working Go vertical slice of Foreman: a single local binary that accepts OpenClaw-style commands, persists project/module/task/run/approval/lease state in SQLite, dispatches a writable Codex run, and exposes a light board with approve/retry/cancel/reprioritize actions.

**Architecture:** Foreman is a single Go binary with internal `DDD Lite + light CQRS + Clean Arch + DIP` boundaries. The command side owns project/module/task mutations and orchestration decisions; the query side serves module-board/task-board/run-detail projections from SQLite. The initial Codex adapter should invoke the native `codex` CLI directly and persist run state inside Foreman rather than relying on legacy wrapper scripts.

**Tech Stack:** Go, `cobra`, `viper`, `zerolog`, `gin`, SQLite, native `codex` CLI, Go `embed`

---

## Scope Guard

This plan supersedes the earlier Python-oriented plan in this repository. Do not continue the Python implementation line.

This Phase 1 plan includes only:

- one Go binary: `foreman`
- one upstream manager-agent gateway: OpenClaw
- one downstream writable runner: Codex
- SQLite-backed command and query models
- filesystem-backed artifact storage
- strict approval flow using the approved trigger matrix
- one board UI with light interactions

Explicitly out of scope for this plan:

- Claude runner implementation
- Nanobot or ZeroClaw gateway implementations
- multi-user RBAC
- separate read/write databases
- event sourcing
- full admin UI inside the board

## File Structure

### Binary and bootstrap

- Create: `go.mod`
  Responsibility: Go module definition and dependency versions.
- Create: `cmd/foreman/main.go`
  Responsibility: start the single binary and delegate wiring to bootstrap.
- Create: `internal/bootstrap/config.go`
  Responsibility: load config from env/files using `viper`.
- Create: `internal/bootstrap/app.go`
  Responsibility: wire repositories, adapters, command handlers, query handlers, and router.
- Create: `internal/bootstrap/runtime.go`
  Responsibility: create runtime directories, SQLite location, artifact root, and existing-script locations.

### Domain

- Create: `internal/domain/project/project.go`
  Responsibility: project aggregate and project invariants.
- Create: `internal/domain/module/module.go`
  Responsibility: module aggregate and module board states.
- Create: `internal/domain/task/task.go`
  Responsibility: task aggregate, task states, task types, write-scope model.
- Create: `internal/domain/approval/approval.go`
  Responsibility: approval aggregate and approval statuses.
- Create: `internal/domain/lease/lease.go`
  Responsibility: lease aggregate and one-writer invariants.
- Create: `internal/domain/policy/policy.go`
  Responsibility: strict approval trigger matrix and policy decisions.

### Application command side

- Create: `internal/app/command/create_project.go`
  Responsibility: command handler for project creation.
- Create: `internal/app/command/create_module.go`
  Responsibility: command handler for module creation.
- Create: `internal/app/command/create_task.go`
  Responsibility: command handler for task creation.
- Create: `internal/app/command/dispatch_task.go`
  Responsibility: orchestrate writable dispatch to Codex with lease acquisition.
- Create: `internal/app/command/approve_task.go`
  Responsibility: approve pending action and resume progression.
- Create: `internal/app/command/retry_task.go`
  Responsibility: retry failed or blocked task.
- Create: `internal/app/command/cancel_task.go`
  Responsibility: cancel runnable or blocked task.
- Create: `internal/app/command/reprioritize_task.go`
  Responsibility: change task priority.

### Application query side

- Create: `internal/app/query/module_board.go`
  Responsibility: module board read model and query handler.
- Create: `internal/app/query/task_board.go`
  Responsibility: task board read model and query handler.
- Create: `internal/app/query/run_detail.go`
  Responsibility: run status, artifact summary, and approval status query handler.
- Create: `internal/app/query/approval_queue.go`
  Responsibility: approval queue read model and query handler.

### Ports

- Create: `internal/ports/repositories.go`
  Responsibility: repository interfaces for command-side aggregates and query-side projections.
- Create: `internal/ports/runners.go`
  Responsibility: runner adapter interface for dispatch/observe/stop.
- Create: `internal/ports/gateways.go`
  Responsibility: upstream manager-agent gateway interface.
- Create: `internal/ports/artifacts.go`
  Responsibility: artifact storage/index interface.

### Infrastructure storage

- Create: `internal/infrastructure/store/sqlite/db.go`
  Responsibility: SQLite connection management, PRAGMA setup, transactions.
- Create: `internal/infrastructure/store/sqlite/migrations/001_init.sql`
  Responsibility: initial schema for projects/modules/tasks/runs/approvals/leases/artifacts/board projections.
- Create: `internal/infrastructure/store/sqlite/project_repo.go`
  Responsibility: `ProjectRepository` implementation.
- Create: `internal/infrastructure/store/sqlite/module_repo.go`
  Responsibility: `ModuleRepository` implementation.
- Create: `internal/infrastructure/store/sqlite/task_repo.go`
  Responsibility: `TaskRepository` implementation.
- Create: `internal/infrastructure/store/sqlite/run_repo.go`
  Responsibility: `RunRepository` implementation.
- Create: `internal/infrastructure/store/sqlite/artifact_repo.go`
  Responsibility: `ArtifactRepository` implementation for persisted artifact index rows.
- Create: `internal/infrastructure/store/sqlite/approval_repo.go`
  Responsibility: `ApprovalRepository` implementation.
- Create: `internal/infrastructure/store/sqlite/lease_repo.go`
  Responsibility: `LeaseRepository` implementation.
- Create: `internal/infrastructure/store/sqlite/board_query_repo.go`
  Responsibility: read-model queries for module board, task board, approval queue, run detail.
- Create: `internal/infrastructure/store/artifactfs/store.go`
  Responsibility: filesystem artifact storage and indexing metadata bridge.

### Adapters

- Create: `internal/adapters/cli/root.go`
  Responsibility: root `cobra` command and top-level wiring.
- Create: `internal/adapters/cli/serve.go`
  Responsibility: `serve` command that starts the embedded control plane.
- Create: `internal/adapters/cli/project.go`
  Responsibility: project/module/task creation commands.
- Create: `internal/adapters/cli/action.go`
  Responsibility: approve/retry/cancel/reprioritize commands.
- Create: `internal/adapters/http/router.go`
  Responsibility: `gin` router and middleware setup.
- Create: `internal/adapters/http/board_handlers.go`
  Responsibility: board view endpoints and light board action endpoints.
- Create: `internal/adapters/http/dto.go`
  Responsibility: request/response DTOs for HTTP adapter only.
- Create: `internal/adapters/gateway/manageragent/types.go`
  Responsibility: common upstream manager-agent envelope and normalization types.
- Create: `internal/adapters/gateway/openclaw/handler.go`
  Responsibility: OpenClaw command ingestion, command dispatch bridging, and outbound response emission.
- Create: `internal/adapters/runner/codex/adapter.go`
  Responsibility: Codex runner adapter over the native `codex` CLI.
- Create: `internal/adapters/runner/codex/session.go`
  Responsibility: translate script status/session metadata into runner-state DTOs.

### Logging and board assets

- Create: `internal/infrastructure/logging/logger.go`
  Responsibility: `zerolog` setup isolated behind a small interface.
- Create: `web/board/index.html`
  Responsibility: board shell.
- Create: `web/board/app.js`
  Responsibility: render module/task board and light interactions.
- Create: `web/board/styles.css`
  Responsibility: board styling.

### Tests

- Create: `internal/domain/task/task_test.go`
  Responsibility: domain-level task state and write-scope tests.
- Create: `internal/domain/policy/policy_test.go`
  Responsibility: approval trigger matrix tests.
- Create: `internal/infrastructure/store/sqlite/db_test.go`
  Responsibility: schema bootstrap and constraint tests.
- Create: `internal/infrastructure/store/sqlite/task_repo_test.go`
  Responsibility: task/module/run/lease persistence tests.
- Create: `internal/app/command/dispatch_task_test.go`
  Responsibility: dispatch orchestration and approval behavior tests.
- Create: `internal/app/query/board_query_test.go`
  Responsibility: board projection tests.
- Create: `internal/adapters/gateway/openclaw/handler_test.go`
  Responsibility: manager-agent normalization tests.
- Create: `internal/adapters/runner/codex/adapter_test.go`
  Responsibility: adapter/script integration tests with fake scripts.
- Create: `internal/adapters/http/board_handlers_test.go`
  Responsibility: board and action endpoint tests.
- Create: `test/e2e_phase1_test.go`
  Responsibility: full vertical slice test from OpenClaw-style command to board-visible result.

### Documentation

- Modify: `README.md`
  Responsibility change: explain Foreman, Go architecture, and current Phase 1 status.
- Modify: `INSTALL.md`
  Responsibility change: add Go setup and Foreman startup.
- Modify: `CHANGELOG.md`
  Responsibility change: add Foreman repo split and Go bootstrap status.

## Task 1: Bootstrap the Go Binary and Dependency Boundaries

**Files:**
- Create: `go.mod`
- Create: `cmd/foreman/main.go`
- Create: `internal/bootstrap/config.go`
- Create: `internal/bootstrap/app.go`
- Create: `internal/bootstrap/runtime.go`
- Create: `internal/infrastructure/logging/logger.go`
- Create: `internal/adapters/cli/root.go`
- Create: `internal/adapters/cli/serve.go`
- Test: `internal/bootstrap/config_test.go`
- Test: `internal/adapters/cli/root_test.go`

- [ ] **Step 1: Write the failing bootstrap tests**

```go
func TestLoadConfigUsesDefaultRuntimeRoot(t *testing.T) {
    cfg, err := LoadConfig()
    require.NoError(t, err)
    require.Contains(t, cfg.RuntimeRoot, ".foreman")
}

func TestRootCommandRequiresSubcommand(t *testing.T) {
    cmd := NewRootCommand(&fakeApp{})
    cmd.SetArgs([]string{})
    err := cmd.Execute()
    require.Error(t, err)
}

func TestServeCommandCallsAppServe(t *testing.T) {
    app := &fakeApp{}
    cmd := NewRootCommand(app)
    cmd.SetArgs([]string{"serve"})
    err := cmd.Execute()
    require.NoError(t, err)
    require.True(t, app.serveCalled)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/bootstrap ./internal/adapters/cli`
Expected: FAIL with missing packages/types

- [ ] **Step 3: Implement the minimal binary, config loading, logger, and root command**

```go
type Config struct {
    RuntimeRoot string
    DBPath      string
    ArtifactRoot string
}

type App interface {
    Serve(context.Context) error
}

type app struct {
    Config Config
}

func BuildApp(cfg Config) (App, error) {
    return &app{Config: cfg}, nil
}

func (a *app) Serve(ctx context.Context) error {
    return nil
}

func NewRootCommand(app App) *cobra.Command {
    cmd := &cobra.Command{
        Use:   "foreman",
        Short: "Foreman embedded control plane",
        RunE: func(cmd *cobra.Command, args []string) error {
            _ = cmd.Help()
            return fmt.Errorf("subcommand required")
        },
    }
    cmd.SilenceUsage = true
    cmd.SilenceErrors = true
    cmd.AddCommand(newServeCommand(app))
    return cmd
}

func newServeCommand(app App) *cobra.Command {
    return &cobra.Command{
        Use:   "serve",
        Short: "Start the embedded control plane",
        RunE: func(cmd *cobra.Command, args []string) error {
            return app.Serve(cmd.Context())
        },
    }
}

func main() {
    cfg, err := bootstrap.LoadConfig()
    if err != nil {
        log.Fatal().Err(err).Msg("load config")
    }
    app, err := bootstrap.BuildApp(cfg)
    if err != nil {
        log.Fatal().Err(err).Msg("build app")
    }
    if err := cli.NewRootCommand(app).Execute(); err != nil {
        log.Fatal().Err(err).Msg("run command")
    }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/bootstrap ./internal/adapters/cli`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add go.mod cmd/foreman/main.go internal/bootstrap/config.go internal/bootstrap/app.go internal/bootstrap/runtime.go internal/infrastructure/logging/logger.go internal/adapters/cli/root.go internal/adapters/cli/serve.go internal/bootstrap/config_test.go internal/adapters/cli/root_test.go
git commit -m "feat: bootstrap foreman go binary"
```

## Task 2: Define Domain Aggregates and Policy Rules

**Files:**
- Create: `internal/domain/project/project.go`
- Create: `internal/domain/module/module.go`
- Create: `internal/domain/task/task.go`
- Create: `internal/domain/approval/approval.go`
- Create: `internal/domain/lease/lease.go`
- Create: `internal/domain/policy/policy.go`
- Test: `internal/domain/task/task_test.go`
- Test: `internal/domain/policy/policy_test.go`

- [ ] **Step 1: Write failing domain tests**

```go
func TestTaskAllowsReadyToRunningPathThroughLease(t *testing.T) {
    task := NewTask(
        "task-1",
        "module-1",
        TaskTypeWrite,
        "Add SQLite store",
        "repo:project-1",
    )
    require.True(t, task.CanTransition(TaskStateLeased))
}

func TestStrictPolicyRequiresApprovalForGitPush(t *testing.T) {
    decision := EvaluateStrictAction("git push origin main")
    require.True(t, decision.RequiresApproval)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/domain/...`
Expected: FAIL with missing domain types

- [ ] **Step 3: Implement aggregates, enums, and approval matrix**

```go
type TaskState string

const (
    TaskStateReady           TaskState = "ready"
    TaskStateRunning         TaskState = "running"
    TaskStateWaitingApproval TaskState = "waiting_approval"
)

type Decision struct {
    RequiresApproval bool
    Reason           string
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/domain/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/domain/project/project.go internal/domain/module/module.go internal/domain/task/task.go internal/domain/approval/approval.go internal/domain/lease/lease.go internal/domain/policy/policy.go internal/domain/task/task_test.go internal/domain/policy/policy_test.go
git commit -m "feat: add foreman domain aggregates"
```

## Task 3: Add Ports and SQLite Persistence Foundation

**Files:**
- Create: `internal/ports/repositories.go`
- Create: `internal/ports/runners.go`
- Create: `internal/ports/gateways.go`
- Create: `internal/ports/artifacts.go`
- Create: `internal/infrastructure/store/sqlite/db.go`
- Create: `internal/infrastructure/store/sqlite/migrations/001_init.sql`
- Create: `internal/infrastructure/store/sqlite/project_repo.go`
- Create: `internal/infrastructure/store/sqlite/module_repo.go`
- Create: `internal/infrastructure/store/sqlite/task_repo.go`
- Create: `internal/infrastructure/store/sqlite/run_repo.go`
- Create: `internal/infrastructure/store/sqlite/artifact_repo.go`
- Create: `internal/infrastructure/store/sqlite/approval_repo.go`
- Create: `internal/infrastructure/store/sqlite/lease_repo.go`
- Create: `internal/infrastructure/store/artifactfs/store.go`
- Test: `internal/infrastructure/store/sqlite/db_test.go`
- Test: `internal/infrastructure/store/sqlite/task_repo_test.go`

- [ ] **Step 1: Write failing persistence tests**

```go
func TestMigrationsCreateCoreTables(t *testing.T) {
    db := OpenTestDB(t)
    requireTable(t, db, "projects")
    requireTable(t, db, "modules")
    requireTable(t, db, "tasks")
    requireTable(t, db, "leases")
    requireTable(t, db, "artifacts")
}

func TestOnlyOneActiveLeaseCanExistForScope(t *testing.T) {
    repo := NewLeaseRepository(testDB)
    err := repo.Acquire(scope)
    require.NoError(t, err)
    err = repo.Acquire(scope)
    require.Error(t, err)
}

func TestArtifactIndexRoundTrip(t *testing.T) {
    repo := NewArtifactRepository(testDB)
    id, err := repo.Create(taskID, "assistant_summary", "artifacts/tasks/task-1/assistant.txt")
    require.NoError(t, err)
    row, err := repo.Get(id)
    require.NoError(t, err)
    require.Equal(t, "assistant_summary", row.Kind)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/infrastructure/store/sqlite`
Expected: FAIL with missing migrations/repos

- [ ] **Step 3: Implement ports, migrations, repos, and artifact store**

```sql
create table projects (
  id text primary key,
  name text not null,
  repo_root text not null
);
create table modules (
  id text primary key,
  project_id text not null references projects(id),
  name text not null,
  board_state text not null
);
create table tasks (
  id text primary key,
  module_id text not null references modules(id),
  task_type text not null,
  state text not null,
  write_scope text not null
);
create table runs (
  id text primary key,
  task_id text not null references tasks(id),
  runner_kind text not null,
  state text not null
);
create table approvals (
  id text primary key,
  task_id text not null references tasks(id),
  reason text not null,
  state text not null
);
create table artifacts (
  id text primary key,
  task_id text not null references tasks(id),
  kind text not null,
  path text not null,
  summary text not null default ''
);
create table leases (
  id text primary key,
  task_id text not null references tasks(id),
  scope_key text not null,
  state text not null
);
create unique index leases_active_scope_idx on leases(scope_key) where state = 'active';
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/infrastructure/store/sqlite`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ports/*.go internal/infrastructure/store/sqlite/db.go internal/infrastructure/store/sqlite/migrations/001_init.sql internal/infrastructure/store/sqlite/*.go internal/infrastructure/store/artifactfs/store.go internal/infrastructure/store/sqlite/db_test.go internal/infrastructure/store/sqlite/task_repo_test.go
git commit -m "feat: add foreman sqlite persistence"
```

## Task 4: Implement Command Handlers for Project, Module, and Task Lifecycle

**Files:**
- Create: `internal/app/command/create_project.go`
- Create: `internal/app/command/create_module.go`
- Create: `internal/app/command/create_task.go`
- Create: `internal/app/command/approve_task.go`
- Create: `internal/app/command/retry_task.go`
- Create: `internal/app/command/cancel_task.go`
- Create: `internal/app/command/reprioritize_task.go`
- Test: `internal/app/command/task_commands_test.go`

- [ ] **Step 1: Write failing command-handler tests**

```go
func TestCreateTaskPersistsReadyTask(t *testing.T) {
    handler := NewCreateTaskHandler(fakeTaskRepo{})
    out, err := handler.Handle(CreateTaskCommand{
        ModuleID:    "module-1",
        Title:       "Implement board query",
        TaskType:    "write",
        WriteScope:  "repo:project-1",
        Acceptance:  "Board query returns module columns",
        Priority:    10,
    })
    require.NoError(t, err)
    require.Equal(t, "ready", out.State)
}

func TestApproveTaskMarksApprovalResolved(t *testing.T) {
    handler := NewApproveTaskHandler(fakeApprovalRepo{}, fakeTaskRepo{})
    err := handler.Handle(ApproveTaskCommand{TaskID: id})
    require.NoError(t, err)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/app/command`
Expected: FAIL with missing handlers

- [ ] **Step 3: Implement minimal command handlers**

```go
type CreateTaskHandler struct {
    Tasks ports.TaskRepository
}

func (h *CreateTaskHandler) Handle(cmd CreateTaskCommand) (TaskDTO, error) {
    task, err := h.Tasks.Create(cmd.ModuleID, cmd.Title, cmd.TaskType, cmd.WriteScope, cmd.Acceptance, cmd.Priority)
    if err != nil {
        return TaskDTO{}, err
    }
    return TaskDTO{ID: task.ID, State: string(task.State)}, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/app/command`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/command/*.go internal/app/command/task_commands_test.go
git commit -m "feat: add foreman command handlers"
```

## Task 5: Implement Query Handlers and Board Read Models

**Files:**
- Create: `internal/app/query/module_board.go`
- Create: `internal/app/query/task_board.go`
- Create: `internal/app/query/run_detail.go`
- Create: `internal/app/query/approval_queue.go`
- Create: `internal/infrastructure/store/sqlite/board_query_repo.go`
- Test: `internal/app/query/board_query_test.go`

- [ ] **Step 1: Write failing board-query tests**

```go
func TestModuleBoardGroupsModulesByBoardState(t *testing.T) {
    query := NewModuleBoardQuery(fakeBoardReadRepo{})
    view, err := query.Execute(projectID)
    require.NoError(t, err)
    require.Contains(t, view.Columns, "Implementing")
}

func TestTaskBoardShowsPendingApprovals(t *testing.T) {
    query := NewTaskBoardQuery(fakeBoardReadRepo{})
    view, err := query.Execute(projectID)
    require.NoError(t, err)
    require.NotEmpty(t, view.Columns["Waiting Approval"])
}

func TestRunDetailIncludesArtifactSummaries(t *testing.T) {
    query := NewRunDetailQuery(fakeBoardReadRepo{})
    view, err := query.Execute(runID)
    require.NoError(t, err)
    require.NotEmpty(t, view.Artifacts)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/app/query`
Expected: FAIL with missing query handlers

- [ ] **Step 3: Implement query-side projections**

```go
type ModuleBoardView struct {
    Columns map[string][]ModuleCard
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/app/query`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/query/*.go internal/infrastructure/store/sqlite/board_query_repo.go internal/app/query/board_query_test.go
git commit -m "feat: add foreman board query models"
```

## Task 6: Add OpenClaw Gateway Adapter

**Files:**
- Create: `internal/adapters/gateway/manageragent/types.go`
- Create: `internal/adapters/gateway/openclaw/handler.go`
- Test: `internal/adapters/gateway/openclaw/handler_test.go`

- [ ] **Step 1: Write failing gateway tests**

```go
func TestOpenClawEnvelopeMapsToCreateTaskCommand(t *testing.T) {
    cmd, err := DecodeEnvelope(payload)
    require.NoError(t, err)
    require.Equal(t, "create_task", cmd.Kind)
}

func TestOpenClawEncodesApprovalNeededResponse(t *testing.T) {
    msg, err := EncodeResponse(Response{
        Kind:    "approval_needed",
        TaskID:  "task-1",
        Summary: "git push origin main requires approval",
    })
    require.NoError(t, err)
    require.Contains(t, string(msg), "approval_needed")
}

func TestOpenClawHandlerReturnsCompletionResponse(t *testing.T) {
    handler := NewHandler(fakeCommandBus{}, fakeQueryBus{})
    resp, err := handler.Handle(context.Background(), Envelope{
        SessionID: "oc-session-1",
        Action:    "create_task",
    })
    require.NoError(t, err)
    require.NotEmpty(t, resp.Kind)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/adapters/gateway/openclaw`
Expected: FAIL with missing gateway types/handler

- [ ] **Step 3: Implement minimal OpenClaw normalization**

```go
type Envelope struct {
    SessionID string `json:"session_id"`
    Action    string `json:"action"`
}

type Response struct {
    Kind    string `json:"kind"`
    TaskID  string `json:"task_id"`
    Summary string `json:"summary"`
}

func (h *Handler) Handle(ctx context.Context, env Envelope) (Response, error) {
    cmd, err := DecodeEnvelope(env)
    if err != nil {
        return Response{}, err
    }
    result, err := h.Commands.Dispatch(ctx, cmd)
    if err != nil {
        return Response{}, err
    }
    return EncodeDomainResult(result), nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/adapters/gateway/openclaw`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/adapters/gateway/manageragent/types.go internal/adapters/gateway/openclaw/handler.go internal/adapters/gateway/openclaw/handler_test.go
git commit -m "feat: add openclaw gateway adapter"
```

## Task 7: Add Codex Runner Adapter on Top of Existing Shell Runtime

**Files:**
- Create: `internal/adapters/runner/codex/adapter.go`
- Create: `internal/adapters/runner/codex/session.go`
- Test: `internal/adapters/runner/codex/adapter_test.go`

- [ ] **Step 1: Write failing runner-adapter tests**

```go
func TestDispatchWritableTaskStartsCodexRun(t *testing.T) {
    runner := NewCodexAdapter(fakeScripts)
    run, err := runner.Dispatch(task)
    require.NoError(t, err)
    require.Equal(t, "codex", run.RunnerKind)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/adapters/runner/codex`
Expected: FAIL with missing adapter

- [ ] **Step 3: Implement shell-script-backed runner adapter**

```go
cmd := exec.Command("codex", "exec", "-C", workdir, prompt)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/adapters/runner/codex`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/adapters/runner/codex/*.go internal/adapters/runner/codex/adapter_test.go
git commit -m "feat: add codex runner adapter"
```

## Task 8: Implement Dispatch Orchestration and Strict Approval Flow

**Files:**
- Create: `internal/app/command/dispatch_task.go`
- Test: `internal/app/command/dispatch_task_test.go`

- [ ] **Step 1: Write failing dispatch tests**

```go
func TestDispatchAcquiresRepoLeaseAndStartsRun(t *testing.T) {
    handler := NewDispatchTaskHandler(fakeTaskRepo{}, fakeLeaseRepo{}, fakePolicy{}, fakeRunner{}, fakeApprovalRepo{}, fakeRunRepo{}, fakeArtifactRepo{})
    out, err := handler.Handle(DispatchTaskCommand{TaskID: taskID})
    require.NoError(t, err)
    require.Equal(t, "running", out.RunState)
}

func TestDispatchCreatesApprovalWhenRiskyActionDetected(t *testing.T) {
    handler := NewDispatchTaskHandler(fakeTaskRepo{}, fakeLeaseRepo{}, fakeStrictPolicy{}, fakeRiskyRunner{}, fakeApprovalRepo{}, fakeRunRepo{}, fakeArtifactRepo{})
    out, err := handler.Handle(DispatchTaskCommand{TaskID: riskyTaskID})
    require.NoError(t, err)
    require.Equal(t, "waiting_approval", out.TaskState)
}

func TestDispatchIndexesAssistantSummaryArtifact(t *testing.T) {
    handler := NewDispatchTaskHandler(fakeTaskRepo{}, fakeLeaseRepo{}, fakePolicy{}, fakeRunner{}, fakeApprovalRepo{}, fakeRunRepo{}, fakeArtifactRepo{})
    out, err := handler.Handle(DispatchTaskCommand{TaskID: taskID})
    require.NoError(t, err)
    require.NotEmpty(t, out.ArtifactIDs)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/app/command -run Dispatch`
Expected: FAIL with missing dispatch orchestration

- [ ] **Step 3: Implement dispatch orchestration**

```go
type ExecutionIntent struct {
    RequestedAction string
    IsWritable      bool
    RunnerKind      string
}

func (h *DispatchTaskHandler) Handle(cmd DispatchTaskCommand) (DispatchTaskResult, error) {
    task, err := h.Tasks.Get(cmd.TaskID)
    if err != nil { return DispatchTaskResult{}, err }
    intent := ExecutionIntent{
        RequestedAction: cmd.RequestedAction,
        IsWritable:      true,
        RunnerKind:      "codex",
    }
    decision := h.Policy.Evaluate(intent)
    if decision.RequiresApproval {
        approvalID, err := h.Approvals.Create(task.ID, decision.Reason)
        if err != nil { return DispatchTaskResult{}, err }
        if err := h.Tasks.SetState(task.ID, "waiting_approval"); err != nil { return DispatchTaskResult{}, err }
        return DispatchTaskResult{TaskState: "waiting_approval", ApprovalID: approvalID}, nil
    }
    if err := h.Leases.Acquire(task.WriteScope); err != nil { return DispatchTaskResult{}, err }
    run, err := h.Runner.Dispatch(task)
    if err != nil { return DispatchTaskResult{}, err }
    if err := h.Runs.Create(task.ID, run.ID, run.State); err != nil { return DispatchTaskResult{}, err }
    if err := h.Tasks.SetState(task.ID, "running"); err != nil { return DispatchTaskResult{}, err }
    artifactID, err := h.Artifacts.Index(task.ID, "assistant_summary", run.AssistantSummaryPath)
    if err != nil { return DispatchTaskResult{}, err }
    return DispatchTaskResult{TaskState: "running", RunState: run.State, ArtifactIDs: []string{artifactID}}, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/app/command -run Dispatch`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/command/dispatch_task.go internal/app/command/dispatch_task_test.go
git commit -m "feat: add dispatch orchestration and approval flow"
```

## Task 9: Add HTTP Board, CLI Commands, and End-to-End Foreman Flow

**Files:**
- Create: `internal/adapters/http/router.go`
- Create: `internal/adapters/http/board_handlers.go`
- Create: `internal/adapters/http/dto.go`
- Create: `internal/adapters/cli/project.go`
- Create: `internal/adapters/cli/action.go`
- Create: `web/board/index.html`
- Create: `web/board/app.js`
- Create: `web/board/styles.css`
- Test: `internal/adapters/http/board_handlers_test.go`
- Test: `test/e2e_phase1_test.go`
- Modify: `README.md`
- Modify: `INSTALL.md`
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Write failing board and end-to-end tests**

```go
func TestBoardReturnsModuleAndTaskColumns(t *testing.T) {
    router := NewRouter(app)
    req := httptest.NewRequest(http.MethodGet, "/board/tasks?project_id=demo", nil)
    rec := httptest.NewRecorder()
    router.ServeHTTP(rec, req)
    require.Equal(t, http.StatusOK, rec.Code)
}

func TestBoardActionEndpointsWireToCommands(t *testing.T) {
    router := NewRouter(app)
    cases := []struct{
        path      string
        body      string
        wantState string
    }{
        {path: "/board/tasks/task-1/approve", body: "", wantState: "review"},
        {path: "/board/tasks/task-1/retry", body: "", wantState: "ready"},
        {path: "/board/tasks/task-1/cancel", body: "", wantState: "canceled"},
        {path: "/board/tasks/task-1/reprioritize", body: `{"priority":5}`, wantState: "ready"},
    }
    for _, tc := range cases {
        req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
        rec := httptest.NewRecorder()
        router.ServeHTTP(rec, req)
        require.Equal(t, http.StatusOK, rec.Code)
        require.Equal(t, tc.wantState, fakeTaskStore("task-1").State)
    }
}

func TestRunDetailEndpointReturnsArtifactSummaries(t *testing.T) {
    router := NewRouter(app)
    req := httptest.NewRequest(http.MethodGet, "/board/runs/run-1", nil)
    rec := httptest.NewRecorder()
    router.ServeHTTP(rec, req)
    require.Equal(t, http.StatusOK, rec.Code)
    require.Contains(t, rec.Body.String(), "assistant_summary")
}

func TestOpenClawGatewayEndpointReturnsResponseEnvelope(t *testing.T) {
    router := NewRouter(app)
    req := httptest.NewRequest(http.MethodPost, "/gateways/openclaw/command", strings.NewReader(`{"session_id":"oc-1","action":"create_task"}`))
    rec := httptest.NewRecorder()
    router.ServeHTTP(rec, req)
    require.Equal(t, http.StatusOK, rec.Code)
    require.Contains(t, rec.Body.String(), "completion")
}

func TestPhase1FlowFromOpenClawCommandToBoardState(t *testing.T) {
    // OpenClaw create_task command -> Foreman persists task -> dispatch -> board query -> approval/completion response emitted
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/adapters/http ./test -run Phase1`
Expected: FAIL with missing router/handlers/board assets or missing end-to-end wiring

- [ ] **Step 3: Implement the final vertical slice**

```go
router.GET("/board/modules", handler.ModuleBoard)
router.GET("/board/tasks", handler.TaskBoard)
router.GET("/board/runs/:id", handler.RunDetail)
router.POST("/board/tasks/:id/approve", handler.ApproveTask)
router.POST("/board/tasks/:id/retry", handler.RetryTask)
router.POST("/board/tasks/:id/cancel", handler.CancelTask)
router.POST("/board/tasks/:id/reprioritize", handler.ReprioritizeTask)
router.POST("/gateways/openclaw/command", handler.OpenClawCommand)
```

- [ ] **Step 4: Run full verification**

Run: `go test ./...`
Expected: PASS across domain, app, adapters, infrastructure, and e2e packages

- [ ] **Step 5: Commit**

```bash
git add internal/adapters/http/*.go internal/adapters/cli/*.go web/board/* test/e2e_phase1_test.go README.md INSTALL.md CHANGELOG.md
git commit -m "feat: ship foreman go phase 1 vertical slice"
```

## Manual Smoke Checks After Task 9

- [ ] Run: `go test ./...`
  Expected: all Go tests pass

- [ ] Run: `go run ./cmd/foreman --help`
  Expected: root CLI help prints successfully

- [ ] Run: `go run ./cmd/foreman serve`
  Expected: Foreman starts with SQLite/artifact/runtime locations initialized

- [ ] Run: `curl http://localhost:<port>/board/tasks?project_id=<id>`
  Expected: JSON task-board payload returns

- [ ] Run an OpenClaw-style command through the chosen gateway adapter
  Expected: command is normalized, task state persists, board view updates

## Deferred Work After Phase 1

- Claude runner adapter
- Nanobot gateway adapter
- ZeroClaw gateway adapter
- daemon split or remote runners
- richer board controls
- separate query database
- event sourcing or domain events beyond simple internal notifications
