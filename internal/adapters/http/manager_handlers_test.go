package http

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sine-io/foreman/internal/app/manageragent"
	"github.com/stretchr/testify/require"
)

func TestManagerCommandEndpointCreatesTaskThroughNormalizedService(t *testing.T) {
	router := NewRouter(newFakeHTTPApp())

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
	router := NewRouter(newFakeHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/api/manager/tasks/task-1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	var resp managerTaskStatusResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "task-1", resp.TaskID)
	require.Equal(t, "project-1", resp.ProjectID)
	require.Equal(t, "run-1", resp.RunID)
	require.Equal(t, "completed", resp.RunState)
}

func TestManagerBoardSnapshotEndpointReturnsBoardShape(t *testing.T) {
	router := NewRouter(newFakeHTTPApp())

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
	router := NewRouter(newFakeHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/api/manager/tasks/missing", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusNotFound, rec.Code)
}

func TestManagerCommandEndpointMapsClientErrorsTo400(t *testing.T) {
	router := NewRouter(newFakeHTTPApp())

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

func (a *fakeHTTPApp) Handle(ctx context.Context, req manageragent.Request) (manageragent.Response, error) {
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

func (a *fakeHTTPApp) TaskStatus(ctx context.Context, projectID, taskID string) (manageragent.TaskStatusView, error) {
	if taskID == "missing" {
		return manageragent.TaskStatusView{}, sql.ErrNoRows
	}
	task := a.tasks[taskID]

	return manageragent.TaskStatusView{
		TaskID:    task.ID,
		ProjectID: "project-1",
		ModuleID:  "module-1",
		Summary:   task.Summary,
		State:     task.State,
		RunID:     "run-1",
		RunState:  "completed",
	}, nil
}

func (a *fakeHTTPApp) BoardSnapshot(ctx context.Context, projectID string) (manageragent.BoardSnapshotView, error) {
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
