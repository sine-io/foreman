package manageragent

import (
	"context"
	"database/sql"
	"testing"

	"github.com/sine-io/foreman/internal/app/command"
	"github.com/sine-io/foreman/internal/app/query"
	"github.com/sine-io/foreman/internal/domain/approval"
	domainpolicy "github.com/sine-io/foreman/internal/domain/policy"
	"github.com/sine-io/foreman/internal/domain/task"
	"github.com/sine-io/foreman/internal/ports"
	"github.com/stretchr/testify/require"
)

func TestHandleCreateTaskReturnsCompletionWhenDispatchFinishes(t *testing.T) {
	svc := newTestService(domainpolicy.Decision{}, nil)

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
	svc := newTestService(domainpolicy.Decision{
		RequiresApproval: true,
		Reason:           "git push origin main requires approval",
	}, nil)

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
	svc := newTestService(domainpolicy.Decision{}, []ports.TaskBoardRow{
		{
			TaskID:          "task-1",
			ModuleID:        "module-default",
			Summary:         "Summarize the module status",
			State:           "waiting_approval",
			Priority:        10,
			PendingApproval: true,
		},
	})

	view, err := svc.TaskStatus(context.Background(), "task-1")
	require.NoError(t, err)
	require.Equal(t, "task-1", view.TaskID)
	require.Equal(t, "waiting_approval", view.State)
	require.True(t, view.PendingApproval)
}

func newTestService(policyDecision domainpolicy.Decision, boardRows []ports.TaskBoardRow) *Service {
	tasks := newFakeTaskRepo()
	approvals := &fakeApprovalRepo{byTaskID: map[string]approval.Approval{}}

	return NewService(
		command.NewCreateTaskHandler(tasks),
		command.NewDispatchTaskHandler(
			tasks,
			&fakeLeaseRepo{},
			fakePolicy{decision: policyDecision},
			fakeRunner{},
			approvals,
			&fakeRunRepo{},
			&fakeArtifactRepo{},
		),
		query.NewTaskBoardQuery(fakeBoardQueryRepo{tasks: boardRows}),
	)
}

type fakeTaskRepo struct {
	byID map[string]task.Task
}

func newFakeTaskRepo() *fakeTaskRepo {
	return &fakeTaskRepo{
		byID: map[string]task.Task{},
	}
}

func (f *fakeTaskRepo) Save(value task.Task) error {
	f.byID[value.ID] = value
	return nil
}

func (f *fakeTaskRepo) Get(id string) (task.Task, error) {
	value, ok := f.byID[id]
	if !ok {
		return task.Task{}, sql.ErrNoRows
	}

	return value, nil
}

type fakeApprovalRepo struct {
	byTaskID map[string]approval.Approval
	saved    approval.Approval
}

func (f *fakeApprovalRepo) Save(value approval.Approval) error {
	f.saved = value
	f.byTaskID[value.TaskID] = value
	return nil
}

func (f *fakeApprovalRepo) Get(id string) (approval.Approval, error) {
	if f.saved.ID == id {
		return f.saved, nil
	}

	for _, value := range f.byTaskID {
		if value.ID == id {
			return value, nil
		}
	}

	return approval.Approval{}, sql.ErrNoRows
}

func (f *fakeApprovalRepo) FindPendingByTask(taskID string) (approval.Approval, error) {
	value, ok := f.byTaskID[taskID]
	if !ok {
		return approval.Approval{}, sql.ErrNoRows
	}

	if value.Status != approval.StatusPending {
		return approval.Approval{}, sql.ErrNoRows
	}

	return value, nil
}

type fakeLeaseRepo struct{}

func (f *fakeLeaseRepo) Acquire(taskID, scopeKey string) error {
	return nil
}

func (f *fakeLeaseRepo) Release(scopeKey string) error {
	return nil
}

type fakePolicy struct {
	decision domainpolicy.Decision
}

func (f fakePolicy) Evaluate(action string) domainpolicy.Decision {
	return f.decision
}

type fakeRunner struct{}

func (fakeRunner) Dispatch(req ports.RunRequest) (ports.Run, error) {
	return ports.Run{
		ID:                   "run-1",
		TaskID:               req.TaskID,
		RunnerKind:           "codex",
		State:                "completed",
		AssistantSummaryPath: "artifacts/tasks/task-1/assistant_summary.txt",
	}, nil
}

func (fakeRunner) Observe(runID string) (ports.Run, error) {
	return ports.Run{}, nil
}

func (fakeRunner) Stop(runID string) error {
	return nil
}

type fakeRunRepo struct {
	saved ports.Run
}

func (f *fakeRunRepo) Save(run ports.Run) error {
	f.saved = run
	return nil
}

func (f *fakeRunRepo) Get(id string) (ports.Run, error) {
	if f.saved.ID != id {
		return ports.Run{}, sql.ErrNoRows
	}

	return f.saved, nil
}

type fakeArtifactRepo struct{}

func (f *fakeArtifactRepo) Create(taskID, kind, path string) (string, error) {
	return "artifact-1", nil
}

func (f *fakeArtifactRepo) Get(id string) (ports.ArtifactRecord, error) {
	if id == "" {
		return ports.ArtifactRecord{}, sql.ErrNoRows
	}

	return ports.ArtifactRecord{ID: id}, nil
}

type fakeBoardQueryRepo struct {
	tasks []ports.TaskBoardRow
}

func (f fakeBoardQueryRepo) ListModules(projectID string) ([]ports.ModuleBoardRow, error) {
	return nil, nil
}

func (f fakeBoardQueryRepo) ListTasks(projectID string) ([]ports.TaskBoardRow, error) {
	return f.tasks, nil
}

func (f fakeBoardQueryRepo) GetRunDetail(runID string) (ports.RunDetailRecord, error) {
	return ports.RunDetailRecord{}, sql.ErrNoRows
}

func (f fakeBoardQueryRepo) ListApprovals(projectID string) ([]ports.ApprovalQueueRow, error) {
	return nil, nil
}
