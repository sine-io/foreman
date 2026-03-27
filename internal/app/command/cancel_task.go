package command

import (
	"github.com/sine-io/foreman/internal/domain/task"
	"github.com/sine-io/foreman/internal/ports"
)

type CancelTaskCommand struct {
	TaskID string
}

type CancelTaskHandler struct {
	Tasks ports.TaskRepository
}

func NewCancelTaskHandler(tasks ports.TaskRepository) *CancelTaskHandler {
	return &CancelTaskHandler{Tasks: tasks}
}

func (h *CancelTaskHandler) Handle(cmd CancelTaskCommand) error {
	record, err := h.Tasks.Get(cmd.TaskID)
	if err != nil {
		return err
	}

	record.State = task.TaskStateCanceled
	return h.Tasks.Save(record)
}
