package sqlite

import (
	"database/sql"

	"github.com/sine-io/foreman/internal/ports"
)

type RunRepository struct {
	db *sql.DB
}

func NewRunRepository(db *sql.DB) *RunRepository {
	return &RunRepository{db: db}
}

func (r *RunRepository) Save(run ports.Run) error {
	_, err := r.db.Exec(
		`insert into runs (id, task_id, runner_kind, state) values (?, ?, ?, ?)
		 on conflict(id) do update set task_id = excluded.task_id, runner_kind = excluded.runner_kind, state = excluded.state`,
		run.ID,
		run.TaskID,
		run.RunnerKind,
		run.State,
	)
	return err
}

func (r *RunRepository) Get(id string) (ports.Run, error) {
	var run ports.Run
	err := r.db.QueryRow(
		`select id, task_id, runner_kind, state from runs where id = ?`,
		id,
	).Scan(&run.ID, &run.TaskID, &run.RunnerKind, &run.State)
	return run, err
}

func (r *RunRepository) FindByTask(taskID string) (ports.Run, error) {
	var run ports.Run
	err := r.db.QueryRow(
		`select id, task_id, runner_kind, state from runs where task_id = ? order by rowid desc limit 1`,
		taskID,
	).Scan(&run.ID, &run.TaskID, &run.RunnerKind, &run.State)
	return run, err
}
