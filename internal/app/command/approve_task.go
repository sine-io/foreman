package command

import (
	"github.com/sine-io/foreman/internal/domain/approval"
	"github.com/sine-io/foreman/internal/domain/task"
	"github.com/sine-io/foreman/internal/ports"
)

type ApproveTaskCommand struct {
	TaskID string
}

type CreateApprovalHandler struct {
	Approvals ports.ApprovalRepository
	Tasks     ports.TaskRepository
}

type ApproveTaskHandler struct {
	Approvals ports.ApprovalRepository
	Tasks     ports.TaskRepository
}

func NewApproveTaskHandler(approvals ports.ApprovalRepository, tasks ports.TaskRepository) *ApproveTaskHandler {
	return &ApproveTaskHandler{
		Approvals: approvals,
		Tasks:     tasks,
	}
}

func (h *ApproveTaskHandler) Handle(cmd ApproveTaskCommand) error {
	record, err := h.Approvals.FindPendingByTask(cmd.TaskID)
	if err != nil {
		return err
	}

	record.Status = approval.StatusApproved
	if err := h.Approvals.Save(record); err != nil {
		return err
	}

	repoTask, err := h.Tasks.Get(cmd.TaskID)
	if err != nil {
		return err
	}

	repoTask.State = task.TaskStateLeased
	return h.Tasks.Save(repoTask)
}
