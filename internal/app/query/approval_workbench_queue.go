package query

import (
	"sort"

	"github.com/sine-io/foreman/internal/ports"
)

type ApprovalWorkbenchItem struct {
	ApprovalID string `json:"approval_id"`
	TaskID     string `json:"task_id"`
	Summary    string `json:"summary"`
	RiskLevel  string `json:"risk_level"`
	Priority   int    `json:"priority"`
}

type ApprovalWorkbenchQueueView struct {
	Items []ApprovalWorkbenchItem `json:"items"`
}

type ApprovalWorkbenchQueueQuery struct {
	Repo ports.BoardQueryRepository
}

func NewApprovalWorkbenchQueueQuery(repo ports.BoardQueryRepository) *ApprovalWorkbenchQueueQuery {
	return &ApprovalWorkbenchQueueQuery{Repo: repo}
}

func (q *ApprovalWorkbenchQueueQuery) Execute(projectID string) (ApprovalWorkbenchQueueView, error) {
	rows, err := q.Repo.ListApprovalWorkbenchQueue(projectID)
	if err != nil {
		return ApprovalWorkbenchQueueView{}, err
	}

	sort.SliceStable(rows, func(i, j int) bool {
		left := rows[i]
		right := rows[j]

		leftRisk := approvalRiskRank(left.RiskLevel)
		rightRisk := approvalRiskRank(right.RiskLevel)
		if leftRisk != rightRisk {
			return leftRisk < rightRisk
		}
		if left.Priority != right.Priority {
			return left.Priority > right.Priority
		}
		if left.CreatedAt != right.CreatedAt {
			return left.CreatedAt < right.CreatedAt
		}
		return left.ApprovalID < right.ApprovalID
	})

	view := ApprovalWorkbenchQueueView{
		Items: make([]ApprovalWorkbenchItem, 0, len(rows)),
	}
	for _, row := range rows {
		view.Items = append(view.Items, ApprovalWorkbenchItem{
			ApprovalID: row.ApprovalID,
			TaskID:     row.TaskID,
			Summary:    row.Summary,
			RiskLevel:  row.RiskLevel,
			Priority:   row.Priority,
		})
	}

	return view, nil
}

func approvalRiskRank(level string) int {
	switch level {
	case "critical":
		return 0
	case "high":
		return 1
	case "medium":
		return 2
	case "low":
		return 3
	default:
		return 4
	}
}
