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
}

func NewApproveTaskHandler(tx ports.Transactor, approvals ports.ApprovalRepository, tasks ports.TaskRepository) *ApproveTaskHandler {
	return &ApproveTaskHandler{
		Tx:        tx,
		Approvals: approvals,
		Tasks:     tasks,
	}
}

func (h *ApproveTaskHandler) Handle(cmd ApproveTaskCommand) error {
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
