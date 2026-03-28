package sqlite

import (
	"database/sql"
	"time"

	"github.com/sine-io/foreman/internal/domain/approval"
)

type ApprovalRepository struct {
	db *sql.DB
}

func NewApprovalRepository(db *sql.DB) *ApprovalRepository {
	return &ApprovalRepository{db: db}
}

func (r *ApprovalRepository) Save(a approval.Approval) error {
	createdAt := a.CreatedAt
	if createdAt == "" {
		createdAt = sortableTimestamp(time.Now())
	} else {
		var err error
		createdAt, err = normalizeSortableTimestamp(createdAt)
		if err != nil {
			return err
		}
	}

	_, err := r.db.Exec(
		`insert into approvals (id, task_id, reason, state, created_at) values (?, ?, ?, ?, ?)
		 on conflict(id) do update set
		   task_id = excluded.task_id,
		   reason = excluded.reason,
		   state = excluded.state,
		   created_at = case
		     when approvals.created_at = '' then excluded.created_at
		     else approvals.created_at
		   end`,
		a.ID,
		a.TaskID,
		a.Reason,
		a.Status,
		createdAt,
	)
	return err
}

func (r *ApprovalRepository) Get(id string) (approval.Approval, error) {
	var a approval.Approval
	err := r.db.QueryRow(
		`select id, task_id, reason, state, created_at from approvals where id = ?`,
		id,
	).Scan(&a.ID, &a.TaskID, &a.Reason, &a.Status, &a.CreatedAt)
	return a, err
}

func (r *ApprovalRepository) FindPendingByTask(taskID string) (approval.Approval, error) {
	var a approval.Approval
	err := r.db.QueryRow(
		`select id, task_id, reason, state, created_at
		 from approvals
		 where task_id = ? and state = ?
		 limit 1`,
		taskID,
		approval.StatusPending,
	).Scan(&a.ID, &a.TaskID, &a.Reason, &a.Status, &a.CreatedAt)
	return a, err
}

func (r *ApprovalRepository) FindLatestByTask(taskID string) (approval.Approval, error) {
	var a approval.Approval
	err := r.db.QueryRow(
		`select id, task_id, reason, state, created_at
		 from approvals
		 where task_id = ?
		 order by created_at desc, id desc
		 limit 1`,
		taskID,
	).Scan(&a.ID, &a.TaskID, &a.Reason, &a.Status, &a.CreatedAt)
	return a, err
}
