package http

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sine-io/foreman/internal/app/command"
	"github.com/sine-io/foreman/internal/app/manageragent"
	"github.com/stretchr/testify/require"
)

func TestManagerCommandEndpointCreatesTaskThroughNormalizedService(t *testing.T) {
	router := NewRouter(newFakeManagerHTTPApp())

	req := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/manager/commands",
		strings.NewReader(`{"kind":"create_task","summary":"Bootstrap board"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	var resp managerCommandResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "completion", resp.Kind)
	require.Equal(t, "task-created", resp.TaskID)
	require.Equal(t, "Bootstrap board", resp.Summary)
}

func TestManagerTaskStatusEndpointReturnsTaskSnapshot(t *testing.T) {
	router := NewRouter(newFakeManagerHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/api/manager/tasks/task-1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	var resp managerTaskStatusResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "task-1", resp.TaskID)
	require.Equal(t, "project-1", resp.ProjectID)
	require.Equal(t, 1, resp.Priority)
	require.Equal(t, "run-1", resp.RunID)
	require.Equal(t, "completed", resp.RunState)
	require.False(t, resp.PendingApproval)
}

func TestManagerTaskWorkbenchEndpointReturnsProjectScopedView(t *testing.T) {
	router := NewRouter(newFakeManagerHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/api/manager/tasks/task-workbench/workbench?project_id=project-2", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	var resp manageragent.TaskWorkbenchView
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "task-workbench", resp.TaskID)
	require.Equal(t, "project-2", resp.ProjectID)
	require.Equal(t, "module-2", resp.ModuleID)
	require.Equal(t, "waiting_approval", resp.TaskState)
	require.Equal(t, "approval-9", resp.LatestApprovalID)
	require.Equal(t, "pending", resp.LatestApprovalState)
	require.Equal(t, "Waiting approval", managerTaskWorkbenchAction(resp.AvailableActions, "dispatch").DisabledReason)
}

func TestManagerBoardSnapshotEndpointReturnsBoardShape(t *testing.T) {
	router := NewRouter(newFakeManagerHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/api/manager/projects/project-1/board", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	var resp managerBoardSnapshotResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "project-1", resp.ProjectID)
	require.Len(t, resp.Modules["Implementing"], 1)
	require.Len(t, resp.Tasks["Done"], 1)
	require.Equal(t, "task-1", resp.Tasks["Done"][0].TaskID)
}

func TestManagerTaskStatusEndpointMapsMissingTaskTo404(t *testing.T) {
	router := NewRouter(newFakeManagerHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/api/manager/tasks/missing", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusNotFound, rec.Code)
}

func TestManagerCommandEndpointMapsClientErrorsTo400(t *testing.T) {
	router := NewRouter(newFakeManagerHTTPApp())

	req := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/manager/commands",
		strings.NewReader(`{"kind":"unsupported_action","summary":"Bootstrap board"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusBadRequest, rec.Code)
}

func TestManagerTaskWorkbenchActionEndpointsRespectProjectScope(t *testing.T) {
	paths := []string{
		"/api/manager/tasks/task-run/dispatch?project_id=project-2",
		"/api/manager/tasks/task-failed/retry?project_id=project-2",
		"/api/manager/tasks/task-ready/cancel?project_id=project-2",
		"/api/manager/tasks/task-ready/reprioritize?project_id=project-2",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			body := ""
			if strings.Contains(path, "reprioritize") {
				body = `{"priority":7}`
			}

			router := NewRouter(newFakeManagerHTTPApp())
			req := httptest.NewRequest(stdhttp.MethodPost, path, strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			require.Equal(t, stdhttp.StatusNotFound, rec.Code)
			require.Contains(t, rec.Body.String(), "not found in project project-2")
		})
	}
}

