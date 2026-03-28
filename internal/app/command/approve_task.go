package command

import (
	"database/sql"
	"errors"

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
	Delegate  *ApproveApprovalHandler
}

func NewApproveTaskHandler(
	tx ports.Transactor,
	approvals ports.ApprovalRepository,
	tasks ports.TaskRepository,
	dispatch ...*DispatchTaskHandler,
) *ApproveTaskHandler {
	handler := &ApproveTaskHandler{
		Approvals: approvals,
	}
	if len(dispatch) > 0 && dispatch[0] != nil {
		handler.Delegate = NewApproveApprovalHandler(tx, approvals, tasks, dispatch[0])
	}
	return handler
}

func (h *ApproveTaskHandler) Handle(cmd ApproveTaskCommand) error {
	if h.Delegate == nil {
		return errors.New("approve task handler requires approval delegate")
	}

	record, err := h.Approvals.FindPendingByTask(cmd.TaskID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		latest, latestErr := h.Approvals.FindLatestByTask(cmd.TaskID)
		if latestErr != nil {
			return latestErr
		}
		_, actionErr := h.Delegate.Handle(ApproveApprovalCommand{ApprovalID: latest.ID})
		return actionErr
	}

	_, err = h.Delegate.Handle(ApproveApprovalCommand{ApprovalID: record.ID})
	return err
}
