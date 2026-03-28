package command

import (
	"database/sql"

	"github.com/sine-io/foreman/internal/domain/task"
	"github.com/sine-io/foreman/internal/ports"
)

type CreateTaskCommand struct {
	ID         string
	ModuleID   string
	Title      string
	TaskType   string
	WriteScope string
	Acceptance string
	Priority   int
}

type TaskDTO struct {
	ID       string
	ModuleID string
	State    string
	Summary  string
	Priority int
}

type CreateTaskHandler struct {
	Modules ports.ModuleRepository
	Tasks   ports.TaskRepository
}

func NewCreateTaskHandler(modules ports.ModuleRepository, tasks ports.TaskRepository) *CreateTaskHandler {
	return &CreateTaskHandler{
		Modules: modules,
		Tasks:   tasks,
	}
}

func (h *CreateTaskHandler) Handle(cmd CreateTaskCommand) (TaskDTO, error) {
	taskType, err := task.ParseTaskType(cmd.TaskType)
	if err != nil {
		return TaskDTO{}, err
	}
	if _, err := h.Modules.Get(cmd.ModuleID); err != nil {
		if err == sql.ErrNoRows {
			return TaskDTO{}, err
		}
		return TaskDTO{}, err
	}

	id := cmd.ID
	if id == "" {
		id = nextID("task")
	}

	record := task.NewTask(id, cmd.ModuleID, taskType, cmd.Title, cmd.WriteScope)
	record.Acceptance = cmd.Acceptance
	record.Priority = cmd.Priority

	if err := h.Tasks.Save(record); err != nil {
		return TaskDTO{}, err
	}

	return TaskDTO{
		ID:       record.ID,
		ModuleID: record.ModuleID,
		State:    string(record.State),
		Summary:  record.Summary,
		Priority: record.Priority,
	}, nil
}
