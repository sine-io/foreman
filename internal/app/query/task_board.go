package query

import "github.com/sine-io/foreman/internal/ports"

type TaskCard struct {
	ID              string
	ModuleID        string
	Summary         string
	State           string
	Priority        int
	PendingApproval bool
}

type TaskBoardView struct {
	Columns map[string][]TaskCard
}

type TaskBoardQuery struct {
	Repo ports.BoardQueryRepository
}

func NewTaskBoardQuery(repo ports.BoardQueryRepository) *TaskBoardQuery {
	return &TaskBoardQuery{Repo: repo}
}

func (q *TaskBoardQuery) Execute(projectID string) (TaskBoardView, error) {
	rows, err := q.Repo.ListTasks(projectID)
	if err != nil {
		return TaskBoardView{}, err
	}

	view := TaskBoardView{
		Columns: map[string][]TaskCard{},
	}

	for _, row := range rows {
		column := taskColumnName(row.State, row.PendingApproval)
		view.Columns[column] = append(view.Columns[column], TaskCard{
			ID:              row.TaskID,
			ModuleID:        row.ModuleID,
			Summary:         row.Summary,
			State:           row.State,
			Priority:        row.Priority,
			PendingApproval: row.PendingApproval,
		})
	}

	return view, nil
}

func taskColumnName(state string, pendingApproval bool) string {
	if pendingApproval || state == "waiting_approval" {
		return "Waiting Approval"
	}

	switch state {
	case "ready":
		return "Ready"
	case "leased", "running":
		return "In Progress"
	case "failed":
		return "Failed"
	case "completed":
		return "Done"
	case "canceled":
		return "Canceled"
	default:
		return state
	}
}
