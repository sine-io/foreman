package sqlite

import (
	"database/sql"

	"github.com/sine-io/foreman/internal/domain/project"
)

type ProjectRepository struct {
	db dbtx
}

func NewProjectRepository(db *sql.DB) *ProjectRepository {
	return newProjectRepository(db)
}

func newProjectRepository(db dbtx) *ProjectRepository {
	return &ProjectRepository{db: db}
}

func (r *ProjectRepository) Save(p project.Project) error {
	_, err := r.db.Exec(
		`insert into projects (id, name, repo_root) values (?, ?, ?)
		 on conflict(id) do update set name = excluded.name, repo_root = excluded.repo_root`,
		p.ID,
		p.Name,
		p.RepoRoot,
	)
	return err
}

func (r *ProjectRepository) Get(id string) (project.Project, error) {
	var p project.Project
	err := r.db.QueryRow(
		`select id, name, repo_root from projects where id = ?`,
		id,
	).Scan(&p.ID, &p.Name, &p.RepoRoot)
	return p, err
}
