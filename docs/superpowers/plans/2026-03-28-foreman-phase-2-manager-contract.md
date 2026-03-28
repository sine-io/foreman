# Foreman Phase 2 Manager-Agent Contract Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first Phase 2 slice: a normalized manager-agent contract that lets OpenClaw, Nanobot, ZeroClaw, or direct Foreman clients call the same control-plane API without pulling ACP or gateway responsibilities into Foreman core.

**Architecture:** Introduce an application-layer manager-agent service that sits between transport adapters and existing command/query handlers. OpenClaw becomes a thin transport adapter over that service, and Foreman exposes the same contract through a native HTTP manager API so future upstream managers can integrate without custom per-adapter control-plane logic.

**Tech Stack:** Go, `gin`, SQLite, existing command/query handlers, existing OpenClaw adapter

---

## Scope Check

Phase 2 is larger than one implementation plan. This plan intentionally covers only the first Phase 2 sub-project:

- normalized manager-agent contract
- OpenClaw refactor onto that contract
- Foreman-native manager HTTP API

Explicitly out of scope for this plan:

- ACP adapter implementation
- Nanobot adapter implementation
- ZeroClaw adapter implementation
- board polish work
- richer approval policy expansion beyond what is needed for contract integration
- new runner types

Follow-on Phase 2 plans should cover:

- control-plane governance hardening
- board/operator UX improvements
- additional upstream adapter packages

## File Structure

### Application integration service

- Create: `internal/app/manageragent/types.go`
  Responsibility: application-level request/response contract independent of transport or ACP.
- Create: `internal/app/manageragent/service.go`
  Responsibility: normalize manager-agent intents into project/module/task creation, dispatch, status lookup, and board snapshot access.
- Create: `internal/app/manageragent/service_test.go`
  Responsibility: service-level tests for completion flow, approval-needed flow, and state lookup.

### Transport adapters

- Modify: `internal/adapters/gateway/manageragent/types.go`
  Responsibility change: keep transport DTOs thin and aligned to the new application contract.
- Modify: `internal/adapters/gateway/openclaw/handler.go`
  Responsibility change: OpenClaw becomes a transport mapper over the manager-agent service.
- Modify: `internal/adapters/gateway/openclaw/handler_test.go`
  Responsibility change: verify OpenClaw still maps envelopes correctly after the refactor.

### Native HTTP manager API

- Create: `internal/adapters/http/manager_handlers.go`
  Responsibility: HTTP endpoints for upstream managers that want Foreman-native integration instead of ACP.
- Create: `internal/adapters/http/manager_handlers_test.go`
  Responsibility: verify POST/GET manager endpoints over the normalized service contract.
- Modify: `internal/adapters/http/router.go`
  Responsibility change: register manager API routes alongside board routes.
- Modify: `internal/adapters/http/dto.go`
  Responsibility change: add request/response DTOs for manager API only.

### Bootstrap wiring

- Modify: `internal/bootstrap/app.go`
  Responsibility change: construct and expose the manager-agent application service; use it from OpenClaw and HTTP.
- Modify: `internal/bootstrap/app_test.go`
  Responsibility change: verify the new manager API is reachable in the live bootstrap path.

### Documentation

- Modify: `README.md`
  Responsibility change: explain the new Foreman-native manager API and the Phase 2 contract boundary.
- Modify: `INSTALL.md`
  Responsibility change: add manager API smoke steps.
- Modify: `CHANGELOG.md`
  Responsibility change: add the Phase 2 manager-agent contract slice.

## Task 1: Define the Application-Level Manager-Agent Contract

**Files:**
- Create: `internal/app/manageragent/types.go`
- Create: `internal/app/manageragent/service.go`
- Create: `internal/app/manageragent/service_test.go`

- [ ] **Step 1: Write the failing service tests**

