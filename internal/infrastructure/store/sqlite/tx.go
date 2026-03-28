package sqlite

import (
	"context"
	"database/sql"

	"github.com/sine-io/foreman/internal/ports"
)

type Transactor struct {
	db *sql.DB
}

func NewTransactor(db *sql.DB) *Transactor {
	return &Transactor{db: db}
}

func (t *Transactor) WithinTransaction(
	ctx context.Context,
	fn func(context.Context, ports.TransactionRepositories) error,
) error {
	tx, err := t.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	repos := ports.TransactionRepositories{
		Projects:  newProjectRepository(tx),
		Modules:   newModuleRepository(tx),
		Tasks:     newTaskRepository(tx),
		Runs:      newRunRepository(tx),
		Approvals: newApprovalRepository(tx),
		Artifacts: newArtifactRepository(tx),
		Leases:    newLeaseRepository(tx),
	}

	if err := fn(ctx, repos); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return err
	}

	return nil
}
