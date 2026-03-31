package http

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"mime"
	nethttp "net/http"
	"os"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sine-io/foreman/internal/app/command"
	"github.com/sine-io/foreman/internal/app/manageragent"
	"github.com/sine-io/foreman/internal/app/query"
)

type ManagerArtifactContent struct {
	Path        string
	ContentType string
	Size        int64
	Reader      io.ReadCloser
}

type ManagerApp interface {
	Handle(context.Context, manageragent.Request) (manageragent.Response, error)
	TaskStatus(context.Context, string, string) (manageragent.TaskStatusView, error)
	RunWorkbench(context.Context, string) (manageragent.RunWorkbenchView, error)
	ArtifactWorkbench(context.Context, string) (manageragent.ArtifactWorkbenchView, error)
	ArtifactContent(context.Context, string) (ManagerArtifactContent, error)
	TaskWorkbench(context.Context, string, string) (manageragent.TaskWorkbenchView, error)
	DispatchTaskWorkbench(context.Context, string, string) (manageragent.TaskWorkbenchActionResponse, error)
	RetryTaskWorkbench(context.Context, string, string) (manageragent.TaskWorkbenchActionResponse, error)
	CancelTaskWorkbench(context.Context, string, string) (manageragent.TaskWorkbenchActionResponse, error)
	ReprioritizeTaskWorkbench(context.Context, string, string, int) (manageragent.TaskWorkbenchActionResponse, error)
	BoardSnapshot(context.Context, string) (manageragent.BoardSnapshotView, error)
	ApprovalWorkbenchQueue(context.Context, string) (manageragent.ApprovalWorkbenchQueueView, error)
	ApprovalWorkbenchDetail(context.Context, string) (manageragent.ApprovalWorkbenchDetailView, error)
	ApproveApproval(context.Context, string) (manageragent.ApprovalWorkbenchActionResponse, error)
	RejectApproval(context.Context, string, string) (manageragent.ApprovalWorkbenchActionResponse, error)
	RetryApprovalDispatch(context.Context, string) (manageragent.ApprovalWorkbenchActionResponse, error)
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
		TaskID:          view.TaskID,
		ProjectID:       view.ProjectID,
		ModuleID:        view.ModuleID,
		Summary:         view.Summary,
		State:           view.State,
		Priority:        view.Priority,
		RunID:           view.RunID,
		RunState:        view.RunState,
		ApprovalID:      view.ApprovalID,
		ApprovalReason:  view.ApprovalReason,
		ApprovalState:   view.ApprovalState,
		PendingApproval: view.PendingApproval,
	})
}

func (h *ManagerHandlers) ManagerTaskWorkbench(c *gin.Context) {
	view, err := h.app.TaskWorkbench(c.Request.Context(), c.Query("project_id"), c.Param("id"))
	if err != nil {
		respondManagerError(c, err)
		return
	}

	c.JSON(nethttp.StatusOK, view)
}

func (h *ManagerHandlers) ManagerRunWorkbench(c *gin.Context) {
	view, err := h.app.RunWorkbench(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondManagerError(c, err)
		return
	}

	resp := managerRunWorkbenchResponse{
		RunID:              view.RunID,
		TaskID:             view.TaskID,
		ProjectID:          view.ProjectID,
		ModuleID:           view.ModuleID,
		TaskSummary:        view.TaskSummary,
		RunState:           view.RunState,
		RunnerKind:         view.RunnerKind,
		PrimarySummary:     view.PrimarySummary,
		RunCreatedAt:       view.RunCreatedAt,
		TaskWorkbenchURL:   view.TaskWorkbenchURL,
		RunWorkbenchURL:    runWorkbenchURL(view.RunID),
		ArtifactTargetURLs: view.ArtifactTargetURLs,
		Artifacts:          make([]managerRunWorkbenchArtifactResponse, 0, len(view.Artifacts)),
	}
	for _, artifact := range view.Artifacts {
		resp.Artifacts = append(resp.Artifacts, managerRunWorkbenchArtifactResponse{
			ID:      artifact.ID,
			Kind:    artifact.Kind,
			Path:    artifact.Path,
			Summary: artifact.Summary,
		})
	}

	c.JSON(nethttp.StatusOK, resp)
}

func (h *ManagerHandlers) ManagerArtifactWorkbench(c *gin.Context) {
	view, err := h.app.ArtifactWorkbench(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondManagerError(c, err)
		return
	}

	c.JSON(nethttp.StatusOK, artifactWorkbenchResponseDTO(view))
}

