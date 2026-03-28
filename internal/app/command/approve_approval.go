package command

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/sine-io/foreman/internal/domain/approval"
	"github.com/sine-io/foreman/internal/domain/task"
	"github.com/sine-io/foreman/internal/ports"
)

var ErrApprovalActionConflict = errors.New("approval action conflict")

type ApprovalActionResult struct {
	ApprovalID     string
	TaskID         string
	ApprovalStatus string
	TaskState      string
	RunState       string
}

type ApproveApprovalCommand struct {
	ApprovalID string
}

type ApproveApprovalHandler struct {
	Tx        ports.Transactor
	Approvals ports.ApprovalRepository
	Tasks     ports.TaskRepository
	Dispatch  *DispatchTaskHandler
}

func NewApproveApprovalHandler(
	tx ports.Transactor,
	approvals ports.ApprovalRepository,
	tasks ports.TaskRepository,
	dispatch *DispatchTaskHandler,
) *ApproveApprovalHandler {
	return &ApproveApprovalHandler{
		Tx:        tx,
		Approvals: approvals,
		Tasks:     tasks,
		Dispatch:  dispatch,
	}
}

func (h *ApproveApprovalHandler) Handle(cmd ApproveApprovalCommand) (ApprovalActionResult, error) {
	record, repoTask, err := h.loadApprovalAndTask(cmd.ApprovalID)
	if err != nil {
		return ApprovalActionResult{}, err
	}

	switch record.Status {
	case approval.StatusApproved:
		return h.currentResult(record, repoTask)
	case approval.StatusRejected:
		return ApprovalActionResult{}, approvalConflict("approval %s is already rejected", record.ID)
	case approval.StatusPending:
		if repoTask.State != task.TaskStateWaitingApproval {
			return ApprovalActionResult{}, approvalConflict("approval %s cannot be approved from task state %s", record.ID, repoTask.State)
		}
	default:
		return ApprovalActionResult{}, approvalConflict("approval %s cannot be approved from status %s", record.ID, record.Status)
	}

	if err := h.Tx.WithinTransaction(context.Background(), func(_ context.Context, repos ports.TransactionRepositories) error {
		record.Status = approval.StatusApproved
		record.RejectionReason = ""
		return repos.Approvals.Save(record)
	}); err != nil {
		return ApprovalActionResult{}, err
	}

	out, err := h.Dispatch.Handle(DispatchTaskCommand{TaskID: record.TaskID})
	if err != nil {
		markErr := h.markApprovedPendingDispatch(record.TaskID)
		if markErr != nil {
			return ApprovalActionResult{}, errors.Join(err, markErr)
		}
		return ApprovalActionResult{
			ApprovalID:     record.ID,
			TaskID:         record.TaskID,
			ApprovalStatus: string(approval.StatusApproved),
			TaskState:      string(task.TaskStateApprovedPendingDispatch),
		}, err
	}

	return ApprovalActionResult{
		ApprovalID:     record.ID,
		TaskID:         record.TaskID,
		ApprovalStatus: string(approval.StatusApproved),
		TaskState:      out.TaskState,
		RunState:       out.RunState,
	}, nil
}

func (h *ApproveApprovalHandler) loadApprovalAndTask(approvalID string) (approval.Approval, task.Task, error) {
	record, err := h.Approvals.Get(approvalID)
	if err != nil {
		return approval.Approval{}, task.Task{}, err
	}

	repoTask, err := h.Tasks.Get(record.TaskID)
	if err != nil {
		return approval.Approval{}, task.Task{}, err
	}

	return record, repoTask, nil
}

func (h *ApproveApprovalHandler) currentResult(record approval.Approval, repoTask task.Task) (ApprovalActionResult, error) {
	result := ApprovalActionResult{
		ApprovalID:     record.ID,
		TaskID:         record.TaskID,
		ApprovalStatus: string(record.Status),
		TaskState:      string(repoTask.State),
	}

	if h.Dispatch == nil {
		return result, nil
	}

	runState, err := h.Dispatch.currentAuthoritativeRunState(record.TaskID)
	if err != nil {
		return ApprovalActionResult{}, err
	}
	result.RunState = runState
	if runState == "running" {
		result.TaskState = string(task.TaskStateRunning)
	}
	if runState == "completed" {
		result.TaskState = string(task.TaskStateCompleted)
	}

	return result, nil
}

func (h *ApproveApprovalHandler) markApprovedPendingDispatch(taskID string) error {
	return h.Tx.WithinTransaction(context.Background(), func(_ context.Context, repos ports.TransactionRepositories) error {
		repoTask, err := repos.Tasks.Get(taskID)
		if err != nil {
			return err
		}

		if repoTask.State != task.TaskStateWaitingApproval && repoTask.State != task.TaskStateApprovedPendingDispatch {
			return nil
		}

		repoTask.State = task.TaskStateApprovedPendingDispatch
		return repos.Tasks.Save(repoTask)
	})
}

func approvalConflict(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrApprovalActionConflict, fmt.Sprintf(format, args...))
}

func currentRunState(runs ports.RunRepository, taskID string) (string, error) {
	if runs == nil {
		return "", nil
	}

	run, err := runs.FindByTask(taskID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	if !isAuthoritativeRun(run) {
		return "", nil
	}

	return run.State, nil
}
