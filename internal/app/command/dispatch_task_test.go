package command

import (
	"database/sql"
	"testing"

	domainpolicy "github.com/sine-io/foreman/internal/domain/policy"
	"github.com/sine-io/foreman/internal/domain/task"
	"github.com/sine-io/foreman/internal/ports"
	"github.com/stretchr/testify/require"
)

func TestDispatchAcquiresRepoLeaseAndStartsRun(t *testing.T) {
	tasks := newFakeTaskRepo()
	repoTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "Implement board query", "repo:project-1")
	require.NoError(t, tasks.Save(repoTask))

	leases := &fakeLeaseRepo{}
	runs := &fakeRunRepo{}
	artifacts := &fakeArtifactRepo{}
	handler := NewDispatchTaskHandler(
		tasks,
		leases,
		fakePolicy{decision: domainpolicy.Decision{}},
		fakeRunner{},
		&fakeApprovalRepo{},
		runs,
		artifacts,
	)

	out, err := handler.Handle(DispatchTaskCommand{TaskID: "task-1"})
	require.NoError(t, err)
	require.Equal(t, "running", out.RunState)
	require.Equal(t, "repo:project-1", leases.scopeKey)
	require.Equal(t, "task-1", runs.saved.TaskID)
}

func TestDispatchCreatesApprovalWhenRiskyActionDetected(t *testing.T) {
	tasks := newFakeTaskRepo()
	riskyTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "git push origin main", "repo:project-1")
	require.NoError(t, tasks.Save(riskyTask))

	approvals := &fakeApprovalRepo{}
	handler := NewDispatchTaskHandler(
		tasks,
		&fakeLeaseRepo{},
		fakePolicy{
			decision: domainpolicy.Decision{
				RequiresApproval: true,
				Reason:           "git push origin main requires approval",
			},
		},
		fakeRunner{},
		approvals,
		&fakeRunRepo{},
		&fakeArtifactRepo{},
	)

	out, err := handler.Handle(DispatchTaskCommand{TaskID: "task-1"})
	require.NoError(t, err)
	require.Equal(t, "waiting_approval", out.TaskState)
	require.Equal(t, "git push origin main requires approval", approvals.saved.Reason)
}

func TestDispatchIndexesAssistantSummaryArtifact(t *testing.T) {
	tasks := newFakeTaskRepo()
	repoTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "Implement board query", "repo:project-1")
	require.NoError(t, tasks.Save(repoTask))

	handler := NewDispatchTaskHandler(
		tasks,
		&fakeLeaseRepo{},
		fakePolicy{decision: domainpolicy.Decision{}},
		fakeRunner{},
		&fakeApprovalRepo{},
		&fakeRunRepo{},
		&fakeArtifactRepo{},
	)

	out, err := handler.Handle(DispatchTaskCommand{TaskID: "task-1"})
	require.NoError(t, err)
	require.NotEmpty(t, out.ArtifactIDs)
}

type fakeLeaseRepo struct {
	taskID   string
	scopeKey string
}

func (f *fakeLeaseRepo) Acquire(taskID, scopeKey string) error {
	f.taskID = taskID
	f.scopeKey = scopeKey
	return nil
}

func (f *fakeLeaseRepo) Release(scopeKey string) error {
	f.scopeKey = scopeKey
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
		State:                "running",
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

type fakeArtifactRepo struct {
	nextID string
}

func (f *fakeArtifactRepo) Create(taskID, kind, path string) (string, error) {
	if f.nextID == "" {
		f.nextID = "artifact-1"
	}
	return f.nextID, nil
}

func (f *fakeArtifactRepo) Get(id string) (ports.ArtifactRecord, error) {
	if id == "" {
		return ports.ArtifactRecord{}, sql.ErrNoRows
	}
	return ports.ArtifactRecord{ID: id}, nil
}
