package manageragent

type Command struct {
	Kind      string
	SessionID string
	TaskID    string
	Summary   string
}

type Result struct {
	Kind    string
	TaskID  string
	Summary string
}
