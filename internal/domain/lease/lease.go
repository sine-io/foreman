package lease

type Status string

const (
	StatusActive   Status = "active"
	StatusReleased Status = "released"
)

type Lease struct {
	ID       string
	TaskID   string
	ScopeKey string
	Status   Status
}

func New(id, taskID, scopeKey string) Lease {
	return Lease{
		ID:       id,
		TaskID:   taskID,
		ScopeKey: scopeKey,
		Status:   StatusActive,
	}
}
