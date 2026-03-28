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
	TaskID          string `json:"task_id"`
	ProjectID       string `json:"project_id"`
	ModuleID        string `json:"module_id"`
	Summary         string `json:"summary"`
	State           string `json:"state"`
	Priority        int    `json:"priority"`
	RunID           string `json:"run_id,omitempty"`
	RunState        string `json:"run_state,omitempty"`
	ApprovalID      string `json:"approval_id,omitempty"`
	ApprovalReason  string `json:"approval_reason,omitempty"`
	ApprovalState   string `json:"approval_state,omitempty"`
	PendingApproval bool   `json:"pending_approval"`
}

type managerRejectApprovalRequest struct {
	RejectionReason string `json:"rejection_reason"`
}

type managerApprovalQueueResponse struct {
	Items []managerApprovalWorkbenchItemResponse `json:"items"`
}

type managerApprovalWorkbenchItemResponse struct {
	ApprovalID string `json:"approval_id"`
	TaskID     string `json:"task_id"`
	Summary    string `json:"summary"`
	RiskLevel  string `json:"risk_level"`
	Priority   int    `json:"priority"`
}

type managerApprovalDetailResponse struct {
	ApprovalID              string                                     `json:"approval_id"`
	TaskID                  string                                     `json:"task_id"`
	Summary                 string                                     `json:"summary"`
	Reason                  string                                     `json:"reason"`
	ApprovalState           string                                     `json:"approval_state"`
	RiskLevel               string                                     `json:"risk_level"`
	PolicyRule              string                                     `json:"policy_rule"`
	RejectionReason         string                                     `json:"rejection_reason,omitempty"`
	Priority                int                                        `json:"priority"`
	CreatedAt               string                                     `json:"created_at"`
	TaskState               string                                     `json:"task_state"`
	RunID                   string                                     `json:"run_id,omitempty"`
	RunState                string                                     `json:"run_state,omitempty"`
	RunDetailURL            string                                     `json:"run_detail_url,omitempty"`
	AssistantSummaryPreview string                                     `json:"assistant_summary_preview"`
	Artifacts               []managerApprovalWorkbenchArtifactResponse `json:"artifacts"`
}

type managerApprovalWorkbenchArtifactResponse struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Path    string `json:"path"`
	Summary string `json:"summary"`
}

type managerApprovalWorkbenchActionResponse struct {
	ApprovalID      string `json:"approval_id"`
	ApprovalState   string `json:"approval_state"`
	RejectionReason string `json:"rejection_reason,omitempty"`
	TaskID          string `json:"task_id"`
	TaskState       string `json:"task_state"`
	RunID           string `json:"run_id,omitempty"`
	RunState        string `json:"run_state,omitempty"`
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
