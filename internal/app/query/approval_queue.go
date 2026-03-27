package query

import "github.com/sine-io/foreman/internal/ports"

type ApprovalCard struct {
	ApprovalID string
	TaskID     string
	ModuleID   string
	Summary    string
	Reason     string
	State      string
}

type ApprovalQueueView struct {
	Items []ApprovalCard
}

type ApprovalQueueQuery struct {
	Repo ports.BoardQueryRepository
}

func NewApprovalQueueQuery(repo ports.BoardQueryRepository) *ApprovalQueueQuery {
	return &ApprovalQueueQuery{Repo: repo}
}

func (q *ApprovalQueueQuery) Execute(projectID string) (ApprovalQueueView, error) {
	rows, err := q.Repo.ListApprovals(projectID)
	if err != nil {
		return ApprovalQueueView{}, err
	}

	view := ApprovalQueueView{}
	for _, row := range rows {
		view.Items = append(view.Items, ApprovalCard{
			ApprovalID: row.ApprovalID,
			TaskID:     row.TaskID,
			ModuleID:   row.ModuleID,
			Summary:    row.Summary,
			Reason:     row.Reason,
			State:      row.State,
		})
	}

	return view, nil
}
