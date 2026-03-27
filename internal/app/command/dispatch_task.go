package command

import (
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
	TaskState   string
	RunState    string
	ApprovalID  string
	ArtifactIDs []string
}

type DispatchTaskHandler struct {
	Tasks     ports.TaskRepository
	Leases    ports.LeaseRepository
	Policy    dispatchPolicy
	Runner    ports.Runner
	Approvals ports.ApprovalRepository
	Runs      ports.RunRepository
	Artifacts ports.ArtifactRepository
}

func NewDispatchTaskHandler(
	tasks ports.TaskRepository,
	leases ports.LeaseRepository,
	policy dispatchPolicy,
	runner ports.Runner,
	approvals ports.ApprovalRepository,
	runs ports.RunRepository,
	artifacts ports.ArtifactRepository,
) *DispatchTaskHandler {
	return &DispatchTaskHandler{
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

	decision := h.Policy.Evaluate(requestedAction)
	if decision.RequiresApproval {
		record := domainapproval.New(nextID("approval"), repoTask.ID, decision.Reason)
		if err := h.Approvals.Save(record); err != nil {
			return DispatchTaskResult{}, err
		}

		repoTask.State = task.TaskStateWaitingApproval
		if err := h.Tasks.Save(repoTask); err != nil {
			return DispatchTaskResult{}, err
		}

		return DispatchTaskResult{
			TaskState:  string(repoTask.State),
			ApprovalID: record.ID,
		}, nil
	}

	if err := h.Leases.Acquire(repoTask.ID, repoTask.WriteScope); err != nil {
		return DispatchTaskResult{}, err
	}

	run, err := h.Runner.Dispatch(ports.RunRequest{
		TaskID:     repoTask.ID,
		Command:    repoTask.Summary,
		WriteScope: repoTask.WriteScope,
	})
	if err != nil {
		return DispatchTaskResult{}, err
	}

	if err := h.Runs.Save(run); err != nil {
		return DispatchTaskResult{}, err
	}

	repoTask.State = task.TaskStateRunning
	if run.State == "completed" {
		repoTask.State = task.TaskStateCompleted
	}
	if err := h.Tasks.Save(repoTask); err != nil {
		return DispatchTaskResult{}, err
	}

	artifactID, err := h.Artifacts.Create(repoTask.ID, "assistant_summary", run.AssistantSummaryPath)
	if err != nil {
		return DispatchTaskResult{}, err
	}

	if run.State == "completed" {
		if err := h.Leases.Release(repoTask.WriteScope); err != nil {
			return DispatchTaskResult{}, err
		}
	}

	return DispatchTaskResult{
		TaskState:   string(repoTask.State),
		RunState:    run.State,
		ArtifactIDs: []string{artifactID},
	}, nil
}
