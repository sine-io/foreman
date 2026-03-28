package query

import (
	"database/sql"
	"fmt"

	"github.com/sine-io/foreman/internal/domain/approval"
	modulepkg "github.com/sine-io/foreman/internal/domain/module"
	"github.com/sine-io/foreman/internal/domain/task"
	"github.com/sine-io/foreman/internal/ports"
)

type TaskStatusView struct {
	TaskID         string
	ProjectID      string
	ModuleID       string
	Summary        string
	State          string
	RunID          string
	RunState       string
	ApprovalID     string
	ApprovalReason string
	ApprovalState  string
}

type TaskStatusRepository interface {
	Task(id string) (task.Task, error)
	Module(id string) (modulepkg.Module, error)
	FindByTask(taskID string) (ports.Run, error)
	FindPendingByTask(taskID string) (approval.Approval, error)
	FindLatestByTask(taskID string) (approval.Approval, error)
}

type taskStatusRepository struct {
	Tasks     ports.TaskRepository
	Modules   ports.ModuleRepository
	Runs      ports.RunRepository
	Approvals ports.ApprovalRepository
}

func (r taskStatusRepository) Task(id string) (task.Task, error) {
	return r.Tasks.Get(id)
}

func (r taskStatusRepository) Module(id string) (modulepkg.Module, error) {
	return r.Modules.Get(id)
}

func (r taskStatusRepository) FindByTask(taskID string) (ports.Run, error) {
	return r.Runs.FindByTask(taskID)
}

func (r taskStatusRepository) FindPendingByTask(taskID string) (approval.Approval, error) {
	return r.Approvals.FindPendingByTask(taskID)
}

func (r taskStatusRepository) FindLatestByTask(taskID string) (approval.Approval, error) {
	return r.Approvals.FindLatestByTask(taskID)
}

type TaskStatusQuery struct {
	Repo TaskStatusRepository
}

func NewTaskStatusQuery(repo TaskStatusRepository) *TaskStatusQuery {
	return &TaskStatusQuery{Repo: repo}
}

func NewTaskStatusQueryFromRepositories(tasks ports.TaskRepository, modules ports.ModuleRepository, runs ports.RunRepository, approvals ports.ApprovalRepository) *TaskStatusQuery {
	return NewTaskStatusQuery(taskStatusRepository{
		Tasks:     tasks,
		Modules:   modules,
		Runs:      runs,
		Approvals: approvals,
	})
}

func (q *TaskStatusQuery) Execute(projectID, taskID string) (TaskStatusView, error) {
	taskRecord, err := q.Repo.Task(taskID)
	if err != nil {
		return TaskStatusView{}, err
	}

	moduleRecord, err := q.Repo.Module(taskRecord.ModuleID)
	if err != nil {
		return TaskStatusView{}, err
	}
	if moduleRecord.ProjectID != projectID {
		return TaskStatusView{}, fmt.Errorf("task %s does not belong to project %s", taskID, projectID)
	}

	view := TaskStatusView{
		TaskID:    taskRecord.ID,
		ProjectID: moduleRecord.ProjectID,
		ModuleID:  taskRecord.ModuleID,
		Summary:   taskRecord.Summary,
		State:     string(taskRecord.State),
	}

	runRecord, err := q.Repo.FindByTask(taskID)
	if err == nil {
		view.RunID = runRecord.ID
		view.RunState = runRecord.State
	} else if err != nil && err != sql.ErrNoRows {
		return TaskStatusView{}, err
	}

	approvalRecord, err := q.Repo.FindPendingByTask(taskID)
	if err == sql.ErrNoRows {
		approvalRecord, err = q.Repo.FindLatestByTask(taskID)
	}
	if err == nil {
		view.ApprovalID = approvalRecord.ID
		view.ApprovalReason = approvalRecord.Reason
		view.ApprovalState = string(approvalRecord.Status)
	} else if err != nil && err != sql.ErrNoRows {
		return TaskStatusView{}, err
	}

	return view, nil
}
