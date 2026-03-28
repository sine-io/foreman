package sqlite

import (
	"database/sql"
	"fmt"
	"time"
)

type LeaseRepository struct {
	db dbtx
}

func NewLeaseRepository(db *sql.DB) *LeaseRepository {
	return newLeaseRepository(db)
}

func newLeaseRepository(db dbtx) *LeaseRepository {
	return &LeaseRepository{db: db}
}

func (r *LeaseRepository) Acquire(taskID, scopeKey string) error {
	_, err := r.db.Exec(
		`insert into leases (id, task_id, scope_key, state) values (?, ?, ?, 'active')`,
		fmt.Sprintf("lease-%d", time.Now().UnixNano()),
		taskID,
		scopeKey,
	)
	return err
}

func (r *LeaseRepository) Release(taskID, scopeKey string) error {
	_, err := r.db.Exec(
		`update leases set state = 'released' where task_id = ? and scope_key = ? and state = 'active'`,
		taskID,
		scopeKey,
	)
	return err
}
