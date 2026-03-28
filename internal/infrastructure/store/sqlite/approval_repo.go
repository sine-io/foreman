package sqlite

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/sine-io/foreman/internal/domain/approval"
	"github.com/sine-io/foreman/internal/ports"
)

type ApprovalRepository struct {
	db dbtx
}

func NewApprovalRepository(db *sql.DB) *ApprovalRepository {
	return newApprovalRepository(db)
}

func newApprovalRepository(db dbtx) *ApprovalRepository {
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
		`insert into approvals (id, task_id, reason, state, risk_level, policy_rule, rejection_reason, created_at) values (?, ?, ?, ?, ?, ?, ?, ?)
		 on conflict(id) do update set
		   task_id = excluded.task_id,
		   reason = excluded.reason,
		   state = excluded.state,
		   risk_level = excluded.risk_level,
		   policy_rule = excluded.policy_rule,
		   rejection_reason = excluded.rejection_reason,
		   created_at = case
		     when approvals.created_at = '' then excluded.created_at
		     else approvals.created_at
		   end`,
		a.ID,
		a.TaskID,
		a.Reason,
		a.Status,
		a.RiskLevel,
		a.PolicyRule,
		a.RejectionReason,
		createdAt,
	)
	if err != nil && isPendingApprovalConflict(err) {
		return fmt.Errorf("%w: %v", ports.ErrPendingApprovalConflict, err)
	}
	return err
}

func (r *ApprovalRepository) Get(id string) (approval.Approval, error) {
	var a approval.Approval
	err := r.db.QueryRow(
		`select id, task_id, reason, state, risk_level, policy_rule, rejection_reason, created_at from approvals where id = ?`,
		id,
	).Scan(&a.ID, &a.TaskID, &a.Reason, &a.Status, &a.RiskLevel, &a.PolicyRule, &a.RejectionReason, &a.CreatedAt)
	return a, err
}

func (r *ApprovalRepository) FindPendingByTask(taskID string) (approval.Approval, error) {
	var a approval.Approval
	err := r.db.QueryRow(
		`select id, task_id, reason, state, risk_level, policy_rule, rejection_reason, created_at
		 from approvals
		 where task_id = ? and state = ?
		 limit 1`,
		taskID,
		approval.StatusPending,
	).Scan(&a.ID, &a.TaskID, &a.Reason, &a.Status, &a.RiskLevel, &a.PolicyRule, &a.RejectionReason, &a.CreatedAt)
	return a, err
}

func (r *ApprovalRepository) FindLatestByTask(taskID string) (approval.Approval, error) {
	var a approval.Approval
	err := r.db.QueryRow(
		`select id, task_id, reason, state, risk_level, policy_rule, rejection_reason, created_at
		 from approvals
		 where task_id = ?
		 order by created_at desc, id desc
		 limit 1`,
		taskID,
	).Scan(&a.ID, &a.TaskID, &a.Reason, &a.Status, &a.RiskLevel, &a.PolicyRule, &a.RejectionReason, &a.CreatedAt)
	return a, err
}

func isPendingApprovalConflict(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "approvals_pending_task_idx") ||
		strings.Contains(msg, "UNIQUE constraint failed: approvals.task_id")
}
