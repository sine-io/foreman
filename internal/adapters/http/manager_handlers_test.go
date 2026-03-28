package http

import (
	"context"
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
	require.Contains(t, rec.Body.String(), "completion")
}

func TestManagerTaskStatusEndpointReturnsTaskSnapshot(t *testing.T) {
	router := NewRouter(newFakeHTTPApp())

	req := httptest.NewRequest(stdhttp.MethodGet, "/api/manager/tasks/task-1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "task-1")
}

func (a *fakeHTTPApp) Handle(ctx context.Context, req manageragent.Request) (manageragent.Response, error) {
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
	task := a.tasks[taskID]

	return manageragent.TaskStatusView{
		TaskID:    task.ID,
		ProjectID: "project-1",
		ModuleID:  "module-1",
		Summary:   task.Summary,
		State:     task.State,
		Priority:  task.Priority,
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
