package ports

import (
	"errors"

	"github.com/sine-io/foreman/internal/domain/approval"
	modulepkg "github.com/sine-io/foreman/internal/domain/module"
	"github.com/sine-io/foreman/internal/domain/project"
	"github.com/sine-io/foreman/internal/domain/task"
)

var ErrPendingApprovalConflict = errors.New("pending approval already exists")

var ErrArtifactRunLinkageConflict = errors.New("artifact run linkage conflict")
var ErrArtifactBrokenLinkage = errors.New("artifact broken linkage")
var ErrArtifactCompareSelectionInvalid = errors.New("artifact compare selection invalid")

type ProjectRepository interface {
	Save(project.Project) error
	Get(id string) (project.Project, error)
}

type ModuleRepository interface {
	Save(modulepkg.Module) error
	Get(id string) (modulepkg.Module, error)
}

type TaskRepository interface {
	Save(task.Task) error
	Get(id string) (task.Task, error)
}

type Run struct {
	ID                   string
	TaskID               string
	RunnerKind           string
	State                string
	AssistantSummaryPath string
	CreatedAt            string
}

type RunRepository interface {
	Save(Run) error
	Get(id string) (Run, error)
	FindByTask(taskID string) (Run, error)
}

type ArtifactRecord struct {
	ID          string
	TaskID      string
	RunID       string
	Kind        string
	Path        string
	StoragePath string
	Summary     string
}

type ArtifactRepository interface {
	Create(taskID, runID, kind, path string) (string, error)
	Get(id string) (ArtifactRecord, error)
}

type ApprovalRepository interface {
	Save(approval.Approval) error
	Get(id string) (approval.Approval, error)
	FindPendingByTask(taskID string) (approval.Approval, error)
	FindLatestByTask(taskID string) (approval.Approval, error)
}

type LeaseRepository interface {
	Acquire(taskID, scopeKey string) error
	Release(taskID, scopeKey string) error
}

type ModuleBoardRow struct {
	ModuleID   string
	Name       string
	BoardState string
}

type TaskBoardRow struct {
	TaskID          string
	ModuleID        string
	Summary         string
	State           string
	Priority        int
	PendingApproval bool
}

type RunDetailRecord struct {
	Run         Run
	TaskSummary string
	Artifacts   []ArtifactRecord
}

type RunWorkbenchRow struct {
	RunID        string
	TaskID       string
	ProjectID    string
	ModuleID     string
	TaskSummary  string
	RunState     string
	RunnerKind   string
	RunCreatedAt string
	Artifacts    []ArtifactRecord
}

type ArtifactWorkbenchRow struct {
	ArtifactID  string
	RunID       string
	TaskID      string
	ProjectID   string
	ModuleID    string
	Kind        string
	Path        string
	StoragePath string
	Summary     string
	Siblings    []ArtifactRecord
}

type ArtifactCompareArtifactRow struct {
	ArtifactID  string
	RunID       string
	TaskID      string
	Kind        string
	Path        string
	StoragePath string
	Summary     string
	CreatedAt   string
}

type ArtifactCompareHistoryItemRow struct {
	ArtifactID string
	RunID      string
	CreatedAt  string
	Summary    string
}

type ArtifactCompareRow struct {
	Current  ArtifactCompareArtifactRow
	Previous *ArtifactCompareArtifactRow
	History  []ArtifactCompareHistoryItemRow
}

type ApprovalQueueRow struct {
	ApprovalID      string
	TaskID          string
	ModuleID        string
	Summary         string
	Reason          string
	State           string
	RiskLevel       string
	PolicyRule      string
	RejectionReason string
}

type ApprovalWorkbenchQueueRow struct {
	ApprovalID string
	TaskID     string
	Summary    string
	RiskLevel  string
	Priority   int
	CreatedAt  string
}

type ApprovalWorkbenchDetailRow struct {
	ApprovalID       string
	TaskID           string
	Summary          string
	Reason           string
	ApprovalState    string
	RiskLevel        string
	PolicyRule       string
	RejectionReason  string
	Priority         int
	CreatedAt        string
	TaskState        string
	RunID            string
	RunState         string
	AssistantSummary string
	Artifacts        []ArtifactRecord
}

type TaskWorkbenchRow struct {
	TaskID               string
	ProjectID            string
	ModuleID             string
	Summary              string
	TaskState            string
	Priority             int
	WriteScope           string
	TaskType             string
	Acceptance           string
	LatestRunID          string
	LatestRunState       string
	LatestRunSummary     string
	LatestApprovalID     string
	LatestApprovalState  string
	LatestApprovalReason string
	Artifacts            []ArtifactRecord
}

type BoardQueryRepository interface {
	ListModules(projectID string) ([]ModuleBoardRow, error)
	ListTasks(projectID string) ([]TaskBoardRow, error)
	GetRunDetail(runID string) (RunDetailRecord, error)
	GetRunWorkbench(runID string) (RunWorkbenchRow, error)
	GetArtifactWorkbench(artifactID string) (ArtifactWorkbenchRow, error)
	GetArtifactCompare(artifactID string, previousArtifactID string) (ArtifactCompareRow, error)
	ListApprovals(projectID string) ([]ApprovalQueueRow, error)
	ListApprovalWorkbenchQueue(projectID string) ([]ApprovalWorkbenchQueueRow, error)
	GetApprovalWorkbenchDetail(approvalID string) (ApprovalWorkbenchDetailRow, error)
	GetTaskWorkbench(taskID string) (TaskWorkbenchRow, error)
}