func TestManagerTaskWorkbenchActionEndpointsReturnConflictForIneligibleActions(t *testing.T) {
	cases := []struct {
		name        string
		path        string
		body        string
		wantMessage string
	}{
		{
			name:        "dispatch waiting approval",
			path:        "/api/manager/tasks/task-workbench/dispatch?project_id=project-2",
			wantMessage: "dispatch unavailable: Waiting approval",
		},
		{
			name:        "retry ready task",
			path:        "/api/manager/tasks/task-ready/retry?project_id=project-1",
			wantMessage: "retry unavailable: Task not failed",
		},
		{
			name:        "cancel completed task",
			path:        "/api/manager/tasks/task-completed/cancel?project_id=project-1",
			wantMessage: "cancel unavailable: Already completed",
		},
		{
			name:        "reprioritize canceled task",
			path:        "/api/manager/tasks/task-canceled/reprioritize?project_id=project-1",
			body:        `{"priority":7}`,
			wantMessage: "reprioritize unavailable: Task canceled",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := NewRouter(newFakeManagerHTTPApp())
			req := httptest.NewRequest(stdhttp.MethodPost, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			require.Equal(t, stdhttp.StatusConflict, rec.Code)
			require.Contains(t, rec.Body.String(), tc.wantMessage)
		})
	}
}

func TestManagerTaskWorkbenchReprioritizeEndpointBindsPriorityBody(t *testing.T) {
	app := newFakeManagerHTTPApp()
	router := NewRouter(app)

	req := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/manager/tasks/task-ready/reprioritize?project_id=project-1",
		strings.NewReader(`{"priority":7}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Equal(t, 7, app.lastReprioritizePriority)
}

func TestManagerTaskWorkbenchReprioritizeEndpointRejectsPriorityBelowOne(t *testing.T) {
	router := NewRouter(newFakeManagerHTTPApp())

	req := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/manager/tasks/task-ready/reprioritize?project_id=project-1",
		strings.NewReader(`{"priority":0}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusBadRequest, rec.Code)
}

func TestManagerTaskWorkbenchActionEndpointsReturnCompactRefreshRequiredPayloads(t *testing.T) {
	cases := []struct {
		name               string
		path               string
		body               string
		wantTaskID         string
		wantTaskState      string
		wantLatestRunState string
	}{
		{
			name:               "dispatch",
			path:               "/api/manager/tasks/task-run/dispatch?project_id=project-1",
			wantTaskID:         "task-run",
			wantTaskState:      "completed",
			wantLatestRunState: "completed",
		},
		{
			name:          "retry",
			path:          "/api/manager/tasks/task-failed/retry?project_id=project-1",
			wantTaskID:    "task-failed",
			wantTaskState: "ready",
		},
		{
			name:          "cancel",
			path:          "/api/manager/tasks/task-ready/cancel?project_id=project-1",
			wantTaskID:    "task-ready",
			wantTaskState: "canceled",
		},
		{
			name:          "reprioritize",
			path:          "/api/manager/tasks/task-ready/reprioritize?project_id=project-1",
			body:          `{"priority":5}`,
			wantTaskID:    "task-ready",
			wantTaskState: "ready",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := NewRouter(newFakeManagerHTTPApp())
			req := httptest.NewRequest(stdhttp.MethodPost, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			require.Equal(t, stdhttp.StatusOK, rec.Code)
			var resp map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
			require.Equal(t, tc.wantTaskID, resp["task_id"])
			require.Equal(t, tc.wantTaskState, resp["task_state"])
			require.Equal(t, true, resp["refresh_required"])
			require.NotContains(t, resp, "available_actions")
			require.NotContains(t, resp, "write_scope")
			if tc.wantLatestRunState != "" {
				require.Equal(t, tc.wantLatestRunState, resp["latest_run_state"])
			}
		})
	}
}

