package http

type reprioritizeRequest struct {
	Priority int `json:"priority"`
}

type taskActionResponse struct {
	State string `json:"state"`
}

type managerCommandRequest struct {
	Kind        string `json:"kind"`
	SessionID   string `json:"session_id,omitempty"`
	ProjectID   string `json:"project_id,omitempty"`
	ModuleID    string `json:"module_id,omitempty"`
	TaskID      string `json:"task_id,omitempty"`
	Name        string `json:"name,omitempty"`
	Summary     string `json:"summary,omitempty"`
	Description string `json:"description,omitempty"`
	RepoRoot    string `json:"repo_root,omitempty"`
	TaskType    string `json:"task_type,omitempty"`
	WriteScope  string `json:"write_scope,omitempty"`
	Acceptance  string `json:"acceptance,omitempty"`
	Priority    int    `json:"priority,omitempty"`
}

type managerCommandResponse struct {
	Kind      string `json:"kind"`
	ProjectID string `json:"project_id,omitempty"`
	ModuleID  string `json:"module_id,omitempty"`
	TaskID    string `json:"task_id,omitempty"`
	Summary   string `json:"summary,omitempty"`
}

type managerTaskStatusResponse struct {
	TaskID         string `json:"task_id"`
	ProjectID      string `json:"project_id"`
	ModuleID       string `json:"module_id"`
	Summary        string `json:"summary"`
	State          string `json:"state"`
	RunID          string `json:"run_id,omitempty"`
	RunState       string `json:"run_state,omitempty"`
	ApprovalID     string `json:"approval_id,omitempty"`
	ApprovalReason string `json:"approval_reason,omitempty"`
	ApprovalState  string `json:"approval_state,omitempty"`
}

type managerBoardSnapshotResponse struct {
	ProjectID string                                     `json:"project_id"`
	Modules   map[string][]managerModuleSnapshotResponse `json:"modules"`
	Tasks     map[string][]managerTaskSnapshotResponse   `json:"tasks"`
}

type managerModuleSnapshotResponse struct {
	ModuleID string `json:"module_id"`
	Name     string `json:"name"`
	State    string `json:"state"`
}

type managerTaskSnapshotResponse struct {
	TaskID          string `json:"task_id"`
	ModuleID        string `json:"module_id"`
	Summary         string `json:"summary"`
	State           string `json:"state"`
	Priority        int    `json:"priority"`
	PendingApproval bool   `json:"pending_approval"`
}