```go
func TestHandleCreateTaskReturnsCompletionWhenDispatchFinishes(t *testing.T) {
    svc := NewService(fakeDeps...)
    out, err := svc.Handle(context.Background(), Request{
        Kind:      "create_task",
        SessionID: "mgr-1",
        Summary:   "Summarize the module status",
    })
    require.NoError(t, err)
    require.Equal(t, "completion", out.Kind)
    require.NotEmpty(t, out.TaskID)
}

func TestHandleCreateTaskReturnsApprovalNeededWhenPolicyRequiresApproval(t *testing.T) {
    svc := NewService(fakeDepsRequiringApproval...)
    out, err := svc.Handle(context.Background(), Request{
        Kind:      "create_task",
        SessionID: "mgr-2",
        Summary:   "git push origin main",
    })
    require.NoError(t, err)
    require.Equal(t, "approval_needed", out.Kind)
    require.NotEmpty(t, out.Summary)
}

func TestTaskStatusReturnsPersistedRunAndApprovalState(t *testing.T) {
    svc := NewService(fakeDeps...)
    view, err := svc.TaskStatus(context.Background(), "task-1")
    require.NoError(t, err)
    require.Equal(t, "task-1", view.TaskID)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/app/manageragent`
Expected: FAIL with missing package/types/service

- [ ] **Step 3: Implement the minimal contract types and service**

