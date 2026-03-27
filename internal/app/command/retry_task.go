package command

import (
	"github.com/sine-io/foreman/internal/domain/task"
	"github.com/sine-io/foreman/internal/ports"
)

type RetryTaskCommand struct {
	TaskID string
}

type RetryTaskHandler struct {
	Tasks ports.TaskRepository
}

func NewRetryTaskHandler(tasks ports.TaskRepository) *RetryTaskHandler {
	return &RetryTaskHandler{Tasks: tasks}
}

func (h *RetryTaskHandler) Handle(cmd RetryTaskCommand) error {
	record, err := h.Tasks.Get(cmd.TaskID)
	if err != nil {
		return err
	}

	record.State = task.TaskStateReady
	return h.Tasks.Save(record)
}
