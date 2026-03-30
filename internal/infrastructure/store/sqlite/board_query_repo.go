package sqlite

import (
	"database/sql"
	"errors"

	"github.com/sine-io/foreman/internal/domain/approval"
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

func (r *BoardQueryRepository) ListApprovalWorkbenchQueue(projectID string) ([]ports.ApprovalWorkbenchQueueRow, error) {
	rows, err := r.db.Query(
		`select a.id, a.task_id, t.summary, a.risk_level, t.priority, a.created_at
		 from approvals a
		 join tasks t on t.id = a.task_id
		 join modules m on m.id = t.module_id
		 where m.project_id = ? and a.state = 'pending'`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ports.ApprovalWorkbenchQueueRow
	for rows.Next() {
		var row ports.ApprovalWorkbenchQueueRow
		if err := rows.Scan(&row.ApprovalID, &row.TaskID, &row.Summary, &row.RiskLevel, &row.Priority, &row.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, row)
	}

	return result, rows.Err()
}

func (r *BoardQueryRepository) GetApprovalWorkbenchDetail(approvalID string) (ports.ApprovalWorkbenchDetailRow, error) {
	var detail ports.ApprovalWorkbenchDetailRow
	err := r.db.QueryRow(
		`select
		   a.id,
		   a.task_id,
		   t.summary,
		   a.reason,
		   a.state,
		   a.risk_level,
		   a.policy_rule,
		   a.rejection_reason,
		   t.priority,
		   a.created_at,
		   t.state
		 from approvals a
		 join tasks t on t.id = a.task_id
		 where a.id = ?`,
		approvalID,
	).Scan(
		&detail.ApprovalID,
		&detail.TaskID,
		&detail.Summary,
		&detail.Reason,
		&detail.ApprovalState,
		&detail.RiskLevel,
		&detail.PolicyRule,
		&detail.RejectionReason,
		&detail.Priority,
		&detail.CreatedAt,
		&detail.TaskState,
	)
	if err != nil {
		return ports.ApprovalWorkbenchDetailRow{}, err
	}

	run, err := r.latestRunForTask(detail.TaskID)
	if err != nil {
		return ports.ApprovalWorkbenchDetailRow{}, err
	}
	detail.RunID = run.ID
	detail.RunState = run.State

	artifacts, preview, err := r.taskArtifacts(detail.TaskID)
	if err != nil {
		return ports.ApprovalWorkbenchDetailRow{}, err
	}
	detail.Artifacts = artifacts
	detail.AssistantSummary = preview

	return detail, nil
}

func (r *BoardQueryRepository) GetTaskWorkbench(taskID string) (ports.TaskWorkbenchRow, error) {
	var row ports.TaskWorkbenchRow
	err := r.db.QueryRow(
		`select
		   t.id,
		   m.project_id,
		   t.module_id,
		   t.summary,
		   t.state,
		   t.priority,
		   t.write_scope,
		   t.task_type,
		   t.acceptance
		 from tasks t
		 join modules m on m.id = t.module_id
		 where t.id = ?`,
		taskID,
	).Scan(
		&row.TaskID,
		&row.ProjectID,
		&row.ModuleID,
		&row.Summary,
		&row.TaskState,
		&row.Priority,
		&row.WriteScope,
		&row.TaskType,
		&row.Acceptance,
	)
	if err != nil {
		return ports.TaskWorkbenchRow{}, err
	}

	run, err := r.latestRunForTask(taskID)
	if err != nil {
		return ports.TaskWorkbenchRow{}, err
	}
	row.LatestRunID = run.ID
	row.LatestRunState = run.State
	row.LatestRunSummary = latestRunSummary(run)

	approval, err := r.latestApprovalForTask(taskID)
	if err != nil {
		return ports.TaskWorkbenchRow{}, err
	}
	row.LatestApprovalID = approval.ID
	row.LatestApprovalState = string(approval.Status)
	row.LatestApprovalReason = approval.Reason

	artifacts, _, err := r.taskArtifacts(taskID)
	if err != nil {
		return ports.TaskWorkbenchRow{}, err
	}
	row.Artifacts = artifacts

	return row, nil
}

func (r *BoardQueryRepository) latestRunForTask(taskID string) (ports.Run, error) {
	var run ports.Run
	err := r.db.QueryRow(
		`select id, task_id, runner_kind, state, created_at
		 from runs
		 where task_id = ?
		 order by created_at desc, id desc
		 limit 1`,
		taskID,
	).Scan(&run.ID, &run.TaskID, &run.RunnerKind, &run.State, &run.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ports.Run{}, nil
	}
	if err != nil {
		return ports.Run{}, err
	}
	return run, nil
}

func latestRunSummary(run ports.Run) string {
	if run.ID == "" {
		return ""
	}
	if run.RunnerKind == "" {
		return run.State
	}
	return run.RunnerKind + " run is " + run.State
}

func (r *BoardQueryRepository) latestApprovalForTask(taskID string) (approval.Approval, error) {
	var record approval.Approval
	err := r.db.QueryRow(
		`select id, task_id, reason, state
		 from approvals
		 where task_id = ?
		 order by created_at desc, id desc
		 limit 1`,
		taskID,
	).Scan(&record.ID, &record.TaskID, &record.Reason, &record.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return approval.Approval{}, nil
	}
	if err != nil {
		return approval.Approval{}, err
	}
	return record, nil
}

func (r *BoardQueryRepository) taskArtifacts(taskID string) ([]ports.ArtifactRecord, string, error) {
	rows, err := r.db.Query(
		`select id, task_id, kind, path, summary
		 from artifacts
		 where task_id = ?
		 order by created_at desc, id desc`,
		taskID,
	)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var (
		artifacts []ports.ArtifactRecord
		preview   string
	)
	for rows.Next() {
		var artifact ports.ArtifactRecord
		if err := rows.Scan(&artifact.ID, &artifact.TaskID, &artifact.Kind, &artifact.Path, &artifact.Summary); err != nil {
			return nil, "", err
		}
		if preview == "" && artifact.Kind == "assistant_summary" {
			preview = artifact.Summary
		}
		artifacts = append(artifacts, artifact)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}

	return artifacts, preview, nil
}
