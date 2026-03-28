package command

import (
	"context"
	"errors"
	"strings"

	"github.com/sine-io/foreman/internal/domain/approval"
	"github.com/sine-io/foreman/internal/domain/task"
	"github.com/sine-io/foreman/internal/ports"
)

type RejectApprovalCommand struct {
	ApprovalID string
	Reason     string
}

type RejectApprovalHandler struct {
	Tx        ports.Transactor
	Approvals ports.ApprovalRepository
	Tasks     ports.TaskRepository
}

func NewRejectApprovalHandler(
	tx ports.Transactor,
	approvals ports.ApprovalRepository,
	tasks ports.TaskRepository,
) *RejectApprovalHandler {
	return &RejectApprovalHandler{
		Tx:        tx,
		Approvals: approvals,
		Tasks:     tasks,
	}
}

func (h *RejectApprovalHandler) Handle(cmd RejectApprovalCommand) (ApprovalActionResult, error) {
	if strings.TrimSpace(cmd.Reason) == "" {
		return ApprovalActionResult{}, errors.New("rejection reason is required")
	}

	record, repoTask, err := loadApprovalTaskPair(h.Approvals, h.Tasks, cmd.ApprovalID)
	if err != nil {
		return ApprovalActionResult{}, err
	}

	switch record.Status {
	case approval.StatusRejected:
		return ApprovalActionResult{
			ApprovalID:     record.ID,
			TaskID:         record.TaskID,
			ApprovalStatus: string(record.Status),
			TaskState:      string(repoTask.State),
		}, nil
	case approval.StatusApproved:
		return ApprovalActionResult{}, approvalConflict("approval %s is already approved", record.ID)
	case approval.StatusPending:
		if repoTask.State != task.TaskStateWaitingApproval {
			return ApprovalActionResult{}, approvalConflict("approval %s cannot be rejected from task state %s", record.ID, repoTask.State)
		}
	default:
		return ApprovalActionResult{}, approvalConflict("approval %s cannot be rejected from status %s", record.ID, record.Status)
	}

	if err := h.Tx.WithinTransaction(context.Background(), func(_ context.Context, repos ports.TransactionRepositories) error {
		record.Status = approval.StatusRejected
		record.RejectionReason = cmd.Reason
		if err := repos.Approvals.Save(record); err != nil {
			return err
		}

		repoTask.State = task.TaskStateReady
		return repos.Tasks.Save(repoTask)
	}); err != nil {
		return ApprovalActionResult{}, err
	}

	return ApprovalActionResult{
		ApprovalID:     record.ID,
		TaskID:         record.TaskID,
		ApprovalStatus: string(approval.StatusRejected),
		TaskState:      string(task.TaskStateReady),
	}, nil
}

func loadApprovalTaskPair(
	approvals ports.ApprovalRepository,
	tasks ports.TaskRepository,
	approvalID string,
) (approval.Approval, task.Task, error) {
	record, err := approvals.Get(approvalID)
	if err != nil {
		return approval.Approval{}, task.Task{}, approvalLookupError(err)
	}

	repoTask, err := tasks.Get(record.TaskID)
	if err != nil {
		return approval.Approval{}, task.Task{}, approvalLookupError(err)
	}

	return record, repoTask, nil
}
