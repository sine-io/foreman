package command

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/sine-io/foreman/internal/domain/approval"
	"github.com/sine-io/foreman/internal/domain/task"
	"github.com/stretchr/testify/require"
)

func TestCreateTaskPersistsReadyTask(t *testing.T) {
	repo := newFakeTaskRepo()
	handler := NewCreateTaskHandler(repo)

	out, err := handler.Handle(CreateTaskCommand{
		ModuleID:   "module-1",
		Title:      "Implement board query",
		TaskType:   "write",
		WriteScope: "repo:project-1",
		Acceptance: "Board query returns module columns",
		Priority:   10,
	})
	require.NoError(t, err)
	require.Equal(t, "ready", out.State)

	saved, err := repo.Get(out.ID)
	require.NoError(t, err)
	require.Equal(t, "Implement board query", saved.Summary)
	require.Equal(t, 10, saved.Priority)
}

func TestApproveTaskMarksApprovalResolved(t *testing.T) {
	approvals := &fakeApprovalRepo{
		byTaskID: map[string]approval.Approval{
			"task-1": approval.New("approval-1", "task-1", "git push origin main"),
		},
	}
	tasks := newFakeTaskRepo()
	require.NoError(t, tasks.Save(task.NewTask("task-1", "module-1", task.TaskTypeWrite, "Implement board", "repo:project-1")))

	handler := NewApproveTaskHandler(approvals, tasks)
	err := handler.Handle(ApproveTaskCommand{TaskID: "task-1"})
	require.NoError(t, err)

	_, err = approvals.FindPendingByTask("task-1")
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.Equal(t, approval.StatusApproved, approvals.saved.Status)

	savedTask, err := tasks.Get("task-1")
	require.NoError(t, err)
	require.Equal(t, task.TaskStateLeased, savedTask.State)
}

func TestRetryTaskMovesFailedTaskToReady(t *testing.T) {
	tasks := newFakeTaskRepo()
	failedTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "Retry me", "repo:project-1")
	failedTask.State = task.TaskStateFailed
	require.NoError(t, tasks.Save(failedTask))

	handler := NewRetryTaskHandler(tasks)
	err := handler.Handle(RetryTaskCommand{TaskID: "task-1"})
	require.NoError(t, err)

	savedTask, err := tasks.Get("task-1")
	require.NoError(t, err)
	require.Equal(t, task.TaskStateReady, savedTask.State)
}

func TestCancelTaskMovesTaskToCanceled(t *testing.T) {
	tasks := newFakeTaskRepo()
	require.NoError(t, tasks.Save(task.NewTask("task-1", "module-1", task.TaskTypeWrite, "Cancel me", "repo:project-1")))

	handler := NewCancelTaskHandler(tasks)
	err := handler.Handle(CancelTaskCommand{TaskID: "task-1"})
	require.NoError(t, err)

	savedTask, err := tasks.Get("task-1")
	require.NoError(t, err)
	require.Equal(t, task.TaskStateCanceled, savedTask.State)
}

func TestReprioritizeTaskUpdatesPriority(t *testing.T) {
	tasks := newFakeTaskRepo()
	repoTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "Prioritize me", "repo:project-1")
	require.NoError(t, tasks.Save(repoTask))

	handler := NewReprioritizeTaskHandler(tasks)
	err := handler.Handle(ReprioritizeTaskCommand{TaskID: "task-1", Priority: 42})
	require.NoError(t, err)

	savedTask, err := tasks.Get("task-1")
	require.NoError(t, err)
	require.Equal(t, 42, savedTask.Priority)
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
	delete(f.byTaskID, value.TaskID)
	return nil
}

func (f *fakeApprovalRepo) Get(id string) (approval.Approval, error) {
	for _, value := range f.byTaskID {
		if value.ID == id {
			return value, nil
		}
	}

	if f.saved.ID == id {
		return f.saved, nil
	}

	return approval.Approval{}, sql.ErrNoRows
}

func (f *fakeApprovalRepo) FindPendingByTask(taskID string) (approval.Approval, error) {
	value, ok := f.byTaskID[taskID]
	if !ok {
		return approval.Approval{}, sql.ErrNoRows
	}

	if value.Status != approval.StatusPending {
		return approval.Approval{}, errors.New("approval is not pending")
	}

	return value, nil
}
