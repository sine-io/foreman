package http

import (
	"context"
	nethttp "net/http"

	"github.com/gin-gonic/gin"
	"github.com/sine-io/foreman/internal/adapters/gateway/openclaw"
	"github.com/sine-io/foreman/internal/app/command"
	"github.com/sine-io/foreman/internal/app/query"
)

type App interface {
	ModuleBoard(projectID string) (query.ModuleBoardView, error)
	TaskBoard(projectID string) (query.TaskBoardView, error)
	RunDetail(runID string) (query.RunDetailView, error)
	ApproveTask(command.ApproveTaskCommand) (string, error)
	RetryTask(command.RetryTaskCommand) (string, error)
	CancelTask(command.CancelTaskCommand) (string, error)
	ReprioritizeTask(command.ReprioritizeTaskCommand) (string, error)
	OpenClawCommand(context.Context, openclaw.Envelope) (openclaw.Response, error)
}

type BoardHandlers struct {
	app App
}

func NewBoardHandlers(app App) *BoardHandlers {
	return &BoardHandlers{app: app}
}

func (h *BoardHandlers) ModuleBoard(c *gin.Context) {
	view, err := h.app.ModuleBoard(c.Query("project_id"))
	if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(nethttp.StatusOK, view)
}

func (h *BoardHandlers) TaskBoard(c *gin.Context) {
	view, err := h.app.TaskBoard(c.Query("project_id"))
	if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(nethttp.StatusOK, view)
}

func (h *BoardHandlers) RunDetail(c *gin.Context) {
	view, err := h.app.RunDetail(c.Param("id"))
	if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(nethttp.StatusOK, view)
}

func (h *BoardHandlers) ApproveTask(c *gin.Context) {
	state, err := h.app.ApproveTask(command.ApproveTaskCommand{TaskID: c.Param("id")})
	if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(nethttp.StatusOK, taskActionResponse{State: state})
}

func (h *BoardHandlers) RetryTask(c *gin.Context) {
	state, err := h.app.RetryTask(command.RetryTaskCommand{TaskID: c.Param("id")})
	if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(nethttp.StatusOK, taskActionResponse{State: state})
}

func (h *BoardHandlers) CancelTask(c *gin.Context) {
	state, err := h.app.CancelTask(command.CancelTaskCommand{TaskID: c.Param("id")})
	if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(nethttp.StatusOK, taskActionResponse{State: state})
}

func (h *BoardHandlers) ReprioritizeTask(c *gin.Context) {
	var req reprioritizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	state, err := h.app.ReprioritizeTask(command.ReprioritizeTaskCommand{
		TaskID:   c.Param("id"),
		Priority: req.Priority,
	})
	if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(nethttp.StatusOK, taskActionResponse{State: state})
}

func (h *BoardHandlers) OpenClawCommand(c *gin.Context) {
	var env openclaw.Envelope
	if err := c.ShouldBindJSON(&env); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.app.OpenClawCommand(c.Request.Context(), env)
	if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(nethttp.StatusOK, resp)
}
