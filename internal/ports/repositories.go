package ports

import (
	"github.com/sine-io/foreman/internal/domain/approval"
	modulepkg "github.com/sine-io/foreman/internal/domain/module"
	"github.com/sine-io/foreman/internal/domain/project"
	"github.com/sine-io/foreman/internal/domain/task"
)

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
	ID         string
	TaskID     string
	RunnerKind string
	State      string
}

type RunRepository interface {
	Save(Run) error
	Get(id string) (Run, error)
}

type ArtifactRecord struct {
	ID      string
	TaskID  string
	Kind    string
	Path    string
	Summary string
}

type ArtifactRepository interface {
	Create(taskID, kind, path string) (string, error)
	Get(id string) (ArtifactRecord, error)
}

type ApprovalRepository interface {
	Save(approval.Approval) error
	Get(id string) (approval.Approval, error)
}

type LeaseRepository interface {
	Acquire(taskID, scopeKey string) error
	Release(scopeKey string) error
}

type BoardQueryRepository interface{}
