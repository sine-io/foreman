package manageragent

import "errors"

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
type RunWorkbenchView = query.RunWorkbenchView
type RunWorkbenchArtifact = query.RunWorkbenchArtifact
type ArtifactWorkbenchView = query.ArtifactWorkbenchView
type ArtifactWorkbenchSibling = query.ArtifactWorkbenchSibling
type ArtifactCompareView = query.ArtifactCompareView
type ArtifactCompareArtifact = query.ArtifactCompareArtifact
type ArtifactCompareHistoryItem = query.ArtifactCompareHistoryItem
type ArtifactCompareDiff = query.ArtifactCompareDiff
type ArtifactCompareLimits = query.ArtifactCompareLimits
type ArtifactCompareMessages = query.ArtifactCompareMessages
type ArtifactCompareNavigation = query.ArtifactCompareNavigation
type TaskWorkbenchView = query.TaskWorkbenchView
type TaskWorkbenchAction = query.TaskWorkbenchAction
type TaskWorkbenchArtifact = query.TaskWorkbenchArtifact
type TaskWorkbenchActionResponse struct {
	TaskID              string
	TaskState           string
	LatestRunID         string
	LatestRunState      string
	LatestApprovalID    string
	LatestApprovalState string
	RefreshRequired     bool
	Message             string
}
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

var (
	ErrTaskActionNotFound        = errors.New("task action not found")
	ErrTaskActionConflict        = errors.New("task action conflict")
	ErrArtifactWorkbenchNotFound = errors.New("artifact workbench not found")
	ErrArtifactWorkbenchConflict = errors.New("artifact workbench conflict")
	ErrArtifactCompareNotFound   = errors.New("artifact compare not found")
	ErrArtifactCompareSelection  = errors.New("artifact compare selection invalid")
)

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
