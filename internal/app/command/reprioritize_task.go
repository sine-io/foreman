package command

import "github.com/sine-io/foreman/internal/ports"

type ReprioritizeTaskCommand struct {
	TaskID   string
	Priority int
}

type ReprioritizeTaskHandler struct {
	Tasks ports.TaskRepository
}

func NewReprioritizeTaskHandler(tasks ports.TaskRepository) *ReprioritizeTaskHandler {
	return &ReprioritizeTaskHandler{Tasks: tasks}
}

func (h *ReprioritizeTaskHandler) Handle(cmd ReprioritizeTaskCommand) error {
	record, err := h.Tasks.Get(cmd.TaskID)
	if err != nil {
		return err
	}

	record.Priority = cmd.Priority
	return h.Tasks.Save(record)
}
