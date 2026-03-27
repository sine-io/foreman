package sqlite

import (
	"database/sql"

	"github.com/sine-io/foreman/internal/domain/task"
)

type TaskRepository struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Save(t task.Task) error {
	_, err := r.db.Exec(
		`insert into tasks (id, module_id, task_type, state, write_scope) values (?, ?, ?, ?, ?)
		 on conflict(id) do update set module_id = excluded.module_id, task_type = excluded.task_type, state = excluded.state, write_scope = excluded.write_scope`,
		t.ID,
		t.ModuleID,
		t.Type,
		t.State,
		t.WriteScope,
	)
	return err
}

func (r *TaskRepository) Get(id string) (task.Task, error) {
	var t task.Task
	err := r.db.QueryRow(
		`select id, module_id, task_type, state, write_scope from tasks where id = ?`,
		id,
	).Scan(&t.ID, &t.ModuleID, &t.Type, &t.State, &t.WriteScope)
	return t, err
}
