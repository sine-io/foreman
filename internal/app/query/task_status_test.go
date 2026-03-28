package query

import (
	"database/sql"
	"testing"

	"github.com/sine-io/foreman/internal/domain/approval"
	modulepkg "github.com/sine-io/foreman/internal/domain/module"
	"github.com/sine-io/foreman/internal/domain/task"
	"github.com/sine-io/foreman/internal/ports"
	"github.com/stretchr/testify/require"
)

func TestTaskStatusIncludesRunAndApprovalFields(t *testing.T) {
	q := NewTaskStatusQuery(fakeStatusRepo{
		task: task.Task{
			ID:       "task-1",
			ModuleID: "module-1",
			Summary:  "Ship dedicated status query",
			State:    task.TaskStateWaitingApproval,
		},
		module: modulepkg.Module{
			ID:        "module-1",
			ProjectID: "project-1",
		},
		run: ports.Run{
			ID:     "run-1",
			TaskID: "task-1",
			State:  "running",
		},
		latestApproval: approval.Approval{
			ID:     "approval-1",
			TaskID: "task-1",
			Reason: "git push origin main requires approval",
			Status: approval.StatusApproved,
		},
	})

	view, err := q.Execute("project-1", "task-1")
	require.NoError(t, err)
	require.Equal(t, "task-1", view.TaskID)
	require.Equal(t, "project-1", view.ProjectID)
	require.Equal(t, "module-1", view.ModuleID)
	require.Equal(t, "Ship dedicated status query", view.Summary)
	require.Equal(t, "waiting_approval", view.State)
	require.Equal(t, "run-1", view.RunID)
	require.Equal(t, "running", view.RunState)
	require.Equal(t, "approval-1", view.ApprovalID)
	require.Equal(t, "git push origin main requires approval", view.ApprovalReason)
	require.Equal(t, "approved", view.ApprovalState)
}

func TestTaskStatusReturnsNotFoundForCrossProjectTask(t *testing.T) {
	q := NewTaskStatusQuery(fakeStatusRepo{
		task: task.Task{
			ID:       "task-1",
			ModuleID: "module-1",
		},
		module: modulepkg.Module{
			ID:        "module-1",
			ProjectID: "project-2",
		},
	})

	_, err := q.Execute("project-1", "task-1")
	require.EqualError(t, err, "task task-1 does not belong to project project-1")
}

type fakeStatusRepo struct {
	task            task.Task
	module          modulepkg.Module
	run             ports.Run
	pendingApproval approval.Approval
	latestApproval  approval.Approval
}

func (f fakeStatusRepo) Task(id string) (task.Task, error) {
	if f.task.ID != id {
		return task.Task{}, sql.ErrNoRows
	}

	return f.task, nil
}

func (f fakeStatusRepo) Module(id string) (modulepkg.Module, error) {
	if f.module.ID != id {
		return modulepkg.Module{}, sql.ErrNoRows
	}

	return f.module, nil
}

func (f fakeStatusRepo) FindByTask(taskID string) (ports.Run, error) {
	if f.run.TaskID != taskID {
		return ports.Run{}, sql.ErrNoRows
	}

	return f.run, nil
}

func (f fakeStatusRepo) FindPendingByTask(taskID string) (approval.Approval, error) {
	if f.pendingApproval.TaskID != taskID {
		return approval.Approval{}, sql.ErrNoRows
	}

	return f.pendingApproval, nil
}

func (f fakeStatusRepo) FindLatestByTask(taskID string) (approval.Approval, error) {
	if f.latestApproval.TaskID != taskID {
		return approval.Approval{}, sql.ErrNoRows
	}

	return f.latestApproval, nil
}
