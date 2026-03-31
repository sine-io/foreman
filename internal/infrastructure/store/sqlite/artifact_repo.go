package sqlite

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
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

func (r *ArtifactRepository) Create(taskID, runID, kind, path string) (string, error) {
	id := fmt.Sprintf("artifact-%d", time.Now().UnixNano())
	createdAt := sortableTimestamp(time.Now())
	displayPath := sanitizeArtifactDisplayPath(path)

	var runRef any
	if runID != "" {
		runRef = runID
	}

	_, err := r.db.Exec(
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

func sanitizeArtifactDisplayPath(path string) string {
	cleaned := filepath.ToSlash(filepath.Clean(path))
	if cleaned == "." {
		return ""
	}
	if idx := strings.Index(cleaned, "/tasks/"); idx >= 0 {
		return strings.TrimPrefix(cleaned[idx+1:], "/")
	}
	if idx := strings.Index(cleaned, "tasks/"); idx >= 0 {
		return cleaned[idx:]
	}
	return strings.TrimLeft(cleaned, "/")
}
