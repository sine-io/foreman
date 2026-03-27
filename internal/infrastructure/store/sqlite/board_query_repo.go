package sqlite

import (
	"database/sql"

	"github.com/sine-io/foreman/internal/ports"
)

type BoardQueryRepository struct {
	db *sql.DB
}

func NewBoardQueryRepository(db *sql.DB) *BoardQueryRepository {
	return &BoardQueryRepository{db: db}
}

func (r *BoardQueryRepository) ListModules(projectID string) ([]ports.ModuleBoardRow, error) {
	rows, err := r.db.Query(
		`select id, name, board_state from modules where project_id = ? order by name`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ports.ModuleBoardRow
	for rows.Next() {
		var row ports.ModuleBoardRow
		if err := rows.Scan(&row.ModuleID, &row.Name, &row.BoardState); err != nil {
			return nil, err
		}
		result = append(result, row)
	}

	return result, rows.Err()
}

func (r *BoardQueryRepository) ListTasks(projectID string) ([]ports.TaskBoardRow, error) {
	rows, err := r.db.Query(
		`select
		   t.id,
		   t.module_id,
		   t.summary,
		   t.state,
		   t.priority,
		   exists(select 1 from approvals a where a.task_id = t.id and a.state = 'pending')
		 from tasks t
		 join modules m on m.id = t.module_id
		 where m.project_id = ?
		 order by t.priority desc, t.id`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ports.TaskBoardRow
	for rows.Next() {
		var row ports.TaskBoardRow
		var pending int
		if err := rows.Scan(&row.TaskID, &row.ModuleID, &row.Summary, &row.State, &row.Priority, &pending); err != nil {
			return nil, err
		}
		row.PendingApproval = pending == 1
		result = append(result, row)
	}

	return result, rows.Err()
}

func (r *BoardQueryRepository) GetRunDetail(runID string) (ports.RunDetailRecord, error) {
	var detail ports.RunDetailRecord
	err := r.db.QueryRow(
		`select r.id, r.task_id, r.runner_kind, r.state, t.summary
		 from runs r
		 join tasks t on t.id = r.task_id
		 where r.id = ?`,
		runID,
	).Scan(
		&detail.Run.ID,
		&detail.Run.TaskID,
		&detail.Run.RunnerKind,
		&detail.Run.State,
		&detail.TaskSummary,
	)
	if err != nil {
		return ports.RunDetailRecord{}, err
	}

	rows, err := r.db.Query(
		`select id, task_id, kind, path, summary from artifacts where task_id = ? order by id`,
		detail.Run.TaskID,
	)
	if err != nil {
		return ports.RunDetailRecord{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var artifact ports.ArtifactRecord
		if err := rows.Scan(&artifact.ID, &artifact.TaskID, &artifact.Kind, &artifact.Path, &artifact.Summary); err != nil {
			return ports.RunDetailRecord{}, err
		}
		detail.Artifacts = append(detail.Artifacts, artifact)
	}

	return detail, rows.Err()
}

func (r *BoardQueryRepository) ListApprovals(projectID string) ([]ports.ApprovalQueueRow, error) {
	rows, err := r.db.Query(
		`select a.id, a.task_id, t.module_id, t.summary, a.reason, a.state
		 from approvals a
		 join tasks t on t.id = a.task_id
		 join modules m on m.id = t.module_id
		 where m.project_id = ? and a.state = 'pending'
		 order by a.id`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ports.ApprovalQueueRow
	for rows.Next() {
		var row ports.ApprovalQueueRow
		if err := rows.Scan(&row.ApprovalID, &row.TaskID, &row.ModuleID, &row.Summary, &row.Reason, &row.State); err != nil {
			return nil, err
		}
		result = append(result, row)
	}

	return result, rows.Err()
}
