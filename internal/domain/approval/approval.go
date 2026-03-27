package approval

type Status string

const (
	StatusPending  Status = "pending"
	StatusApproved Status = "approved"
	StatusRejected Status = "rejected"
)

type Approval struct {
	ID     string
	TaskID string
	Reason string
	Status Status
}

func New(id, taskID, reason string) Approval {
	return Approval{
		ID:     id,
		TaskID: taskID,
		Reason: reason,
		Status: StatusPending,
	}
}
