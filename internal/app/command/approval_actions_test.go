package command

import (
	"errors"
	"testing"

	"github.com/sine-io/foreman/internal/domain/approval"
	domainpolicy "github.com/sine-io/foreman/internal/domain/policy"
	"github.com/sine-io/foreman/internal/domain/task"
	"github.com/sine-io/foreman/internal/ports"
	"github.com/stretchr/testify/require"
)

func TestApproveApprovalDispatchesImmediatelyByApprovalID(t *testing.T) {
	tasks := newFakeTaskRepo()
	repoTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "git push origin main", "repo:project-1")
	repoTask.State = task.TaskStateWaitingApproval
	require.NoError(t, tasks.Save(repoTask))

	approvals := &fakeApprovalRepo{
		byTaskID: map[string]approval.Approval{
			"task-1": approval.New("approval-1", "task-1", "git push origin main requires approval"),
		},
	}
	runner := &fakeRunner{}
	dispatch := newApprovalTestDispatchHandler(tasks, approvals, runner)
	handler := NewApproveApprovalHandler(newFakeTransactor(tasks, approvals, &fakeRunRepo{}, &fakeArtifactRepo{}), approvals, tasks, dispatch)

	out, err := handler.Handle(ApproveApprovalCommand{ApprovalID: "approval-1"})
	require.NoError(t, err)
	require.Equal(t, "approval-1", out.ApprovalID)
	require.Equal(t, string(approval.StatusApproved), out.ApprovalStatus)
	require.Equal(t, "completed", out.TaskState)
	require.Equal(t, "completed", out.RunState)
	require.Equal(t, 1, runner.dispatchCount)

	savedApproval, err := approvals.Get("approval-1")
	require.NoError(t, err)
	require.Equal(t, approval.StatusApproved, savedApproval.Status)

	savedTask, err := tasks.Get("task-1")
	require.NoError(t, err)
	require.Equal(t, task.TaskStateCompleted, savedTask.State)
}

func TestApproveApprovalMarksApprovedPendingDispatchWhenPostApprovalDispatchFails(t *testing.T) {
	tasks := newFakeTaskRepo()
	repoTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "git push origin main", "repo:project-1")
	repoTask.State = task.TaskStateWaitingApproval
	require.NoError(t, tasks.Save(repoTask))

	approvals := &fakeApprovalRepo{
		byTaskID: map[string]approval.Approval{
			"task-1": approval.New("approval-1", "task-1", "git push origin main requires approval"),
		},
	}
	runner := &fakeRunner{dispatchErr: errors.New("runner unavailable")}
	dispatch := newApprovalTestDispatchHandler(tasks, approvals, runner)
	handler := NewApproveApprovalHandler(newFakeTransactor(tasks, approvals, &fakeRunRepo{}, &fakeArtifactRepo{}), approvals, tasks, dispatch)

	_, err := handler.Handle(ApproveApprovalCommand{ApprovalID: "approval-1"})
	require.EqualError(t, err, "runner unavailable")

	savedApproval, getErr := approvals.Get("approval-1")
	require.NoError(t, getErr)
	require.Equal(t, approval.StatusApproved, savedApproval.Status)

	savedTask, getErr := tasks.Get("task-1")
	require.NoError(t, getErr)
	require.Equal(t, task.TaskStateApprovedPendingDispatch, savedTask.State)
}

func TestRejectApprovalPersistsReasonAndReturnsTaskToReady(t *testing.T) {
	tasks := newFakeTaskRepo()
	repoTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "git push origin main", "repo:project-1")
	repoTask.State = task.TaskStateWaitingApproval
	require.NoError(t, tasks.Save(repoTask))

	approvals := &fakeApprovalRepo{
		byTaskID: map[string]approval.Approval{
			"task-1": approval.New("approval-1", "task-1", "git push origin main requires approval"),
		},
	}
	handler := NewRejectApprovalHandler(newFakeTransactor(tasks, approvals, &fakeRunRepo{}, &fakeArtifactRepo{}), approvals, tasks)

	out, err := handler.Handle(RejectApprovalCommand{
		ApprovalID: "approval-1",
		Reason:     "manual reviewer rejected the action",
	})
	require.NoError(t, err)
	require.Equal(t, "approval-1", out.ApprovalID)
	require.Equal(t, string(approval.StatusRejected), out.ApprovalStatus)
	require.Equal(t, "ready", out.TaskState)

	savedApproval, getErr := approvals.Get("approval-1")
	require.NoError(t, getErr)
	require.Equal(t, approval.StatusRejected, savedApproval.Status)
	require.Equal(t, "manual reviewer rejected the action", savedApproval.RejectionReason)

	savedTask, getErr := tasks.Get("task-1")
	require.NoError(t, getErr)
	require.Equal(t, task.TaskStateReady, savedTask.State)
}