func TestManagerApprovalQueueEndpointReturnsPendingApprovals(t *testing.T) {
	router := NewRouter(newFakeManagerHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/api/manager/projects/project-1/approvals", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	var resp managerApprovalQueueResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Items, 1)
	require.Equal(t, "approval-1", resp.Items[0].ApprovalID)
	require.Equal(t, "high", resp.Items[0].RiskLevel)
	require.Equal(t, 9, resp.Items[0].Priority)
}

func TestManagerApprovalDetailEndpointReturnsHistoricalApproval(t *testing.T) {
	router := NewRouter(newFakeManagerHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/api/manager/approvals/approval-approved", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	var resp managerApprovalDetailResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "approval-approved", resp.ApprovalID)
	require.Equal(t, "approved", resp.ApprovalState)
	require.Equal(t, "run-2", resp.RunID)
	require.Equal(t, "/board/runs/run-2", resp.RunDetailURL)
}

func TestManagerApprovalDetailEndpointReturnsHistoricalRejectedApproval(t *testing.T) {
	router := NewRouter(newFakeManagerHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/api/manager/approvals/approval-rejected", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	var resp managerApprovalDetailResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "approval-rejected", resp.ApprovalID)
	require.Equal(t, "rejected", resp.ApprovalState)
	require.Equal(t, "manager rejected the action", resp.RejectionReason)
	require.Equal(t, "ready", resp.TaskState)
}

func TestManagerApprovalActionEndpointsReturnApprovalActionResponse(t *testing.T) {
	cases := []struct {
		name                string
		path                string
		body                string
		wantApprovalState   string
		wantTaskState       string
		wantRejectionReason string
	}{
		{
			name:              "approve",
			path:              "/api/manager/approvals/approval-1/approve",
			wantApprovalState: "approved",
			wantTaskState:     "completed",
		},
		{
			name:                "reject",
			path:                "/api/manager/approvals/approval-1/reject",
			body:                `{"rejection_reason":"needs rollback plan"}`,
			wantApprovalState:   "rejected",
			wantTaskState:       "ready",
			wantRejectionReason: "needs rollback plan",
		},
		{
			name:              "retry dispatch",
			path:              "/api/manager/approvals/approval-approved/retry-dispatch",
			wantApprovalState: "approved",
			wantTaskState:     "running",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := NewRouter(newFakeManagerHTTPApp())

			req := httptest.NewRequest(stdhttp.MethodPost, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			require.Equal(t, stdhttp.StatusOK, rec.Code)
			var resp managerApprovalWorkbenchActionResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
			require.Equal(t, tc.wantApprovalState, resp.ApprovalState)
			require.Equal(t, tc.wantTaskState, resp.TaskState)
			require.Equal(t, tc.wantRejectionReason, resp.RejectionReason)
			require.Equal(t, "task-1", resp.TaskID)
		})
	}
}

func TestManagerRejectApprovalEndpointPreservesStoredRejectionReasonOnRepeatReject(t *testing.T) {
	router := NewRouter(newFakeManagerHTTPApp())

	req := httptest.NewRequest(
		stdhttp.MethodPost,
		"/api/manager/approvals/approval-rejected/reject",
		strings.NewReader(`{"rejection_reason":"new request body reason"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	var resp managerApprovalWorkbenchActionResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "rejected", resp.ApprovalState)
	require.Equal(t, "stored persisted rejection reason", resp.RejectionReason)
}

func TestManagerApprovalEndpointsMapMissingAndConflictErrors(t *testing.T) {
	router := NewRouter(newFakeManagerHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/api/manager/approvals/missing", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, stdhttp.StatusNotFound, rec.Code)

	req = httptest.NewRequest(stdhttp.MethodPost, "/api/manager/approvals/conflict/approve", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, stdhttp.StatusConflict, rec.Code)
}

type fakeManagerHTTPApp struct {
	*fakeHTTPApp
	lastReprioritizePriority int
}

func newFakeManagerHTTPApp() *fakeManagerHTTPApp {
	return &fakeManagerHTTPApp{fakeHTTPApp: newFakeHTTPApp()}
}

func (a *fakeManagerHTTPApp) Handle(ctx context.Context, req manageragent.Request) (manageragent.Response, error) {
	if req.Kind == "unsupported_action" {
		return manageragent.Response{}, errors.New("unsupported manager action")
	}
	if req.Kind == "create_task" {
		a.tasks["task-created"] = fakeHTTPTask{
			ID:      "task-created",
			Summary: req.Summary,
			State:   "completed",
		}
	}

	return manageragent.Response{
		Kind:    "completion",
		TaskID:  "task-created",
		Summary: req.Summary,
	}, nil
}

func (a *fakeManagerHTTPApp) TaskStatus(ctx context.Context, projectID, taskID string) (manageragent.TaskStatusView, error) {
	if taskID == "missing" {
		return manageragent.TaskStatusView{}, sql.ErrNoRows
	}
	task := a.tasks[taskID]

	return manageragent.TaskStatusView{
		TaskID:          task.ID,
		ProjectID:       "project-1",
		ModuleID:        "module-1",
		Summary:         task.Summary,
		State:           task.State,
		Priority:        task.Priority,
		RunID:           "run-1",
		RunState:        "completed",
		PendingApproval: false,
	}, nil
}

func (a *fakeManagerHTTPApp) TaskWorkbench(ctx context.Context, projectID, taskID string) (manageragent.TaskWorkbenchView, error) {
	expectedProjectID := expectedManagerTaskProject(taskID)
	if taskID == "missing" {
		return manageragent.TaskWorkbenchView{}, sql.ErrNoRows
	}
	if projectID != expectedProjectID {
		return manageragent.TaskWorkbenchView{}, fmt.Errorf("task %s does not belong to project %s", taskID, projectID)
	}

	view := manageragent.TaskWorkbenchView{
		TaskID:               taskID,
		ProjectID:            projectID,
		ModuleID:             "module-1",
		Summary:              "Review push",
		TaskState:            "ready",
		Priority:             3,
		WriteScope:           "repo:" + projectID,
		TaskType:             "write",
		Acceptance:           "Review push",
		ApprovalWorkbenchURL: "/board/approvals/workbench?project_id=" + projectID,
		Artifacts:            []manageragent.TaskWorkbenchArtifact{},
		AvailableActions: []manageragent.TaskWorkbenchAction{
			{ActionID: "dispatch", Enabled: true},
			{ActionID: "cancel", Enabled: true},
			{ActionID: "reprioritize", Enabled: true, CurrentValue: 3},
			{ActionID: "retry", DisabledReason: "Task not failed"},
		},
		DisabledReasons: map[string]string{"retry": "Task not failed"},
	}

	switch taskID {
	case "task-workbench":
		view.ModuleID = "module-2"
		view.TaskState = "waiting_approval"
		view.LatestApprovalID = "approval-9"
		view.LatestApprovalState = "pending"
		view.LatestApprovalReason = "git push origin main requires approval"
		view.ApprovalWorkbenchURL = "/board/approvals/workbench?project_id=project-2&approval_id=approval-9"
		view.AvailableActions[0] = manageragent.TaskWorkbenchAction{ActionID: "dispatch", DisabledReason: "Waiting approval"}
		view.DisabledReasons["dispatch"] = "Waiting approval"
	case "task-failed":
		view.TaskState = "failed"
		view.LatestRunID = "run-failed"
		view.LatestRunState = "failed"
		view.AvailableActions[0] = manageragent.TaskWorkbenchAction{ActionID: "dispatch", DisabledReason: "Use retry for failed tasks"}
		view.AvailableActions[3] = manageragent.TaskWorkbenchAction{ActionID: "retry", Enabled: true}
		view.DisabledReasons["dispatch"] = "Use retry for failed tasks"
		delete(view.DisabledReasons, "retry")
	case "task-completed":
		view.TaskState = "completed"
		view.LatestRunID = "run-done"
		view.LatestRunState = "completed"
		view.AvailableActions[0] = manageragent.TaskWorkbenchAction{ActionID: "dispatch", DisabledReason: "Already completed"}
		view.AvailableActions[1] = manageragent.TaskWorkbenchAction{ActionID: "cancel", DisabledReason: "Already completed"}
		view.AvailableActions[2] = manageragent.TaskWorkbenchAction{ActionID: "reprioritize", DisabledReason: "Already completed", CurrentValue: 3}
		view.DisabledReasons["dispatch"] = "Already completed"
		view.DisabledReasons["cancel"] = "Already completed"
		view.DisabledReasons["reprioritize"] = "Already completed"
	case "task-canceled":
		view.TaskState = "canceled"
		view.AvailableActions[0] = manageragent.TaskWorkbenchAction{ActionID: "dispatch", DisabledReason: "Task canceled"}
		view.AvailableActions[1] = manageragent.TaskWorkbenchAction{ActionID: "cancel", DisabledReason: "Task canceled"}
		view.AvailableActions[2] = manageragent.TaskWorkbenchAction{ActionID: "reprioritize", DisabledReason: "Task canceled", CurrentValue: 3}
		view.AvailableActions[3] = manageragent.TaskWorkbenchAction{ActionID: "retry", DisabledReason: "Task canceled"}
		view.DisabledReasons = map[string]string{
			"dispatch":     "Task canceled",
			"cancel":       "Task canceled",
			"reprioritize": "Task canceled",
			"retry":        "Task canceled",
		}
	}

	return view, nil
}

func (a *fakeManagerHTTPApp) BoardSnapshot(ctx context.Context, projectID string) (manageragent.BoardSnapshotView, error) {
	return manageragent.BoardSnapshotView{
		ProjectID: projectID,
		Modules: map[string][]manageragent.ModuleSnapshot{
			"Implementing": {
				{ModuleID: "module-1", Name: "Bootstrap", State: "active"},
			},
		},
		Tasks: map[string][]manageragent.TaskSnapshot{
			"Done": {
				{TaskID: "task-1", ModuleID: "module-1", Summary: "Review push", State: "completed", Priority: 1},
			},
		},
	}, nil
}

func (a *fakeManagerHTTPApp) ApprovalWorkbenchQueue(ctx context.Context, projectID string) (manageragent.ApprovalWorkbenchQueueView, error) {
	return manageragent.ApprovalWorkbenchQueueView{
		Items: []manageragent.ApprovalWorkbenchItem{
			{
				ApprovalID: "approval-1",
				TaskID:     "task-1",
				Summary:    "Review push",
				RiskLevel:  "high",
				Priority:   9,
			},
		},
	}, nil
}

func (a *fakeManagerHTTPApp) ApprovalWorkbenchDetail(ctx context.Context, approvalID string) (manageragent.ApprovalWorkbenchDetailView, error) {
	switch approvalID {
	case "missing":
		return manageragent.ApprovalWorkbenchDetailView{}, command.ErrApprovalActionNotFound
	case "approval-approved":
		return manageragent.ApprovalWorkbenchDetailView{
			ApprovalID:              "approval-approved",
			TaskID:                  "task-1",
			Summary:                 "Review push",
			ApprovalState:           "approved",
			RiskLevel:               "high",
			TaskState:               "completed",
			RunID:                   "run-2",
			RunState:                "completed",
			RunDetailURL:            "/board/runs/run-2",
			AssistantSummaryPreview: "approved run completed",
		}, nil
	case "approval-rejected":
		return manageragent.ApprovalWorkbenchDetailView{
			ApprovalID:              "approval-rejected",
			TaskID:                  "task-1",
			Summary:                 "Review push",
			ApprovalState:           "rejected",
			RiskLevel:               "high",
			RejectionReason:         "manager rejected the action",
			TaskState:               "ready",
			AssistantSummaryPreview: "rejected by manager",
		}, nil
	default:
		return manageragent.ApprovalWorkbenchDetailView{
			ApprovalID:              "approval-1",
			TaskID:                  "task-1",
			Summary:                 "Review push",
			ApprovalState:           "pending",
			RiskLevel:               "high",
			TaskState:               "waiting_approval",
			AssistantSummaryPreview: "waiting on manager action",
		}, nil
	}
}

func (a *fakeManagerHTTPApp) ApproveApproval(ctx context.Context, approvalID string) (manageragent.ApprovalWorkbenchActionResponse, error) {
	switch approvalID {
	case "missing":
		return manageragent.ApprovalWorkbenchActionResponse{}, command.ErrApprovalActionNotFound
	case "conflict":
		return manageragent.ApprovalWorkbenchActionResponse{}, command.ErrApprovalActionConflict
	default:
		return manageragent.ApprovalWorkbenchActionResponse{
			ApprovalID:    approvalID,
			ApprovalState: "approved",
			TaskID:        "task-1",
			TaskState:     "completed",
			RunID:         "run-3",
			RunState:      "completed",
		}, nil
	}
}

func (a *fakeManagerHTTPApp) RejectApproval(ctx context.Context, approvalID, rejectionReason string) (manageragent.ApprovalWorkbenchActionResponse, error) {
	switch approvalID {
	case "missing":
		return manageragent.ApprovalWorkbenchActionResponse{}, command.ErrApprovalActionNotFound
	case "conflict":
		return manageragent.ApprovalWorkbenchActionResponse{}, command.ErrApprovalActionConflict
	case "approval-rejected":
		return manageragent.ApprovalWorkbenchActionResponse{
			ApprovalID:      approvalID,
			ApprovalState:   "rejected",
			RejectionReason: "stored persisted rejection reason",
			TaskID:          "task-1",
			TaskState:       "ready",
		}, nil
	default:
		return manageragent.ApprovalWorkbenchActionResponse{
			ApprovalID:      approvalID,
			ApprovalState:   "rejected",
			RejectionReason: rejectionReason,
			TaskID:          "task-1",
			TaskState:       "ready",
		}, nil
	}
}

func (a *fakeManagerHTTPApp) RetryApprovalDispatch(ctx context.Context, approvalID string) (manageragent.ApprovalWorkbenchActionResponse, error) {
	switch approvalID {
	case "missing":
		return manageragent.ApprovalWorkbenchActionResponse{}, command.ErrApprovalActionNotFound
	case "conflict":
		return manageragent.ApprovalWorkbenchActionResponse{}, command.ErrApprovalActionConflict
	default:
		return manageragent.ApprovalWorkbenchActionResponse{
			ApprovalID:    approvalID,
			ApprovalState: "approved",
			TaskID:        "task-1",
			TaskState:     "running",
			RunID:         "run-4",
			RunState:      "running",
		}, nil
	}
}

func (a *fakeManagerHTTPApp) DispatchTaskWorkbench(ctx context.Context, projectID, taskID string) (manageragent.TaskWorkbenchActionResponse, error) {
	if projectID != expectedManagerTaskProject(taskID) {
		return manageragent.TaskWorkbenchActionResponse{}, fmt.Errorf("%w: task %s not found in project %s", manageragent.ErrTaskActionNotFound, taskID, projectID)
	}
	if taskID == "task-workbench" {
		return manageragent.TaskWorkbenchActionResponse{}, fmt.Errorf("%w: dispatch unavailable: Waiting approval", manageragent.ErrTaskActionConflict)
	}

	return manageragent.TaskWorkbenchActionResponse{
		TaskID:          taskID,
		TaskState:       "completed",
		LatestRunID:     "run-dispatch",
		LatestRunState:  "completed",
		RefreshRequired: true,
	}, nil
}

func (a *fakeManagerHTTPApp) RetryTaskWorkbench(ctx context.Context, projectID, taskID string) (manageragent.TaskWorkbenchActionResponse, error) {
	if projectID != expectedManagerTaskProject(taskID) {
		return manageragent.TaskWorkbenchActionResponse{}, fmt.Errorf("%w: task %s not found in project %s", manageragent.ErrTaskActionNotFound, taskID, projectID)
	}
	if taskID == "task-ready" {
		return manageragent.TaskWorkbenchActionResponse{}, fmt.Errorf("%w: retry unavailable: Task not failed", manageragent.ErrTaskActionConflict)
	}

	return manageragent.TaskWorkbenchActionResponse{
		TaskID:          taskID,
		TaskState:       "ready",
		RefreshRequired: true,
	}, nil
}

func (a *fakeManagerHTTPApp) CancelTaskWorkbench(ctx context.Context, projectID, taskID string) (manageragent.TaskWorkbenchActionResponse, error) {
	if projectID != expectedManagerTaskProject(taskID) {
		return manageragent.TaskWorkbenchActionResponse{}, fmt.Errorf("%w: task %s not found in project %s", manageragent.ErrTaskActionNotFound, taskID, projectID)
	}
	if taskID == "task-completed" {
		return manageragent.TaskWorkbenchActionResponse{}, fmt.Errorf("%w: cancel unavailable: Already completed", manageragent.ErrTaskActionConflict)
	}

	return manageragent.TaskWorkbenchActionResponse{
		TaskID:          taskID,
		TaskState:       "canceled",
		RefreshRequired: true,
	}, nil
}

func (a *fakeManagerHTTPApp) ReprioritizeTaskWorkbench(ctx context.Context, projectID, taskID string, priority int) (manageragent.TaskWorkbenchActionResponse, error) {
	if projectID != expectedManagerTaskProject(taskID) {
		return manageragent.TaskWorkbenchActionResponse{}, fmt.Errorf("%w: task %s not found in project %s", manageragent.ErrTaskActionNotFound, taskID, projectID)
	}
	if taskID == "task-canceled" {
		return manageragent.TaskWorkbenchActionResponse{}, fmt.Errorf("%w: reprioritize unavailable: Task canceled", manageragent.ErrTaskActionConflict)
	}

	a.lastReprioritizePriority = priority
	return manageragent.TaskWorkbenchActionResponse{
		TaskID:          taskID,
		TaskState:       "ready",
		RefreshRequired: true,
		Message:         fmt.Sprintf("priority updated to %d", priority),
	}, nil
}

func expectedManagerTaskProject(taskID string) string {
	switch taskID {
	case "task-workbench":
		return "project-2"
	default:
		return "project-1"
	}
}

func managerTaskWorkbenchAction(actions []manageragent.TaskWorkbenchAction, actionID string) manageragent.TaskWorkbenchAction {
	for _, action := range actions {
		if action.ActionID == actionID {
			return action
		}
	}
	return manageragent.TaskWorkbenchAction{}
}
