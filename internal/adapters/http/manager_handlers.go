package http

import (
	"context"
	"database/sql"
	"errors"
	nethttp "net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sine-io/foreman/internal/app/manageragent"
)

type ManagerApp interface {
	Handle(context.Context, manageragent.Request) (manageragent.Response, error)
	TaskStatus(context.Context, string, string) (manageragent.TaskStatusView, error)
	BoardSnapshot(context.Context, string) (manageragent.BoardSnapshotView, error)
}

type ManagerHandlers struct {
	app ManagerApp
}

func NewManagerHandlers(app ManagerApp) *ManagerHandlers {
	return &ManagerHandlers{app: app}
}

func (h *ManagerHandlers) ManagerCommand(c *gin.Context) {
	var req managerCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.app.Handle(c.Request.Context(), manageragent.Request{
		Kind:        req.Kind,
		SessionID:   req.SessionID,
		ProjectID:   req.ProjectID,
		ModuleID:    req.ModuleID,
		TaskID:      req.TaskID,
		Name:        req.Name,
		Summary:     req.Summary,
		Description: req.Description,
		RepoRoot:    req.RepoRoot,
		TaskType:    req.TaskType,
		WriteScope:  req.WriteScope,
		Acceptance:  req.Acceptance,
		Priority:    req.Priority,
	})
	if err != nil {
		respondManagerError(c, err)
		return
	}

	c.JSON(nethttp.StatusOK, managerCommandResponse{
		Kind:      resp.Kind,
		ProjectID: resp.ProjectID,
		ModuleID:  resp.ModuleID,
		TaskID:    resp.TaskID,
		Summary:   resp.Summary,
	})
}

func (h *ManagerHandlers) ManagerTaskStatus(c *gin.Context) {
	view, err := h.app.TaskStatus(c.Request.Context(), c.Query("project_id"), c.Param("id"))
	if err != nil {
		respondManagerError(c, err)
		return
	}

	c.JSON(nethttp.StatusOK, managerTaskStatusResponse{
		TaskID:         view.TaskID,
		ProjectID:      view.ProjectID,
		ModuleID:       view.ModuleID,
		Summary:        view.Summary,
		State:          view.State,
		RunID:          view.RunID,
		RunState:       view.RunState,
		ApprovalID:     view.ApprovalID,
		ApprovalReason: view.ApprovalReason,
		ApprovalState:  view.ApprovalState,
	})
}

func (h *ManagerHandlers) ManagerBoardSnapshot(c *gin.Context) {
	view, err := h.app.BoardSnapshot(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondManagerError(c, err)
		return
	}

	resp := managerBoardSnapshotResponse{
		ProjectID: view.ProjectID,
		Modules:   make(map[string][]managerModuleSnapshotResponse, len(view.Modules)),
		Tasks:     make(map[string][]managerTaskSnapshotResponse, len(view.Tasks)),
	}

	for column, modules := range view.Modules {
		resp.Modules[column] = make([]managerModuleSnapshotResponse, 0, len(modules))
		for _, module := range modules {
			resp.Modules[column] = append(resp.Modules[column], managerModuleSnapshotResponse{
				ModuleID: module.ModuleID,
				Name:     module.Name,
				State:    module.State,
			})
		}
	}

	for column, tasks := range view.Tasks {
		resp.Tasks[column] = make([]managerTaskSnapshotResponse, 0, len(tasks))
		for _, task := range tasks {
			resp.Tasks[column] = append(resp.Tasks[column], managerTaskSnapshotResponse{
				TaskID:          task.TaskID,
				ModuleID:        task.ModuleID,
				Summary:         task.Summary,
				State:           task.State,
				Priority:        task.Priority,
				PendingApproval: task.PendingApproval,
			})
		}
	}

	c.JSON(nethttp.StatusOK, resp)
}

func respondManagerError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		c.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
	case isManagerClientError(err):
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

func isManagerClientError(err error) bool {
	if err == nil {
		return false
	}

	msg := err.Error()
	return strings.Contains(msg, "required") ||
		strings.Contains(msg, "unsupported") ||
		strings.Contains(msg, "does not belong")
}
