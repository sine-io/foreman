package test

import (
	"context"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sine-io/foreman/internal/adapters/gateway/openclaw"
	httpadapter "github.com/sine-io/foreman/internal/adapters/http"
	"github.com/sine-io/foreman/internal/app/command"
	"github.com/sine-io/foreman/internal/app/query"
	"github.com/stretchr/testify/require"
)

func TestPhase1FlowFromOpenClawCommandToBoardState(t *testing.T) {
	app := &e2eApp{tasks: map[string]e2eTask{}}
	router := httpadapter.NewRouter(app)

	req := httptest.NewRequest(
		stdhttp.MethodPost,
		"/gateways/openclaw/command",
		strings.NewReader(`{"session_id":"oc-1","action":"create_task","summary":"Bootstrap board"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, stdhttp.StatusOK, rec.Code)

	req = httptest.NewRequest(stdhttp.MethodGet, "/board/tasks?project_id=demo", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "Bootstrap board")
}

type e2eTask struct {
	ID      string
	Summary string
	State   string
}

type e2eApp struct {
	tasks map[string]e2eTask
}

func (a *e2eApp) ModuleBoard(projectID string) (query.ModuleBoardView, error) {
	return query.ModuleBoardView{}, nil
}

func (a *e2eApp) TaskBoard(projectID string) (query.TaskBoardView, error) {
	view := query.TaskBoardView{Columns: map[string][]query.TaskCard{"Ready": {}}}
	for _, task := range a.tasks {
		view.Columns["Ready"] = append(view.Columns["Ready"], query.TaskCard{
			ID:      task.ID,
			Summary: task.Summary,
			State:   task.State,
		})
	}
	return view, nil
}

func (a *e2eApp) RunDetail(runID string) (query.RunDetailView, error) {
	return query.RunDetailView{}, nil
}

func (a *e2eApp) ApproveTask(cmd command.ApproveTaskCommand) (string, error) {
	return "leased", nil
}

func (a *e2eApp) RetryTask(cmd command.RetryTaskCommand) (string, error) {
	return "ready", nil
}

func (a *e2eApp) CancelTask(cmd command.CancelTaskCommand) (string, error) {
	return "canceled", nil
}

func (a *e2eApp) ReprioritizeTask(cmd command.ReprioritizeTaskCommand) (string, error) {
	return "ready", nil
}

func (a *e2eApp) OpenClawCommand(ctx context.Context, env openclaw.Envelope) (openclaw.Response, error) {
	a.tasks["task-openclaw"] = e2eTask{
		ID:      "task-openclaw",
		Summary: env.Summary,
		State:   "ready",
	}
	return openclaw.Response{Kind: "completion", TaskID: "task-openclaw", Summary: env.Summary}, nil
}
