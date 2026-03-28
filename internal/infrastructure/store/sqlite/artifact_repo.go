package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/sine-io/foreman/internal/ports"
)

type ArtifactRepository struct {
	db dbtx
}

func NewArtifactRepository(db *sql.DB) *ArtifactRepository {
	return newArtifactRepository(db)
}

func newArtifactRepository(db dbtx) *ArtifactRepository {
	return &ArtifactRepository{db: db}
}

func (r *ArtifactRepository) Create(taskID, kind, path string) (string, error) {
	id := fmt.Sprintf("artifact-%d", time.Now().UnixNano())
	createdAt := sortableTimestamp(time.Now())

	_, err := r.db.Exec(
		`insert into artifacts (id, task_id, kind, path, summary, created_at) values (?, ?, ?, ?, '', ?)`,
		id,
		taskID,
		kind,
		path,
		createdAt,
	)
	if err != nil {
		return "", err
	}

	return id, nil
}

func (r *ArtifactRepository) Get(id string) (ports.ArtifactRecord, error) {
	var row ports.ArtifactRecord
	err := r.db.QueryRow(
		`select id, task_id, kind, path, summary from artifacts where id = ?`,
		id,
	).Scan(&row.ID, &row.TaskID, &row.Kind, &row.Path, &row.Summary)
	return row, err
}