```go
type Request struct {
    Kind      string
    SessionID string
    TaskID    string
    Summary   string
}

type Response struct {
    Kind    string
    TaskID  string
    Summary string
}

type Service struct {
    CreateTask   *command.CreateTaskHandler
    DispatchTask *command.DispatchTaskHandler
    QueryBoard   *query.TaskBoardQuery
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/app/manageragent`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/manageragent/types.go internal/app/manageragent/service.go internal/app/manageragent/service_test.go
git commit -m "feat: add manager-agent application service"
```

## Task 2: Refactor OpenClaw to Use the Manager-Agent Service

**Files:**
- Modify: `internal/adapters/gateway/manageragent/types.go`
- Modify: `internal/adapters/gateway/openclaw/handler.go`
- Modify: `internal/adapters/gateway/openclaw/handler_test.go`

- [ ] **Step 1: Write the failing transport-adapter tests**

```go
func TestOpenClawHandlerDelegatesToManagerService(t *testing.T) {
    svc := fakeManagerService{
        response: manageragent.Response{
            Kind:   "completion",
            TaskID: "task-1",
        },
    }
    handler := NewHandler(svc)
    resp, err := handler.Handle(context.Background(), Envelope{
        SessionID: "oc-1",
        Action:    "create_task",
        Summary:   "Bootstrap board",
    })
    require.NoError(t, err)
    require.Equal(t, "completion", resp.Kind)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/adapters/gateway/openclaw`
Expected: FAIL because the handler still depends on the older ad-hoc command bus shape

- [ ] **Step 3: Implement the minimal refactor**

```go
type Service interface {
    Handle(context.Context, manageragent.Request) (manageragent.Response, error)
}

func (h *Handler) Handle(ctx context.Context, env Envelope) (Response, error) {
    result, err := h.service.Handle(ctx, mapEnvelope(env))
    if err != nil {
        return Response{}, err
    }
    return encodeResponse(result), nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/adapters/gateway/openclaw`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/adapters/gateway/manageragent/types.go internal/adapters/gateway/openclaw/handler.go internal/adapters/gateway/openclaw/handler_test.go
git commit -m "refactor: move openclaw onto manager-agent service"
```

## Task 3: Add a Foreman-Native Manager HTTP API

**Files:**
- Create: `internal/adapters/http/manager_handlers.go`
- Create: `internal/adapters/http/manager_handlers_test.go`
- Modify: `internal/adapters/http/router.go`
- Modify: `internal/adapters/http/dto.go`

- [ ] **Step 1: Write the failing manager API tests**

```go
func TestManagerCommandEndpointCreatesTaskThroughNormalizedService(t *testing.T) {
    router := NewRouter(fakeApp)
    req := httptest.NewRequest(
        http.MethodPost,
        "/api/manager/commands",
        strings.NewReader(`{"kind":"create_task","summary":"Bootstrap board"}`),
    )
    rec := httptest.NewRecorder()
    router.ServeHTTP(rec, req)
    require.Equal(t, http.StatusOK, rec.Code)
    require.Contains(t, rec.Body.String(), "completion")
}

func TestManagerTaskStatusEndpointReturnsTaskSnapshot(t *testing.T) {
    router := NewRouter(fakeApp)
    req := httptest.NewRequest(http.MethodGet, "/api/manager/tasks/task-1", nil)
    rec := httptest.NewRecorder()
    router.ServeHTTP(rec, req)
    require.Equal(t, http.StatusOK, rec.Code)
    require.Contains(t, rec.Body.String(), "task-1")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/adapters/http -run Manager`
Expected: FAIL with missing routes/handlers/DTOs

- [ ] **Step 3: Implement the minimal HTTP manager API**

```go
router.POST("/api/manager/commands", handler.ManagerCommand)
router.GET("/api/manager/tasks/:id", handler.ManagerTaskStatus)
router.GET("/api/manager/projects/:id/board", handler.ManagerBoardSnapshot)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/adapters/http -run Manager`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/adapters/http/manager_handlers.go internal/adapters/http/manager_handlers_test.go internal/adapters/http/router.go internal/adapters/http/dto.go
git commit -m "feat: add manager http api"
```

## Task 4: Wire Bootstrap and Live Integration Path

**Files:**
- Modify: `internal/bootstrap/app.go`
- Modify: `internal/bootstrap/app_test.go`

- [ ] **Step 1: Write the failing bootstrap integration test**

```go
func TestServeExposesManagerCommandAPI(t *testing.T) {
    cfg := testConfig(t)
    app, err := BuildApp(cfg)
    require.NoError(t, err)

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    go app.Serve(ctx)
    waitForHTTP(t, cfg.HTTPAddr)

    resp, err := http.Post(
        "http://"+cfg.HTTPAddr+"/api/manager/commands",
        "application/json",
        strings.NewReader(`{"kind":"create_task","summary":"Bootstrap board"}`),
    )
    require.NoError(t, err)
    require.Equal(t, http.StatusOK, resp.StatusCode)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/bootstrap -run Manager`
Expected: FAIL because bootstrap does not expose the new manager API yet

- [ ] **Step 3: Implement the minimal bootstrap wiring**

```go
type app struct {
    managerService *manageragent.Service
}

instance.managerService = manageragent.NewService(...)
instance.router = httpadapter.NewRouter(instance)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/bootstrap -run Manager`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/bootstrap/app.go internal/bootstrap/app_test.go
git commit -m "feat: wire manager-agent service into bootstrap"
```

## Task 5: Update Docs and Verification Guidance

**Files:**
- Modify: `README.md`
- Modify: `INSTALL.md`
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Write the failing documentation expectation**

Create a short checklist in your working notes:

```text
- README references the Foreman-native manager API
- INSTALL includes manager API smoke commands
- CHANGELOG records the Phase 2 manager-agent contract slice
```

- [ ] **Step 2: Check docs to verify the new contract is not documented yet**

Run: `rg -n "/api/manager|manager-agent service|manager http api" README.md INSTALL.md CHANGELOG.md`
Expected: no or incomplete matches

- [ ] **Step 3: Update the docs**

Add exact examples such as:

```bash
curl -X POST http://localhost:<port>/api/manager/commands \
  -H 'Content-Type: application/json' \
  -d '{"kind":"create_task","summary":"Bootstrap board"}'
```

- [ ] **Step 4: Run verification**

Run: `rg -n "/api/manager|manager-agent service|manager http api" README.md INSTALL.md CHANGELOG.md`
Expected: matches present in all intended files

- [ ] **Step 5: Commit**

```bash
git add README.md INSTALL.md CHANGELOG.md
git commit -m "docs: add manager-agent api guidance"
```

## Milestone Verification

Run these after all tasks complete:

```bash
go test ./internal/app/manageragent
go test ./internal/adapters/gateway/openclaw
go test ./internal/adapters/http -run Manager
go test ./internal/bootstrap -run Manager
go test ./...
```

Manual smoke:

```bash
go run ./cmd/foreman serve
curl -X POST http://localhost:<port>/api/manager/commands \
  -H 'Content-Type: application/json' \
  -d '{"kind":"create_task","summary":"Summarize current project status"}'
curl http://localhost:<port>/api/manager/tasks/<task-id>
curl http://localhost:<port>/api/manager/projects/demo/board
```