func (h *ManagerHandlers) ManagerArtifactContent(c *gin.Context) {
	content, err := h.app.ArtifactContent(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondManagerArtifactContentError(c, err)
		return
	}
	if content.Reader == nil {
		respondManagerArtifactContentError(c, errors.New("artifact content reader missing"))
		return
	}
	defer func() {
		_ = content.Reader.Close()
	}()

	contentType := strings.TrimSpace(content.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	c.DataFromReader(nethttp.StatusOK, content.Size, contentType, content.Reader, map[string]string{
		"X-Content-Type-Options": "nosniff",
		"Content-Disposition":    artifactContentDisposition(content.Path, contentType),
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

func (h *ManagerHandlers) ManagerApprovalWorkbenchQueue(c *gin.Context) {
	view, err := h.app.ApprovalWorkbenchQueue(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondManagerError(c, err)
		return
	}

	resp := managerApprovalQueueResponse{
		Items: make([]managerApprovalWorkbenchItemResponse, 0, len(view.Items)),
	}
	for _, item := range view.Items {
		resp.Items = append(resp.Items, managerApprovalWorkbenchItemResponse{
			ApprovalID: item.ApprovalID,
			TaskID:     item.TaskID,
			Summary:    item.Summary,
			RiskLevel:  item.RiskLevel,
			Priority:   item.Priority,
		})
	}

	c.JSON(nethttp.StatusOK, resp)
}

func (h *ManagerHandlers) ManagerApprovalWorkbenchDetail(c *gin.Context) {
	view, err := h.app.ApprovalWorkbenchDetail(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondManagerError(c, err)
		return
	}

	resp := managerApprovalDetailResponse{
		ApprovalID:              view.ApprovalID,
		TaskID:                  view.TaskID,
		Summary:                 view.Summary,
		Reason:                  view.Reason,
		ApprovalState:           view.ApprovalState,
		RiskLevel:               view.RiskLevel,
		PolicyRule:              view.PolicyRule,
		RejectionReason:         view.RejectionReason,
		Priority:                view.Priority,
		CreatedAt:               view.CreatedAt,
		TaskState:               view.TaskState,
		RunID:                   view.RunID,
		RunState:                view.RunState,
		RunDetailURL:            view.RunDetailURL,
		AssistantSummaryPreview: view.AssistantSummaryPreview,
		Artifacts:               make([]managerApprovalWorkbenchArtifactResponse, 0, len(view.Artifacts)),
	}
	for _, artifact := range view.Artifacts {
		resp.Artifacts = append(resp.Artifacts, managerApprovalWorkbenchArtifactResponse{
			ID:      artifact.ID,
			Kind:    artifact.Kind,
			Path:    artifact.Path,
			Summary: artifact.Summary,
		})
	}

	c.JSON(nethttp.StatusOK, resp)
}

func (h *ManagerHandlers) ApproveApproval(c *gin.Context) {
	resp, err := h.app.ApproveApproval(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondManagerError(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, approvalActionResponseDTO(resp))
}

func (h *ManagerHandlers) RejectApproval(c *gin.Context) {
	var req managerRejectApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.app.RejectApproval(c.Request.Context(), c.Param("id"), req.RejectionReason)
	if err != nil {
		respondManagerError(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, approvalActionResponseDTO(resp))
}

func (h *ManagerHandlers) RetryApprovalDispatch(c *gin.Context) {
	resp, err := h.app.RetryApprovalDispatch(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondManagerError(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, approvalActionResponseDTO(resp))
}

func (h *ManagerHandlers) DispatchTaskWorkbench(c *gin.Context) {
	resp, err := h.app.DispatchTaskWorkbench(c.Request.Context(), c.Query("project_id"), c.Param("id"))
	if err != nil {
		respondManagerError(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, taskWorkbenchActionResponseDTO(resp))
}

func (h *ManagerHandlers) RetryTaskWorkbench(c *gin.Context) {
	resp, err := h.app.RetryTaskWorkbench(c.Request.Context(), c.Query("project_id"), c.Param("id"))
	if err != nil {
		respondManagerError(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, taskWorkbenchActionResponseDTO(resp))
}

func (h *ManagerHandlers) CancelTaskWorkbench(c *gin.Context) {
	resp, err := h.app.CancelTaskWorkbench(c.Request.Context(), c.Query("project_id"), c.Param("id"))
	if err != nil {
		respondManagerError(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, taskWorkbenchActionResponseDTO(resp))
}

func (h *ManagerHandlers) ReprioritizeTaskWorkbench(c *gin.Context) {
	var req managerTaskWorkbenchReprioritizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Priority < 1 {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "priority must be >= 1"})
		return
	}

	resp, err := h.app.ReprioritizeTaskWorkbench(c.Request.Context(), c.Query("project_id"), c.Param("id"), req.Priority)
	if err != nil {
		respondManagerError(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, taskWorkbenchActionResponseDTO(resp))
}

func respondManagerError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, manageragent.ErrTaskActionNotFound):
		c.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, manageragent.ErrTaskActionConflict):
		c.JSON(nethttp.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, manageragent.ErrArtifactWorkbenchNotFound):
		c.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, manageragent.ErrArtifactWorkbenchConflict):
		c.JSON(nethttp.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, command.ErrApprovalActionNotFound):
		c.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, command.ErrApprovalActionConflict):
		c.JSON(nethttp.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, sql.ErrNoRows):
		c.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
	case isManagerClientError(err):
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

func respondManagerArtifactContentError(c *gin.Context, err error) {
	if errors.Is(err, os.ErrNotExist) {
		c.JSON(nethttp.StatusGone, gin.H{"error": err.Error()})
		return
	}

	respondManagerError(c, err)
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

func approvalActionResponseDTO(resp manageragent.ApprovalWorkbenchActionResponse) managerApprovalWorkbenchActionResponse {
	return managerApprovalWorkbenchActionResponse{
		ApprovalID:      resp.ApprovalID,
		ApprovalState:   resp.ApprovalState,
		RejectionReason: resp.RejectionReason,
		TaskID:          resp.TaskID,
		TaskState:       resp.TaskState,
		RunID:           resp.RunID,
		RunState:        resp.RunState,
	}
}

func taskWorkbenchActionResponseDTO(resp manageragent.TaskWorkbenchActionResponse) managerTaskWorkbenchActionResponse {
	return managerTaskWorkbenchActionResponse{
		TaskID:              resp.TaskID,
		TaskState:           resp.TaskState,
		LatestRunID:         resp.LatestRunID,
		LatestRunState:      resp.LatestRunState,
		LatestApprovalID:    resp.LatestApprovalID,
		LatestApprovalState: resp.LatestApprovalState,
		RefreshRequired:     resp.RefreshRequired,
		Message:             resp.Message,
	}
}

func artifactWorkbenchResponseDTO(view manageragent.ArtifactWorkbenchView) managerArtifactWorkbenchResponse {
	resp := managerArtifactWorkbenchResponse{
		ArtifactID:       view.ArtifactID,
		RunID:            view.RunID,
		TaskID:           view.TaskID,
		ProjectID:        view.ProjectID,
		ModuleID:         view.ModuleID,
		Kind:             view.Kind,
		Summary:          view.Summary,
		Path:             view.Path,
		ContentType:      view.ContentType,
		Preview:          view.Preview,
		PreviewTruncated: view.PreviewTruncated,
		RunWorkbenchURL:  view.RunWorkbenchURL,
		RawContentURL:    view.RawContentURL,
		Siblings:         make([]managerArtifactWorkbenchSiblingResponse, 0, len(view.Siblings)),
	}
	for _, sibling := range view.Siblings {
		resp.Siblings = append(resp.Siblings, managerArtifactWorkbenchSiblingResponse{
			ArtifactID: sibling.ArtifactID,
			Kind:       sibling.Kind,
			Summary:    sibling.Summary,
			Selected:   sibling.Selected,
		})
	}
	return resp
}

func artifactContentDisposition(artifactPath, contentType string) string {
	dispositionType := "attachment"
	if safeInlineArtifactContentType(contentType) {
		dispositionType = "inline"
	}

	filename := path.Base(strings.ReplaceAll(artifactPath, "\\", "/"))
	if filename == "." || filename == "/" || filename == "" {
		filename = "artifact"
	}

	return mime.FormatMediaType(dispositionType, map[string]string{"filename": filename})
}

func safeInlineArtifactContentType(contentType string) bool {
	return query.ArtifactWorkbenchAllowsInlineRawContent(contentType)
}

func runWorkbenchURL(runID string) string {
	if runID == "" {
		return ""
	}
	return "/board/runs/workbench?run_id=" + runID
}
