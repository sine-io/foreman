package command

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	domainapproval "github.com/sine-io/foreman/internal/domain/approval"
	domainpolicy "github.com/sine-io/foreman/internal/domain/policy"
	"github.com/sine-io/foreman/internal/domain/task"
	"github.com/sine-io/foreman/internal/ports"
)

type dispatchPolicy interface {
	Evaluate(action string) domainpolicy.Decision
}

type DispatchTaskCommand struct {
	TaskID          string
	RequestedAction string
}

type DispatchTaskResult struct {
	TaskState      string
	RunState       string
	ApprovalID     string
	ApprovalReason string
	ArtifactIDs    []string
}

type DispatchTaskHandler struct {
	Tx        ports.Transactor
	Tasks     ports.TaskRepository
	Leases    ports.LeaseRepository
	Policy    dispatchPolicy
	Runner    ports.Runner
	Approvals ports.ApprovalRepository
	Runs      ports.RunRepository
	Artifacts ports.ArtifactRepository
}

func NewDispatchTaskHandler(
	tx ports.Transactor,
	tasks ports.TaskRepository,
	leases ports.LeaseRepository,
	policy dispatchPolicy,
	runner ports.Runner,
	approvals ports.ApprovalRepository,
	runs ports.RunRepository,
	artifacts ports.ArtifactRepository,
) *DispatchTaskHandler {
	return &DispatchTaskHandler{
		Tx:        tx,
		Tasks:     tasks,
		Leases:    leases,
		Policy:    policy,
		Runner:    runner,
		Approvals: approvals,
		Runs:      runs,
		Artifacts: artifacts,
	}
}

func (h *DispatchTaskHandler) Handle(cmd DispatchTaskCommand) (DispatchTaskResult, error) {
	repoTask, err := h.Tasks.Get(cmd.TaskID)
	if err != nil {
		return DispatchTaskResult{}, err
	}

	requestedAction := cmd.RequestedAction
	if requestedAction == "" {
		requestedAction = repoTask.Summary
	}

	if persistedRun, err := h.Runs.FindByTask(repoTask.ID); err == nil && isAuthoritativeRun(persistedRun) {
		return h.authoritativeRunResult(repoTask, persistedRun)
	} else if err != nil && err != sql.ErrNoRows {
		return DispatchTaskResult{}, err
	}

	decision := h.Policy.Evaluate(requestedAction)
	if decision.RequiresApproval {
		approved, err := h.hasApprovedDispatch(repoTask.ID)
		if err != nil {
			return DispatchTaskResult{}, err
		}
		if !approved {
			return h.handleApprovalDispatch(cmd.TaskID, decision)
		}
	}

	if err := h.Leases.Acquire(repoTask.ID, repoTask.WriteScope); err != nil {
		return DispatchTaskResult{}, err
	}
	if persistedRun, err := h.Runs.FindByTask(repoTask.ID); err == nil && isAuthoritativeRun(persistedRun) {
		return h.authoritativeRunResult(repoTask, persistedRun)
	} else if err != nil && err != sql.ErrNoRows {
		releaseErr := h.Leases.Release(repoTask.ID, repoTask.WriteScope)
		if releaseErr != nil {
			return DispatchTaskResult{}, errors.Join(err, releaseErr)
		}
		return DispatchTaskResult{}, err
	}

	run, err := h.Runner.Dispatch(ports.RunRequest{
		TaskID:     repoTask.ID,
		Command:    repoTask.Summary,
		WriteScope: repoTask.WriteScope,
	})
	if err != nil {
		releaseErr := h.Leases.Release(repoTask.ID, repoTask.WriteScope)
		if releaseErr != nil {
			return DispatchTaskResult{}, errors.Join(err, releaseErr)
		}
		return DispatchTaskResult{}, err
	}

	var out DispatchTaskResult
	if err := h.Tx.WithinTransaction(context.Background(), func(_ context.Context, repos ports.TransactionRepositories) error {
		if err := repos.Runs.Save(run); err != nil {
			return err
		}

		persistedTask, err := repos.Tasks.Get(repoTask.ID)
		if err != nil {
			return err
		}

		persistedTask.State = task.TaskStateRunning
		if run.State == "completed" {
			persistedTask.State = task.TaskStateCompleted
		}
		if err := repos.Tasks.Save(persistedTask); err != nil {
			return err
		}

		artifactID, err := repos.Artifacts.Create(repoTask.ID, "assistant_summary", run.AssistantSummaryPath)
		if err != nil {
			return err
		}

		out = DispatchTaskResult{
			TaskState:   string(persistedTask.State),
			RunState:    run.State,
			ArtifactIDs: []string{artifactID},
		}
		return nil
	}); err != nil {
		persistErr := fmt.Errorf("persist dispatch result: %w", err)
		releaseErr := h.Leases.Release(repoTask.ID, repoTask.WriteScope)
		if releaseErr != nil {
			return DispatchTaskResult{}, errors.Join(persistErr, releaseErr)
		}
		return DispatchTaskResult{}, persistErr
	}

	if run.State == "completed" {
		if err := h.Leases.Release(repoTask.ID, repoTask.WriteScope); err != nil {
			return DispatchTaskResult{}, err
		}
	}

	return out, nil
}

