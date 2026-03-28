package sqlite

import (
	"database/sql"
	"time"

	"github.com/sine-io/foreman/internal/ports"
)

type RunRepository struct {
	db *sql.DB
}

func NewRunRepository(db *sql.DB) *RunRepository {
	return &RunRepository{db: db}
}

func (r *RunRepository) Save(run ports.Run) error {
	createdAt := run.CreatedAt
	if createdAt == "" {
		createdAt = time.Now().UTC().Format(time.RFC3339Nano)
	}

	_, err := r.db.Exec(
		`insert into runs (id, task_id, runner_kind, state, created_at) values (?, ?, ?, ?, ?)
		 on conflict(id) do update set
		   task_id = excluded.task_id,
		   runner_kind = excluded.runner_kind,
		   state = excluded.state,
		   created_at = case
		     when runs.created_at = '' then excluded.created_at
		     else runs.created_at
		   end`,
		run.ID,
		run.TaskID,
		run.RunnerKind,
		run.State,
		createdAt,
	)
	return err
}

func (r *RunRepository) Get(id string) (ports.Run, error) {
	var run ports.Run
	err := r.db.QueryRow(
		`select id, task_id, runner_kind, state, created_at from runs where id = ?`,
		id,
	).Scan(&run.ID, &run.TaskID, &run.RunnerKind, &run.State, &run.CreatedAt)
	return run, err
}

func (r *RunRepository) FindByTask(taskID string) (ports.Run, error) {
	var run ports.Run
	err := r.db.QueryRow(
		`select id, task_id, runner_kind, state, created_at
		 from runs
		 where task_id = ?
		 order by created_at desc, id desc
		 limit 1`,
		taskID,
	).Scan(&run.ID, &run.TaskID, &run.RunnerKind, &run.State, &run.CreatedAt)
	return run, err
}
