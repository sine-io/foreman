package command

import (
	"context"
	"database/sql"
	"errors"

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
	Tx        ports.Transactor
	Approvals ports.ApprovalRepository
	Tasks     ports.TaskRepository
	Delegate  *ApproveApprovalHandler
}

func NewApproveTaskHandler(
	tx ports.Transactor,
	approvals ports.ApprovalRepository,
	tasks ports.TaskRepository,
	dispatch ...*DispatchTaskHandler,
) *ApproveTaskHandler {
	handler := &ApproveTaskHandler{
		Tx:        tx,
		Approvals: approvals,
		Tasks:     tasks,
	}
	if len(dispatch) > 0 && dispatch[0] != nil {
		handler.Delegate = NewApproveApprovalHandler(tx, approvals, tasks, dispatch[0])
	}
	return handler
}

func (h *ApproveTaskHandler) Handle(cmd ApproveTaskCommand) error {
	if h.Delegate != nil {
		record, err := h.Approvals.FindPendingByTask(cmd.TaskID)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			latest, latestErr := h.Approvals.FindLatestByTask(cmd.TaskID)
			if latestErr != nil {
				return latestErr
			}
			if latest.Status == approval.StatusApproved {
				_, actionErr := h.Delegate.Handle(ApproveApprovalCommand{ApprovalID: latest.ID})
				return actionErr
			}
			return err
		}

		_, err = h.Delegate.Handle(ApproveApprovalCommand{ApprovalID: record.ID})
		return err
	}

	return h.Tx.WithinTransaction(context.Background(), func(_ context.Context, repos ports.TransactionRepositories) error {
		record, err := repos.Approvals.FindPendingByTask(cmd.TaskID)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			latest, latestErr := repos.Approvals.FindLatestByTask(cmd.TaskID)
			if latestErr != nil {
				return latestErr
			}
			repoTask, taskErr := repos.Tasks.Get(cmd.TaskID)
			if taskErr != nil {
				return taskErr
			}
			if latest.Status == approval.StatusApproved && repoTask.State != task.TaskStateWaitingApproval {
				return nil
			}
			return err
		}

		record.Status = approval.StatusApproved
		if err := repos.Approvals.Save(record); err != nil {
			return err
		}

		repoTask, err := repos.Tasks.Get(cmd.TaskID)
		if err != nil {
			return err
		}

		repoTask.State = task.TaskStateLeased
		return repos.Tasks.Save(repoTask)
	})
}