func (h *DispatchTaskHandler) handleApprovalDispatch(taskID string, decision domainpolicy.Decision) (DispatchTaskResult, error) {
	var out DispatchTaskResult

	err := h.Tx.WithinTransaction(context.Background(), func(_ context.Context, repos ports.TransactionRepositories) error {
		repoTask, err := repos.Tasks.Get(taskID)
		if err != nil {
			return err
		}

		record, err := repos.Approvals.FindPendingByTask(taskID)
		if err != nil {
			if err != sql.ErrNoRows {
				return err
			}

			record = domainapproval.New(nextID("approval"), repoTask.ID, decision.Reason)
			record.RiskLevel = decision.RiskLevel
			record.PolicyRule = decision.PolicyRule
			if err := repos.Approvals.Save(record); err != nil {
				if !errors.Is(err, ports.ErrPendingApprovalConflict) {
					return err
				}
				existing, findErr := repos.Approvals.FindPendingByTask(taskID)
				if findErr != nil {
					return findErr
				}
				record = existing
			}
		}

		repoTask.State = task.TaskStateWaitingApproval
		if err := repos.Tasks.Save(repoTask); err != nil {
			return err
		}

		out = DispatchTaskResult{
			TaskState:      string(repoTask.State),
			ApprovalID:     record.ID,
			ApprovalReason: record.Reason,
		}
		return nil
	})
	if err != nil {
		return DispatchTaskResult{}, err
	}

	return out, nil
}

func isAuthoritativeRun(run ports.Run) bool {
	return run.State == "running" || run.State == "completed"
}

func (h *DispatchTaskHandler) authoritativeRunResult(repoTask task.Task, run ports.Run) (DispatchTaskResult, error) {
	if run.State == "completed" {
		if err := h.Leases.Release(repoTask.ID, repoTask.WriteScope); err != nil {
			return DispatchTaskResult{}, err
		}
	}
	return persistedRunResult(repoTask, run), nil
}

func (h *DispatchTaskHandler) hasApprovedDispatch(taskID string) (bool, error) {
	latest, err := h.Approvals.FindLatestByTask(taskID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	return latest.Status == domainapproval.StatusApproved, nil
}

func (h *DispatchTaskHandler) currentAuthoritativeRunState(taskID string) (string, error) {
	return currentRunState(h.Runs, taskID)
}

func persistedRunResult(repoTask task.Task, run ports.Run) DispatchTaskResult {
	taskState := string(repoTask.State)
	switch run.State {
	case "running":
		taskState = string(task.TaskStateRunning)
	case "completed":
		taskState = string(task.TaskStateCompleted)
	}

	return DispatchTaskResult{
		TaskState: taskState,
		RunState:  run.State,
	}
}
