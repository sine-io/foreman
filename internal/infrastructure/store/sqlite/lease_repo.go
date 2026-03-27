package sqlite

import (
	"database/sql"
	"fmt"
	"time"
)

type LeaseRepository struct {
	db *sql.DB
}

func NewLeaseRepository(db *sql.DB) *LeaseRepository {
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

func (r *LeaseRepository) Release(scopeKey string) error {
	_, err := r.db.Exec(
		`update leases set state = 'released' where scope_key = ? and state = 'active'`,
		scopeKey,
	)
	return err
}
