package manageragent

type Request struct {
	Kind      string
	SessionID string
	TaskID    string
	Summary   string
}

type Response struct {
	Kind    string
	TaskID  string
	Summary string
}

type TaskStatusView struct {
	TaskID          string
	ModuleID        string
	Summary         string
	State           string
	Priority        int
	PendingApproval bool
}
