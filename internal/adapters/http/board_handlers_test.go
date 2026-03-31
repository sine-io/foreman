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

func TestRunDetailEndpointRedirectsToRunWorkbench(t *testing.T) {
	router := NewRouter(newFakeHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/board/runs/run-1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusSeeOther, rec.Code)
	require.Equal(t, "/board/runs/workbench?run_id=run-1", rec.Header().Get("Location"))
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

func TestApprovalWorkbenchPageServesHTML(t *testing.T) {
	router := NewRouter(newFakeManagerHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/board/approvals/workbench?project_id=demo", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "Approval Workbench")
	require.Contains(t, rec.Body.String(), "/board/assets/approval-workbench.js")
}

func TestTaskWorkbenchPageServesHTML(t *testing.T) {
	router := NewRouter(newFakeManagerHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/board/tasks/workbench?project_id=project-2&task_id=task-workbench", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "Task Workbench")
	require.Contains(t, rec.Body.String(), "/board/assets/task-workbench.js")
}

func TestArtifactWorkbenchPlaceholderPageServes(t *testing.T) {
	router := NewRouter(newFakeManagerHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/board/artifacts/workbench?artifact_id=artifact-1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "Artifact workbench placeholder")
	require.Contains(t, rec.Body.String(), "artifact-1")
}

func TestRunWorkbenchPageServes(t *testing.T) {
	router := NewRouter(newFakeManagerHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/board/runs/workbench?run_id=run-1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "Run Workbench")
	require.Contains(t, rec.Body.String(), "/board/assets/run-workbench.js")
}

func TestTaskWorkbenchJavaScriptUsesProjectAndTaskURLState(t *testing.T) {
	router := NewRouter(newFakeManagerHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/board/assets/task-workbench.js", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "new URLSearchParams(window.location.search)")
	require.Contains(t, rec.Body.String(), "searchParams.get(\"project_id\")")
	require.Contains(t, rec.Body.String(), "searchParams.get(\"task_id\")")
	require.Contains(t, rec.Body.String(), "searchParams.set(\"project_id\", projectId || \"demo\")")
	require.Contains(t, rec.Body.String(), "searchParams.set(\"task_id\", taskId)")
}

func TestRunWorkbenchJavaScriptUsesRunIDURLState(t *testing.T) {
	router := NewRouter(newFakeManagerHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/board/assets/run-workbench.js", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "new URLSearchParams(window.location.search)")
	require.Contains(t, rec.Body.String(), "searchParams.get(\"run_id\")")
	require.Contains(t, rec.Body.String(), "searchParams.set(\"run_id\", runId)")
}

func TestTaskWorkbenchJavaScriptIncludesDisabledActionReasons(t *testing.T) {
	router := NewRouter(newFakeManagerHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/board/assets/task-workbench.js", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "detail.disabled_reasons")
	require.Contains(t, rec.Body.String(), "action.disabled_reason")
	require.Contains(t, rec.Body.String(), "No approval history")
}

func TestTaskWorkbenchJavaScriptLinksLatestRunsToRunWorkbenchRoute(t *testing.T) {
	router := NewRouter(newFakeManagerHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/board/assets/task-workbench.js", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "/board/runs/workbench?run_id=")
	require.Contains(t, rec.Body.String(), "detail.latest_run_id")
	require.NotContains(t, rec.Body.String(), "detail.run_detail_url")
}

func TestRunWorkbenchJavaScriptUsesServerProvidedTaskWorkbenchURL(t *testing.T) {
	router := NewRouter(newFakeManagerHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/board/assets/run-workbench.js", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "detail.task_workbench_url")
}

func TestRunWorkbenchJavaScriptUsesArtifactTargetURLs(t *testing.T) {
	router := NewRouter(newFakeManagerHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/board/assets/run-workbench.js", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "detail.artifact_target_urls")
	require.Contains(t, rec.Body.String(), "detail.artifact_target_urls[artifact.id]")
}

func TestRunWorkbenchJavaScriptIncludesRefreshControl(t *testing.T) {
	router := NewRouter(newFakeManagerHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/board/assets/run-workbench.js", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "document.getElementById(\"run-workbench-refresh\")")
	require.Contains(t, rec.Body.String(), "refreshButton.addEventListener(\"click\", refreshWorkbench)")
}

func TestRunWorkbenchJavaScriptGuardsAgainstStaleResponses(t *testing.T) {
	router := NewRouter(newFakeManagerHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/board/assets/run-workbench.js", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "const requestedRunId = state.runId;")
	require.Contains(t, rec.Body.String(), "const requestToken = ++state.requestToken;")
	require.Contains(t, rec.Body.String(), "if (requestToken !== state.requestToken) {")
}

func TestRunWorkbenchJavaScriptRendersSupplementalMetadata(t *testing.T) {
	router := NewRouter(newFakeManagerHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/board/assets/run-workbench.js", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "detail.run_created_at")
	require.Contains(t, rec.Body.String(), "detail.runner_kind")
	require.Contains(t, rec.Body.String(), "detail.project_id")
	require.Contains(t, rec.Body.String(), "detail.module_id")
}

func TestApprovalWorkbenchJavaScriptAssetServesManagerApprovalClient(t *testing.T) {
	router := NewRouter(newFakeManagerHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/board/assets/approval-workbench.js", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "/api/manager/projects/")
	require.Contains(t, rec.Body.String(), "/api/manager/approvals/")
	require.Contains(t, rec.Body.String(), "/retry-dispatch")
}

func TestApprovalWorkbenchJavaScriptUsesApprovalIDURLState(t *testing.T) {
	router := NewRouter(newFakeManagerHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/board/assets/approval-workbench.js", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "new URLSearchParams(window.location.search)")
	require.Contains(t, rec.Body.String(), "searchParams.set(\"approval_id\", approvalId)")
	require.Contains(t, rec.Body.String(), "searchParams.get(\"approval_id\")")
}

func TestApprovalWorkbenchJavaScriptAdvancesToNextItemAfterAction(t *testing.T) {
	router := NewRouter(newFakeManagerHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/board/assets/approval-workbench.js", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "const currentIndex = state.queue.findIndex((item) => item.approval_id === approvalId);")
	require.Contains(t, rec.Body.String(), "const nextItem = state.queue[currentIndex + 1] || state.queue[currentIndex - 1] || null;")
	require.Contains(t, rec.Body.String(), "await selectApproval(nextItem.approval_id);")
}

func TestApprovalWorkbenchJavaScriptClearsStaleApprovalIDOnProjectSwitch(t *testing.T) {
	router := NewRouter(newFakeManagerHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/board/assets/approval-workbench.js", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "const previousProjectId = state.projectId;")
	require.Contains(t, rec.Body.String(), "const projectChanged = nextProjectId !== previousProjectId;")
	require.Contains(t, rec.Body.String(), "const requestedApprovalID = projectChanged ? \"\" : readApprovalID();")
}

func TestApprovalWorkbenchJavaScriptClearsStateBeforeRefreshFailureCanRender(t *testing.T) {
	router := NewRouter(newFakeManagerHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/board/assets/approval-workbench.js", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "state.queue = [];")
	require.Contains(t, rec.Body.String(), "state.detail = null;")
	require.Contains(t, rec.Body.String(), "state.selectedApprovalID = \"\";")
	require.Contains(t, rec.Body.String(), "state.queueState = \"loading\";")
}

func TestApprovalWorkbenchJavaScriptLinksDetailToTaskWorkbench(t *testing.T) {
	router := NewRouter(newFakeManagerHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/board/assets/approval-workbench.js", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "/board/tasks/workbench?project_id=")
	require.Contains(t, rec.Body.String(), "detail.task_id")
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
