package manageragent

import "github.com/sine-io/foreman/internal/app/query"

type Request struct {
	Kind        string
	SessionID   string
	ProjectID   string
	ModuleID    string
	TaskID      string
	Name        string
	Summary     string
	Description string
	RepoRoot    string
	TaskType    string
	WriteScope  string
	Acceptance  string
	Priority    int
}

type Response struct {
	Kind      string
	ProjectID string
	ModuleID  string
	TaskID    string
	Summary   string
}

type TaskStatusView = query.TaskStatusView
type TaskWorkbenchView = query.TaskWorkbenchView
type TaskWorkbenchAction = query.TaskWorkbenchAction
type TaskWorkbenchArtifact = query.TaskWorkbenchArtifact
type ApprovalWorkbenchQueueView = query.ApprovalWorkbenchQueueView
type ApprovalWorkbenchItem = query.ApprovalWorkbenchItem
type ApprovalWorkbenchDetailView = query.ApprovalWorkbenchDetailView
type ApprovalWorkbenchArtifact = query.ApprovalWorkbenchArtifact

type ApprovalWorkbenchActionResponse struct {
	ApprovalID      string
	ApprovalState   string
	RejectionReason string
	TaskID          string
	TaskState       string
	RunID           string
	RunState        string
}

type ModuleSnapshot struct {
	ModuleID string
	Name     string
	State    string
}

type TaskSnapshot struct {
	TaskID          string
	ModuleID        string
	Summary         string
	State           string
	Priority        int
	PendingApproval bool
}

type BoardSnapshotView struct {
	ProjectID string
	Modules   map[string][]ModuleSnapshot
	Tasks     map[string][]TaskSnapshot
}
