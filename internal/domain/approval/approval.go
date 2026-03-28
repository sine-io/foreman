package approval

type Status string

const (
	StatusPending  Status = "pending"
	StatusApproved Status = "approved"
	StatusRejected Status = "rejected"
)

type RiskLevel string

const (
	RiskCritical RiskLevel = "critical"
	RiskHigh     RiskLevel = "high"
	RiskMedium   RiskLevel = "medium"
	RiskLow      RiskLevel = "low"
)

type Approval struct {
	ID              string
	TaskID          string
	Reason          string
	Status          Status
	RiskLevel       RiskLevel
	PolicyRule      string
	RejectionReason string
	CreatedAt       string
}

func New(id, taskID, reason string) Approval {
	return Approval{
		ID:     id,
		TaskID: taskID,
		Reason: reason,
		Status: StatusPending,
	}
}
