package sqlite

import (
	"database/sql"

	modulepkg "github.com/sine-io/foreman/internal/domain/module"
)

type ModuleRepository struct {
	db *sql.DB
}

func NewModuleRepository(db *sql.DB) *ModuleRepository {
	return &ModuleRepository{db: db}
}

func (r *ModuleRepository) Save(m modulepkg.Module) error {
	_, err := r.db.Exec(
		`insert into modules (id, project_id, name, board_state) values (?, ?, ?, ?)
		 on conflict(id) do update set project_id = excluded.project_id, name = excluded.name, board_state = excluded.board_state`,
		m.ID,
		m.ProjectID,
		m.Name,
		m.State,
	)
	return err
}

func (r *ModuleRepository) Get(id string) (modulepkg.Module, error) {
	var m modulepkg.Module
	err := r.db.QueryRow(
		`select id, project_id, name, board_state from modules where id = ?`,
		id,
	).Scan(&m.ID, &m.ProjectID, &m.Name, &m.State)
	return m, err
}