func TestRetryApprovalDispatchDoesNotCreateANewApproval(t *testing.T) {
	tasks := newFakeTaskRepo()
	repoTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "git push origin main", "repo:project-1")
	repoTask.State = task.TaskStateApprovedPendingDispatch
	require.NoError(t, tasks.Save(repoTask))

	approvals := &fakeApprovalRepo{
		byTaskID: map[string]approval.Approval{
			"task-1": {
				ID:     "approval-1",
				TaskID: "task-1",
				Reason: "git push origin main requires approval",
				Status: approval.StatusApproved,
			},
		},
	}
	runner := &fakeRunner{}
	dispatch := newApprovalTestDispatchHandler(tasks, approvals, runner)
	handler := NewRetryApprovalDispatchHandler(approvals, tasks, dispatch)

	out, err := handler.Handle(RetryApprovalDispatchCommand{ApprovalID: "approval-1"})
	require.NoError(t, err)
	require.Equal(t, "approval-1", out.ApprovalID)
	require.Equal(t, string(approval.StatusApproved), out.ApprovalStatus)
	require.Equal(t, "completed", out.TaskState)
	require.Equal(t, "completed", out.RunState)
	require.Equal(t, 0, approvals.saveCount)
	require.Equal(t, 1, runner.dispatchCount)
}

