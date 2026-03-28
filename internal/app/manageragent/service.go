package manageragent

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/sine-io/foreman/internal/app/command"
	"github.com/sine-io/foreman/internal/app/query"
)

const (
	defaultProjectID = "demo"
	defaultModuleID  = "module-default"
)

type Service struct {
	CreateTask   *command.CreateTaskHandler
	DispatchTask *command.DispatchTaskHandler
	QueryBoard   *query.TaskBoardQuery
}

func NewService(
	createTask *command.CreateTaskHandler,
	dispatchTask *command.DispatchTaskHandler,
	queryBoard *query.TaskBoardQuery,
) *Service {
	return &Service{
		CreateTask:   createTask,
		DispatchTask: dispatchTask,
		QueryBoard:   queryBoard,
	}
}

func (s *Service) Handle(ctx context.Context, req Request) (Response, error) {
	_ = ctx

	switch req.Kind {
	case "create_task":
		taskDTO, err := s.CreateTask.Handle(command.CreateTaskCommand{
			ModuleID:   defaultModuleID,
			Title:      req.Summary,
			TaskType:   "write",
			WriteScope: "repo:" + defaultProjectID,
			Acceptance: req.Summary,
			Priority:   10,
		})
		if err != nil {
			return Response{}, err
		}

		result, err := s.DispatchTask.Handle(command.DispatchTaskCommand{
			TaskID:          taskDTO.ID,
			RequestedAction: req.Summary,
		})
		if err != nil {
			return Response{}, err
		}

		if result.TaskState == "waiting_approval" {
			summary := taskDTO.Summary
			if result.ApprovalID != "" && s.DispatchTask.Approvals != nil {
				approval, err := s.DispatchTask.Approvals.Get(result.ApprovalID)
				if err == nil && approval.Reason != "" {
					summary = approval.Reason
				}
			}

			return Response{
				Kind:    "approval_needed",
				TaskID:  taskDTO.ID,
				Summary: summary,
			}, nil
		}

		return Response{
			Kind:    "completion",
			TaskID:  taskDTO.ID,
			Summary: taskDTO.Summary,
		}, nil
	default:
		return Response{}, fmt.Errorf("unsupported manager-agent action: %s", req.Kind)
	}
}

func (s *Service) TaskStatus(ctx context.Context, taskID string) (TaskStatusView, error) {
	_ = ctx

	board, err := s.QueryBoard.Execute(defaultProjectID)
	if err != nil {
		return TaskStatusView{}, err
	}

	for _, cards := range board.Columns {
		for _, card := range cards {
			if card.ID != taskID {
				continue
			}

			return TaskStatusView{
				TaskID:          card.ID,
				ModuleID:        card.ModuleID,
				Summary:         card.Summary,
				State:           card.State,
				Priority:        card.Priority,
				PendingApproval: card.PendingApproval,
			}, nil
		}
	}

	return TaskStatusView{}, sql.ErrNoRows
}
