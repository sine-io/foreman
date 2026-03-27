package http

import (
	"context"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sine-io/foreman/internal/adapters/gateway/openclaw"
	"github.com/sine-io/foreman/internal/app/command"
	"github.com/sine-io/foreman/internal/app/query"
	"github.com/stretchr/testify/require"
)

func TestBoardReturnsModuleAndTaskColumns(t *testing.T) {
	router := NewRouter(newFakeHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/board/tasks?project_id=demo", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "Waiting Approval")
}

func TestBoardActionEndpointsWireToCommands(t *testing.T) {
	cases := []struct {
		path      string
		body      string
		wantState string
	}{
		{path: "/board/tasks/task-1/approve", body: "", wantState: "leased"},
		{path: "/board/tasks/task-1/retry", body: "", wantState: "ready"},
		{path: "/board/tasks/task-1/cancel", body: "", wantState: "canceled"},
		{path: "/board/tasks/task-1/reprioritize", body: `{"priority":5}`, wantState: "ready"},
	}

	for _, tc := range cases {
		app := newFakeHTTPApp()
		router := NewRouter(app)

		req := httptest.NewRequest(stdhttp.MethodPost, tc.path, strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		require.Equal(t, stdhttp.StatusOK, rec.Code)
		require.Equal(t, tc.wantState, app.tasks["task-1"].State)
	}
}

func TestRunDetailEndpointReturnsArtifactSummaries(t *testing.T) {
	router := NewRouter(newFakeHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/board/runs/run-1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "assistant_summary")
}

func TestApprovalQueueEndpointReturnsPendingItems(t *testing.T) {
	router := NewRouter(newFakeHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/board/approvals?project_id=demo", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "approval-1")
	require.Contains(t, rec.Body.String(), "git push origin main")
}

func TestBoardIndexServesHTML(t *testing.T) {
	router := NewRouter(newFakeHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/board", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "Foreman Board")
}

func TestBoardJavaScriptAssetServesInteractiveApp(t *testing.T) {
	router := NewRouter(newFakeHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/board/assets/app.js", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "/board/tasks")
	require.Contains(t, rec.Body.String(), "/board/approvals")
}

func TestOpenClawGatewayEndpointReturnsResponseEnvelope(t *testing.T) {
	router := NewRouter(newFakeHTTPApp())

	req := httptest.NewRequest(
		stdhttp.MethodPost,
		"/gateways/openclaw/command",
		strings.NewReader(`{"session_id":"oc-1","action":"create_task","summary":"Bootstrap board"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "completion")
}

type fakeHTTPTask struct {
	ID       string
	Summary  string
	State    string
	Priority int
}

type fakeHTTPApp struct {
	tasks map[string]fakeHTTPTask
}

func newFakeHTTPApp() *fakeHTTPApp {
	return &fakeHTTPApp{
		tasks: map[string]fakeHTTPTask{
			"task-1": {
				ID:       "task-1",
				Summary:  "Review push",
				State:    "ready",
				Priority: 1,
			},
		},
	}
}

func (a *fakeHTTPApp) ModuleBoard(projectID string) (query.ModuleBoardView, error) {
	return query.ModuleBoardView{
		Columns: map[string][]query.ModuleCard{
			"Implementing": {
				{ID: "module-1", Name: "Bootstrap", State: "active"},
			},
		},
	}, nil
}

func (a *fakeHTTPApp) TaskBoard(projectID string) (query.TaskBoardView, error) {
	view := query.TaskBoardView{
		Columns: map[string][]query.TaskCard{
			"Waiting Approval": {},
			"Ready":            {},
			"Done":             {},
		},
	}

	for _, item := range a.tasks {
		card := query.TaskCard{
			ID:       item.ID,
			Summary:  item.Summary,
			State:    item.State,
			Priority: item.Priority,
		}

		switch item.State {
		case "ready":
			view.Columns["Ready"] = append(view.Columns["Ready"], card)
		case "completed":
			view.Columns["Done"] = append(view.Columns["Done"], card)
		default:
			view.Columns["Waiting Approval"] = append(view.Columns["Waiting Approval"], card)
		}
	}

	return view, nil
}

func (a *fakeHTTPApp) RunDetail(runID string) (query.RunDetailView, error) {
	return query.RunDetailView{
		ID:          runID,
		TaskID:      "task-1",
		RunnerKind:  "codex",
		State:       "completed",
		TaskSummary: "Bootstrap board",
		Artifacts: []query.ArtifactSummary{
			{ID: "artifact-1", Kind: "assistant_summary", Summary: "Created board view"},
		},
	}, nil
}

func (a *fakeHTTPApp) ApprovalQueue(projectID string) (query.ApprovalQueueView, error) {
	return query.ApprovalQueueView{
		Items: []query.ApprovalCard{
			{
				ApprovalID: "approval-1",
				TaskID:     "task-1",
				ModuleID:   "module-1",
				Summary:    "Review push",
				Reason:     "git push origin main",
				State:      "pending",
			},
		},
	}, nil
}

func (a *fakeHTTPApp) ApproveTask(cmd command.ApproveTaskCommand) (string, error) {
	task := a.tasks[cmd.TaskID]
	task.State = "leased"
	a.tasks[cmd.TaskID] = task
	return task.State, nil
}

func (a *fakeHTTPApp) RetryTask(cmd command.RetryTaskCommand) (string, error) {
	task := a.tasks[cmd.TaskID]
	task.State = "ready"
	a.tasks[cmd.TaskID] = task
	return task.State, nil
}

func (a *fakeHTTPApp) CancelTask(cmd command.CancelTaskCommand) (string, error) {
	task := a.tasks[cmd.TaskID]
	task.State = "canceled"
	a.tasks[cmd.TaskID] = task
	return task.State, nil
}

func (a *fakeHTTPApp) ReprioritizeTask(cmd command.ReprioritizeTaskCommand) (string, error) {
	task := a.tasks[cmd.TaskID]
	task.Priority = cmd.Priority
	a.tasks[cmd.TaskID] = task
	return task.State, nil
}

func (a *fakeHTTPApp) OpenClawCommand(ctx context.Context, env openclaw.Envelope) (openclaw.Response, error) {
	if env.Action == "create_task" {
		a.tasks["task-openclaw"] = fakeHTTPTask{
			ID:      "task-openclaw",
			Summary: env.Summary,
			State:   "ready",
		}
	}

	return openclaw.Response{
		Kind:    "completion",
		TaskID:  "task-openclaw",
		Summary: env.Summary,
	}, nil
}