func TestApprovalActionsEnforceIdempotencyAndConflicts(t *testing.T) {
	t.Run("approve repeated approval is no-op success", func(t *testing.T) {
		tasks := newFakeTaskRepo()
		repoTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "git push origin main", "repo:project-1")
		repoTask.State = task.TaskStateCompleted
		require.NoError(t, tasks.Save(repoTask))

		approvals := &fakeApprovalRepo{
			byTaskID: map[string]approval.Approval{
				"task-1": {
					ID:     "approval-1",
					TaskID: "task-1",
					Reason: "git push origin main requires approval",
					Status: approval.StatusApproved,
				},
			},
		}
		runs := &fakeRunRepo{
			saved: ports.Run{
				ID:                   "run-1",
				TaskID:               "task-1",
				RunnerKind:           "codex",
				State:                "completed",
				AssistantSummaryPath: "artifacts/tasks/task-1/assistant_summary.txt",
			},
		}
		runner := &fakeRunner{}
		dispatch := NewDispatchTaskHandler(
			newFakeTransactor(tasks, approvals, runs, &fakeArtifactRepo{}),
			tasks,
			&fakeLeaseRepo{},
			fakePolicy{
				decision: domainpolicy.Decision{
					RequiresApproval: true,
					Reason:           "git push origin main requires approval",
				},
			},
			runner,
			approvals,
			runs,
			&fakeArtifactRepo{},
		)
		handler := NewApproveApprovalHandler(newFakeTransactor(tasks, approvals, runs, &fakeArtifactRepo{}), approvals, tasks, dispatch)

		out, err := handler.Handle(ApproveApprovalCommand{ApprovalID: "approval-1"})
		require.NoError(t, err)
		require.Equal(t, string(approval.StatusApproved), out.ApprovalStatus)
		require.Equal(t, "completed", out.TaskState)
		require.Equal(t, "completed", out.RunState)
		require.Equal(t, 0, approvals.saveCount)
		require.Equal(t, 0, runner.dispatchCount)
	})

	t.Run("reject repeated rejection is no-op success", func(t *testing.T) {
		tasks := newFakeTaskRepo()
		repoTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "git push origin main", "repo:project-1")
		require.NoError(t, tasks.Save(repoTask))

		approvals := &fakeApprovalRepo{
			byTaskID: map[string]approval.Approval{
				"task-1": {
					ID:              "approval-1",
					TaskID:          "task-1",
					Reason:          "git push origin main requires approval",
					Status:          approval.StatusRejected,
					RejectionReason: "already rejected",
				},
			},
		}
		handler := NewRejectApprovalHandler(newFakeTransactor(tasks, approvals, &fakeRunRepo{}, &fakeArtifactRepo{}), approvals, tasks)

		out, err := handler.Handle(RejectApprovalCommand{
			ApprovalID: "approval-1",
			Reason:     "ignored on repeat",
		})
		require.NoError(t, err)
		require.Equal(t, string(approval.StatusRejected), out.ApprovalStatus)
		require.Equal(t, "ready", out.TaskState)
		require.Equal(t, 0, approvals.saveCount)
	})

	t.Run("approve rejected approval conflicts", func(t *testing.T) {
		tasks := newFakeTaskRepo()
		require.NoError(t, tasks.Save(task.NewTask("task-1", "module-1", task.TaskTypeWrite, "git push origin main", "repo:project-1")))

		approvals := &fakeApprovalRepo{
			byTaskID: map[string]approval.Approval{
				"task-1": {
					ID:     "approval-1",
					TaskID: "task-1",
					Reason: "git push origin main requires approval",
					Status: approval.StatusRejected,
				},
			},
		}
		handler := NewApproveApprovalHandler(newFakeTransactor(tasks, approvals, &fakeRunRepo{}, &fakeArtifactRepo{}), approvals, tasks, newApprovalTestDispatchHandler(tasks, approvals, &fakeRunner{}))

		_, err := handler.Handle(ApproveApprovalCommand{ApprovalID: "approval-1"})
		require.ErrorIs(t, err, ErrApprovalActionConflict)
	})

	t.Run("reject approved approval conflicts", func(t *testing.T) {
		tasks := newFakeTaskRepo()
		require.NoError(t, tasks.Save(task.NewTask("task-1", "module-1", task.TaskTypeWrite, "git push origin main", "repo:project-1")))

		approvals := &fakeApprovalRepo{
			byTaskID: map[string]approval.Approval{
				"task-1": {
					ID:     "approval-1",
					TaskID: "task-1",
					Reason: "git push origin main requires approval",
					Status: approval.StatusApproved,
				},
			},
		}
		handler := NewRejectApprovalHandler(newFakeTransactor(tasks, approvals, &fakeRunRepo{}, &fakeArtifactRepo{}), approvals, tasks)

		_, err := handler.Handle(RejectApprovalCommand{
			ApprovalID: "approval-1",
			Reason:     "should fail",
		})
		require.ErrorIs(t, err, ErrApprovalActionConflict)
	})

	t.Run("retry dispatch on ineligible approval conflicts", func(t *testing.T) {
		tasks := newFakeTaskRepo()
		repoTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "git push origin main", "repo:project-1")
		repoTask.State = task.TaskStateWaitingApproval
		require.NoError(t, tasks.Save(repoTask))

		approvals := &fakeApprovalRepo{
			byTaskID: map[string]approval.Approval{
				"task-1": approval.New("approval-1", "task-1", "git push origin main requires approval"),
			},
		}
		handler := NewRetryApprovalDispatchHandler(approvals, tasks, newApprovalTestDispatchHandler(tasks, approvals, &fakeRunner{}))

		_, err := handler.Handle(RetryApprovalDispatchCommand{ApprovalID: "approval-1"})
		require.ErrorIs(t, err, ErrApprovalActionConflict)
	})
}

func newApprovalTestDispatchHandler(tasks *fakeTaskRepo, approvals *fakeApprovalRepo, runner *fakeRunner) *DispatchTaskHandler {
	runs := &fakeRunRepo{}
	artifacts := &fakeArtifactRepo{}
	tx := newFakeTransactor(tasks, approvals, runs, artifacts)
	return NewDispatchTaskHandler(
		tx,
		tasks,
		&fakeLeaseRepo{},
		fakePolicy{
			decision: domainpolicy.Decision{
				RequiresApproval: true,
				Reason:           "git push origin main requires approval",
			},
		},
		runner,
		approvals,
		runs,
		artifacts,
	)
}
