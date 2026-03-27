package task

import (
	"fmt"
	"strings"
)

type TaskType string

const (
	TaskTypeRead  TaskType = "read"
	TaskTypeWrite TaskType = "write"
)

type TaskState string

const (
	TaskStateReady           TaskState = "ready"
	TaskStateLeased          TaskState = "leased"
	TaskStateRunning         TaskState = "running"
	TaskStateWaitingApproval TaskState = "waiting_approval"
	TaskStateCompleted       TaskState = "completed"
	TaskStateFailed          TaskState = "failed"
	TaskStateCanceled        TaskState = "canceled"
)

type Task struct {
	ID         string
	ModuleID   string
	Type       TaskType
	Summary    string
	Acceptance string
	Priority   int
	WriteScope string
	State      TaskState
}

func NewTask(id, moduleID string, taskType TaskType, summary, writeScope string) Task {
	return Task{
		ID:         id,
		ModuleID:   moduleID,
		Type:       taskType,
		Summary:    summary,
		WriteScope: writeScope,
		State:      TaskStateReady,
	}
}

func (t Task) CanTransition(next TaskState) bool {
	switch t.State {
	case TaskStateReady:
		return next == TaskStateLeased || next == TaskStateCanceled
	case TaskStateLeased:
		return next == TaskStateRunning || next == TaskStateWaitingApproval || next == TaskStateCanceled
	case TaskStateRunning:
		return next == TaskStateWaitingApproval || next == TaskStateCompleted || next == TaskStateFailed || next == TaskStateCanceled
	case TaskStateWaitingApproval:
		return next == TaskStateLeased || next == TaskStateCanceled
	default:
		return false
	}
}

func ParseTaskType(raw string) (TaskType, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(TaskTypeRead):
		return TaskTypeRead, nil
	case string(TaskTypeWrite):
		return TaskTypeWrite, nil
	default:
		return "", fmt.Errorf("unsupported task type: %s", raw)
	}
}
