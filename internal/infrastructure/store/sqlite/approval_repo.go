package sqlite

import (
	"database/sql"

	"github.com/sine-io/foreman/internal/domain/approval"
)

type ApprovalRepository struct {
	db *sql.DB
}

func NewApprovalRepository(db *sql.DB) *ApprovalRepository {
	return &ApprovalRepository{db: db}
}

func (r *ApprovalRepository) Save(a approval.Approval) error {
	_, err := r.db.Exec(
		`insert into approvals (id, task_id, reason, state) values (?, ?, ?, ?)
		 on conflict(id) do update set task_id = excluded.task_id, reason = excluded.reason, state = excluded.state`,
		a.ID,
		a.TaskID,
		a.Reason,
		a.Status,
	)
	return err
}

func (r *ApprovalRepository) Get(id string) (approval.Approval, error) {
	var a approval.Approval
	err := r.db.QueryRow(
		`select id, task_id, reason, state from approvals where id = ?`,
		id,
	).Scan(&a.ID, &a.TaskID, &a.Reason, &a.Status)
	return a, err
}
