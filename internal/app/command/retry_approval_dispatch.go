package command

import (
	"github.com/sine-io/foreman/internal/domain/approval"
	"github.com/sine-io/foreman/internal/domain/task"
	"github.com/sine-io/foreman/internal/ports"
)

type RetryApprovalDispatchCommand struct {
	ApprovalID string
}

type RetryApprovalDispatchHandler struct {
	Approvals ports.ApprovalRepository
	Tasks     ports.TaskRepository
	Dispatch  *DispatchTaskHandler
}

func NewRetryApprovalDispatchHandler(
	approvals ports.ApprovalRepository,
	tasks ports.TaskRepository,
	dispatch *DispatchTaskHandler,
) *RetryApprovalDispatchHandler {
	return &RetryApprovalDispatchHandler{
		Approvals: approvals,
		Tasks:     tasks,
		Dispatch:  dispatch,
	}
}

func (h *RetryApprovalDispatchHandler) Handle(cmd RetryApprovalDispatchCommand) (ApprovalActionResult, error) {
	record, repoTask, err := loadApprovalTaskPair(h.Approvals, h.Tasks, cmd.ApprovalID)
	if err != nil {
		return ApprovalActionResult{}, err
	}

	if record.Status != approval.StatusApproved || repoTask.State != task.TaskStateApprovedPendingDispatch {
		return ApprovalActionResult{}, approvalConflict(
			"approval %s cannot retry dispatch from status %s and task state %s",
			record.ID,
			record.Status,
			repoTask.State,
		)
	}

	out, err := h.Dispatch.Handle(DispatchTaskCommand{TaskID: record.TaskID})
	if err != nil {
		return ApprovalActionResult{
			ApprovalID:     record.ID,
			TaskID:         record.TaskID,
			ApprovalStatus: string(record.Status),
			TaskState:      string(task.TaskStateApprovedPendingDispatch),
		}, err
	}

	return ApprovalActionResult{
		ApprovalID:     record.ID,
		TaskID:         record.TaskID,
		ApprovalStatus: string(record.Status),
		TaskState:      out.TaskState,
		RunState:       out.RunState,
	}, nil
}
