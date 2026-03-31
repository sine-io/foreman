package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/sine-io/foreman/internal/ports"
)

type ArtifactRepository struct {
	db    dbtx
	store ports.ArtifactStore
}

func NewArtifactRepository(db *sql.DB, store ports.ArtifactStore) *ArtifactRepository {
	return newArtifactRepository(db, store)
}

func newArtifactRepository(db dbtx, store ports.ArtifactStore) *ArtifactRepository {
	return &ArtifactRepository{db: db, store: store}
}

func (r *ArtifactRepository) Create(taskID, runID, kind, path string) (string, error) {
	displayPath, err := r.store.ResolveDisplayPath(path)
	if err != nil {
		return "", err
	}

	id := fmt.Sprintf("artifact-%d", time.Now().UnixNano())
	createdAt := sortableTimestamp(time.Now())

	var runRef any
	if runID != "" {
		runRef = runID
	}

	_, err = r.db.Exec(
		`insert into artifacts (id, task_id, run_id, kind, path, storage_path, summary, created_at) values (?, ?, ?, ?, ?, ?, '', ?)`,
		id,
		taskID,
		runRef,
		kind,
		displayPath,
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
	var runID sql.NullString
	var storagePath sql.NullString
	err := r.db.QueryRow(
		`select id, task_id, run_id, kind, path, nullif(storage_path, ''), summary from artifacts where id = ?`,
		id,
	).Scan(&row.ID, &row.TaskID, &runID, &row.Kind, &row.Path, &storagePath, &row.Summary)
	if runID.Valid {
		row.RunID = runID.String
	}
	if storagePath.Valid {
		row.StoragePath = storagePath.String
	} else {
		row.StoragePath = row.Path
	}
	return row, err
}
