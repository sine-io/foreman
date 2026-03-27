package task

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
