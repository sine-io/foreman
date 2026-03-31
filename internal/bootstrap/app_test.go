package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	stdhttp "net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sine-io/foreman/internal/app/command"
	"github.com/sine-io/foreman/internal/ports"
	"github.com/stretchr/testify/require"
)

func TestServeExposesBoardFlowFromOpenClawCommand(t *testing.T) {
	cfg := testConfig(t)
	appIface, err := BuildApp(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- appIface.Serve(ctx)
	}()
	waitForHTTP(t, cfg.HTTPAddr)

	resp, err := stdhttp.Post(
		"http://"+cfg.HTTPAddr+"/gateways/openclaw/command",
		"application/json",
		strings.NewReader(`{"session_id":"oc-1","action":"create_task","summary":"Bootstrap board"}`),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	require.Equal(t, stdhttp.StatusOK, resp.StatusCode)

	req, err := stdhttp.NewRequest(stdhttp.MethodGet, "http://"+cfg.HTTPAddr+"/board/tasks?project_id=demo", nil)
	require.NoError(t, err)
	boardResp, err := stdhttp.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = boardResp.Body.Close() })
	require.Equal(t, stdhttp.StatusOK, boardResp.StatusCode)

	body, err := io.ReadAll(boardResp.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "Bootstrap board")

	cancel()
	require.NoError(t, <-errCh)
}

func TestServeRoutesRiskyActionsIntoApprovalQueue(t *testing.T) {
	cfg := testConfig(t)
	appIface, err := BuildApp(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- appIface.Serve(ctx)
	}()
	waitForHTTP(t, cfg.HTTPAddr)

	resp, err := stdhttp.Post(
		"http://"+cfg.HTTPAddr+"/gateways/openclaw/command",
		"application/json",
		strings.NewReader(`{"session_id":"oc-2","action":"create_task","summary":"git push origin main"}`),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	require.Equal(t, stdhttp.StatusOK, resp.StatusCode)

	var payload map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
	require.Equal(t, "approval_needed", payload["kind"])

	boardResp, err := stdhttp.Get("http://" + cfg.HTTPAddr + "/board/tasks?project_id=demo")
	require.NoError(t, err)
	t.Cleanup(func() { _ = boardResp.Body.Close() })
	require.Equal(t, stdhttp.StatusOK, boardResp.StatusCode)

	body, err := io.ReadAll(boardResp.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "Waiting Approval")
	require.Contains(t, string(body), "git push origin main")

	cancel()
	require.NoError(t, <-errCh)
}

func TestServeExposesManagerCommandAPI(t *testing.T) {
	cfg := testConfig(t)
	appIface, err := BuildApp(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- appIface.Serve(ctx)
	}()
	waitForHTTP(t, cfg.HTTPAddr)

	resp, err := stdhttp.Post(
		"http://"+cfg.HTTPAddr+"/api/manager/commands",
		"application/json",
		strings.NewReader(`{"kind":"create_task","summary":"Bootstrap board"}`),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	require.Equal(t, stdhttp.StatusOK, resp.StatusCode)

	var payload map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
	require.Equal(t, "completion", payload["kind"])

	cancel()
	require.NoError(t, <-errCh)
}

func TestServeExposesManagerApprovalWorkbenchAPI(t *testing.T) {
	cfg := testConfig(t)
	appIface, err := BuildApp(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- appIface.Serve(ctx)
	}()
	waitForHTTP(t, cfg.HTTPAddr)

	createResp, err := stdhttp.Post(
		"http://"+cfg.HTTPAddr+"/api/manager/commands",
		"application/json",
		strings.NewReader(`{"kind":"create_task","summary":"git push origin main"}`),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = createResp.Body.Close() })
	require.Equal(t, stdhttp.StatusOK, createResp.StatusCode)

	var created map[string]any
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))
	require.Equal(t, "approval_needed", created["kind"])

	queueResp, err := stdhttp.Get("http://" + cfg.HTTPAddr + "/api/manager/projects/demo/approvals")
	require.NoError(t, err)
	t.Cleanup(func() { _ = queueResp.Body.Close() })
	require.Equal(t, stdhttp.StatusOK, queueResp.StatusCode)

	var queuePayload struct {
		Items []struct {
			ApprovalID string `json:"approval_id"`
		} `json:"items"`
	}
	require.NoError(t, json.NewDecoder(queueResp.Body).Decode(&queuePayload))
	require.Len(t, queuePayload.Items, 1)
	require.NotEmpty(t, queuePayload.Items[0].ApprovalID)

	detailResp, err := stdhttp.Get("http://" + cfg.HTTPAddr + "/api/manager/approvals/" + queuePayload.Items[0].ApprovalID)
	require.NoError(t, err)
	t.Cleanup(func() { _ = detailResp.Body.Close() })
	require.Equal(t, stdhttp.StatusOK, detailResp.StatusCode)

	var detailPayload map[string]any
	require.NoError(t, json.NewDecoder(detailResp.Body).Decode(&detailPayload))
	require.Equal(t, "pending", detailPayload["approval_state"])

	approveResp, err := stdhttp.Post(
		"http://"+cfg.HTTPAddr+"/api/manager/approvals/"+queuePayload.Items[0].ApprovalID+"/approve",
		"application/json",
		strings.NewReader(`{}`),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = approveResp.Body.Close() })
	require.Equal(t, stdhttp.StatusOK, approveResp.StatusCode)

	var actionPayload map[string]any
	require.NoError(t, json.NewDecoder(approveResp.Body).Decode(&actionPayload))
	require.Equal(t, "approved", actionPayload["approval_state"])

	cancel()
	require.NoError(t, <-errCh)
}

func TestServeKeepsRejectedApprovalDirectlyViewableAfterProcessing(t *testing.T) {
	cfg := testConfig(t)
	appIface, err := BuildApp(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- appIface.Serve(ctx)
	}()
	waitForHTTP(t, cfg.HTTPAddr)

	createResp, err := stdhttp.Post(
		"http://"+cfg.HTTPAddr+"/api/manager/commands",
		"application/json",
		strings.NewReader(`{"kind":"create_task","summary":"git push origin main"}`),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = createResp.Body.Close() })
	require.Equal(t, stdhttp.StatusOK, createResp.StatusCode)

	queueResp, err := stdhttp.Get("http://" + cfg.HTTPAddr + "/api/manager/projects/demo/approvals")
	require.NoError(t, err)
	t.Cleanup(func() { _ = queueResp.Body.Close() })
	require.Equal(t, stdhttp.StatusOK, queueResp.StatusCode)

	var queuePayload struct {
		Items []struct {
			ApprovalID string `json:"approval_id"`
		} `json:"items"`
	}
	require.NoError(t, json.NewDecoder(queueResp.Body).Decode(&queuePayload))
	require.Len(t, queuePayload.Items, 1)
	approvalID := queuePayload.Items[0].ApprovalID

	rejectResp, err := stdhttp.Post(
		"http://"+cfg.HTTPAddr+"/api/manager/approvals/"+approvalID+"/reject",
		"application/json",
		strings.NewReader(`{"rejection_reason":"missing rollback plan"}`),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rejectResp.Body.Close() })
	require.Equal(t, stdhttp.StatusOK, rejectResp.StatusCode)

	var rejectPayload map[string]any
	require.NoError(t, json.NewDecoder(rejectResp.Body).Decode(&rejectPayload))
	require.Equal(t, "rejected", rejectPayload["approval_state"])
	require.Equal(t, "missing rollback plan", rejectPayload["rejection_reason"])

	queueResp, err = stdhttp.Get("http://" + cfg.HTTPAddr + "/api/manager/projects/demo/approvals")
	require.NoError(t, err)
	t.Cleanup(func() { _ = queueResp.Body.Close() })
	require.Equal(t, stdhttp.StatusOK, queueResp.StatusCode)
	queuePayload.Items = nil
	require.NoError(t, json.NewDecoder(queueResp.Body).Decode(&queuePayload))
	require.Empty(t, queuePayload.Items)

	detailResp, err := stdhttp.Get("http://" + cfg.HTTPAddr + "/api/manager/approvals/" + approvalID)
	require.NoError(t, err)
	t.Cleanup(func() { _ = detailResp.Body.Close() })
	require.Equal(t, stdhttp.StatusOK, detailResp.StatusCode)

	var detailPayload map[string]any
	require.NoError(t, json.NewDecoder(detailResp.Body).Decode(&detailPayload))
	require.Equal(t, "rejected", detailPayload["approval_state"])
	require.Equal(t, "missing rollback plan", detailPayload["rejection_reason"])

	cancel()
	require.NoError(t, <-errCh)
}

func TestServeReturns404ForMissingApprovalWorkbenchDetail(t *testing.T) {
	cfg := testConfig(t)
	appIface, err := BuildApp(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- appIface.Serve(ctx)
	}()
	waitForHTTP(t, cfg.HTTPAddr)

	resp, err := stdhttp.Get("http://" + cfg.HTTPAddr + "/api/manager/approvals/missing-approval")
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	require.Equal(t, stdhttp.StatusNotFound, resp.StatusCode)

	cancel()
	require.NoError(t, <-errCh)
}

func TestServeExposesManagerTaskWorkbenchAPI(t *testing.T) {
	cfg := testConfig(t)
	appIface, err := BuildApp(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- appIface.Serve(ctx)
	}()
	waitForHTTP(t, cfg.HTTPAddr)

	_, err = appIface.CreateProject(command.CreateProjectCommand{
		ID:       defaultProjectID,
		Name:     "Demo Project",
		RepoRoot: cfg.RuntimeRoot,
	})
	require.NoError(t, err)

	_, err = appIface.CreateModule(command.CreateModuleCommand{
		ID:          defaultModuleID,
		ProjectID:   defaultProjectID,
		Name:        "Inbox",
		Description: "Task workbench test module",
	})
	require.NoError(t, err)

	taskDTO, err := appIface.CreateTask(command.CreateTaskCommand{
		ModuleID:   defaultModuleID,
		Title:      "Inspect repo state",
		TaskType:   "write",
		WriteScope: "repo:demo",
		Acceptance: "Inspect repo state",
		Priority:   10,
	})
	require.NoError(t, err)
	taskID := taskDTO.ID
	require.NotEmpty(t, taskID)

	workbenchResp, err := stdhttp.Get("http://" + cfg.HTTPAddr + "/api/manager/tasks/" + taskID + "/workbench?project_id=demo")
	require.NoError(t, err)
	t.Cleanup(func() { _ = workbenchResp.Body.Close() })
	require.Equal(t, stdhttp.StatusOK, workbenchResp.StatusCode)

	var workbenchPayload map[string]any
	require.NoError(t, json.NewDecoder(workbenchResp.Body).Decode(&workbenchPayload))
	require.Equal(t, taskID, workbenchPayload["task_id"])

	reprioritizeResp, err := stdhttp.Post(
		"http://"+cfg.HTTPAddr+"/api/manager/tasks/"+taskID+"/reprioritize?project_id=demo",
		"application/json",
		strings.NewReader(`{"priority":42}`),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = reprioritizeResp.Body.Close() })
	require.Equal(t, stdhttp.StatusOK, reprioritizeResp.StatusCode)

	var actionPayload map[string]any
	require.NoError(t, json.NewDecoder(reprioritizeResp.Body).Decode(&actionPayload))
	require.Equal(t, taskID, actionPayload["task_id"])
	require.Equal(t, "ready", actionPayload["task_state"])
	require.Equal(t, true, actionPayload["refresh_required"])

	cancel()
	require.NoError(t, <-errCh)
}

func TestServeExposesRunWorkbenchAPI(t *testing.T) {
	cfg := testConfig(t)
	appIface, err := BuildApp(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- appIface.Serve(ctx)
	}()
	waitForHTTP(t, cfg.HTTPAddr)

	resp, err := stdhttp.Post(
		"http://"+cfg.HTTPAddr+"/api/manager/commands",
		"application/json",
		strings.NewReader(`{"kind":"create_task","summary":"Inspect repo state"}`),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	require.Equal(t, stdhttp.StatusOK, resp.StatusCode)

	var created map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&created))
	taskID, _ := created["task_id"].(string)
	require.NotEmpty(t, taskID)

	taskWorkbenchResp, err := stdhttp.Get("http://" + cfg.HTTPAddr + "/api/manager/tasks/" + taskID + "/workbench?project_id=demo")
	require.NoError(t, err)
	t.Cleanup(func() { _ = taskWorkbenchResp.Body.Close() })
	require.Equal(t, stdhttp.StatusOK, taskWorkbenchResp.StatusCode)

	var taskWorkbenchPayload map[string]any
	require.NoError(t, json.NewDecoder(taskWorkbenchResp.Body).Decode(&taskWorkbenchPayload))
	runID, _ := taskWorkbenchPayload["latest_run_id"].(string)
	require.NotEmpty(t, runID)

	workbenchResp, err := stdhttp.Get("http://" + cfg.HTTPAddr + "/api/manager/runs/" + runID + "/workbench")
	require.NoError(t, err)
	t.Cleanup(func() { _ = workbenchResp.Body.Close() })
	require.Equal(t, stdhttp.StatusOK, workbenchResp.StatusCode)

	var workbenchPayload map[string]any
	require.NoError(t, json.NewDecoder(workbenchResp.Body).Decode(&workbenchPayload))
	require.Equal(t, runID, workbenchPayload["run_id"])
	require.NotEmpty(t, workbenchPayload["task_workbench_url"])

	redirectClient := &stdhttp.Client{
		CheckRedirect: func(req *stdhttp.Request, via []*stdhttp.Request) error {
			return stdhttp.ErrUseLastResponse
		},
	}
	legacyResp, err := redirectClient.Get("http://" + cfg.HTTPAddr + "/board/runs/" + runID)
	require.NoError(t, err)
	t.Cleanup(func() { _ = legacyResp.Body.Close() })
	require.Contains(t, []int{stdhttp.StatusFound, stdhttp.StatusSeeOther}, legacyResp.StatusCode)
	require.Equal(t, "/board/runs/workbench?run_id="+runID, legacyResp.Header.Get("Location"))

	cancel()
	require.NoError(t, <-errCh)
}

func TestServeExposesArtifactWorkbenchAPI(t *testing.T) {
	cfg := testConfig(t)
	appIface, err := BuildApp(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- appIface.Serve(ctx)
	}()
	waitForHTTP(t, cfg.HTTPAddr)

	resp, err := stdhttp.Post(
		"http://"+cfg.HTTPAddr+"/api/manager/commands",
		"application/json",
		strings.NewReader(`{"kind":"create_task","summary":"Inspect repo state"}`),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	require.Equal(t, stdhttp.StatusOK, resp.StatusCode)

	var created struct {
		TaskID string `json:"task_id"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&created))
	require.NotEmpty(t, created.TaskID)

	taskWorkbenchResp, err := stdhttp.Get("http://" + cfg.HTTPAddr + "/api/manager/tasks/" + created.TaskID + "/workbench?project_id=demo")
	require.NoError(t, err)
	t.Cleanup(func() { _ = taskWorkbenchResp.Body.Close() })
	require.Equal(t, stdhttp.StatusOK, taskWorkbenchResp.StatusCode)

	var taskWorkbenchPayload struct {
		LatestRunID string `json:"latest_run_id"`
	}
	require.NoError(t, json.NewDecoder(taskWorkbenchResp.Body).Decode(&taskWorkbenchPayload))
	require.NotEmpty(t, taskWorkbenchPayload.LatestRunID)

	runWorkbenchResp, err := stdhttp.Get("http://" + cfg.HTTPAddr + "/api/manager/runs/" + taskWorkbenchPayload.LatestRunID + "/workbench")
	require.NoError(t, err)
	t.Cleanup(func() { _ = runWorkbenchResp.Body.Close() })
	require.Equal(t, stdhttp.StatusOK, runWorkbenchResp.StatusCode)

	var runWorkbenchPayload struct {
		Artifacts []struct {
			ID string `json:"id"`
		} `json:"artifacts"`
	}
	require.NoError(t, json.NewDecoder(runWorkbenchResp.Body).Decode(&runWorkbenchPayload))
	require.Len(t, runWorkbenchPayload.Artifacts, 1)
	artifactID := runWorkbenchPayload.Artifacts[0].ID
	require.NotEmpty(t, artifactID)

	artifactWorkbenchResp, err := stdhttp.Get("http://" + cfg.HTTPAddr + "/api/manager/artifacts/" + artifactID + "/workbench")
	require.NoError(t, err)
	t.Cleanup(func() { _ = artifactWorkbenchResp.Body.Close() })
	require.Equal(t, stdhttp.StatusOK, artifactWorkbenchResp.StatusCode)

	var artifactWorkbenchPayload struct {
		ArtifactID      string `json:"artifact_id"`
		RunID           string `json:"run_id"`
		RawContentURL   string `json:"raw_content_url"`
		RunWorkbenchURL string `json:"run_workbench_url"`
	}
	require.NoError(t, json.NewDecoder(artifactWorkbenchResp.Body).Decode(&artifactWorkbenchPayload))
	require.Equal(t, artifactID, artifactWorkbenchPayload.ArtifactID)
	require.Equal(t, taskWorkbenchPayload.LatestRunID, artifactWorkbenchPayload.RunID)
	require.Equal(t, "/api/manager/artifacts/"+artifactID+"/content", artifactWorkbenchPayload.RawContentURL)
	require.Equal(t, "/board/runs/workbench?run_id="+taskWorkbenchPayload.LatestRunID, artifactWorkbenchPayload.RunWorkbenchURL)

	contentResp, err := stdhttp.Get("http://" + cfg.HTTPAddr + "/api/manager/artifacts/" + artifactID + "/content")
	require.NoError(t, err)
	t.Cleanup(func() { _ = contentResp.Body.Close() })
	require.Equal(t, stdhttp.StatusOK, contentResp.StatusCode)
	require.Equal(t, "nosniff", contentResp.Header.Get("X-Content-Type-Options"))
	require.Equal(t, "text/plain; charset=utf-8", contentResp.Header.Get("Content-Type"))
	require.Contains(t, contentResp.Header.Get("Content-Disposition"), "inline;")

	cancel()
	require.NoError(t, <-errCh)
}

func TestServeArtifactContentPreservesImagePreviewContract(t *testing.T) {
	cfg := testConfig(t)
	appIface, err := BuildApp(cfg)
	require.NoError(t, err)

	appImpl, ok := appIface.(*app)
	require.True(t, ok)
	require.NoError(t, appImpl.ensureDefaultProject())
	require.NoError(t, appImpl.ensureDefaultModule())

	taskDTO, err := appIface.CreateTask(command.CreateTaskCommand{
		ModuleID:   defaultModuleID,
		Title:      "Inspect screenshot artifact",
		TaskType:   "write",
		WriteScope: "repo:demo",
		Acceptance: "Inspect screenshot artifact",
		Priority:   10,
	})
	require.NoError(t, err)

	pngBody := []byte{0x89, 0x50, 0x4e, 0x47}
	artifactPath := filepath.ToSlash(filepath.Join("tasks", taskDTO.ID, "preview.png"))
	fullArtifactPath := filepath.Join(cfg.ArtifactRoot, filepath.FromSlash(artifactPath))
	require.NoError(t, os.MkdirAll(filepath.Dir(fullArtifactPath), 0o755))
	require.NoError(t, os.WriteFile(fullArtifactPath, pngBody, 0o644))

	require.NoError(t, appImpl.runs.Save(ports.Run{
		ID:         "run-image-preview",
		TaskID:     taskDTO.ID,
		RunnerKind: "codex",
		State:      "completed",
		CreatedAt:  "2026-03-31T09:00:00.000000000Z",
	}))

	artifactID, err := appImpl.artifacts.Create(taskDTO.ID, "run-image-preview", "screenshot", artifactPath)
	require.NoError(t, err)
	_, err = appImpl.db.Exec(`update artifacts set path = '' where id = ?`, artifactID)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- appIface.Serve(ctx)
	}()
	waitForHTTP(t, cfg.HTTPAddr)

	workbenchResp, err := stdhttp.Get("http://" + cfg.HTTPAddr + "/api/manager/artifacts/" + artifactID + "/workbench")
	require.NoError(t, err)
	t.Cleanup(func() { _ = workbenchResp.Body.Close() })
	require.Equal(t, stdhttp.StatusOK, workbenchResp.StatusCode)

	var workbenchPayload struct {
		ContentType   string `json:"content_type"`
		Preview       string `json:"preview"`
		RawContentURL string `json:"raw_content_url"`
	}
	require.NoError(t, json.NewDecoder(workbenchResp.Body).Decode(&workbenchPayload))
	require.Equal(t, "image/png", workbenchPayload.ContentType)
	require.Empty(t, workbenchPayload.Preview)
	require.Equal(t, "/api/manager/artifacts/"+artifactID+"/content", workbenchPayload.RawContentURL)

	contentResp, err := stdhttp.Get("http://" + cfg.HTTPAddr + workbenchPayload.RawContentURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = contentResp.Body.Close() })
	require.Equal(t, stdhttp.StatusOK, contentResp.StatusCode)
	require.Equal(t, "nosniff", contentResp.Header.Get("X-Content-Type-Options"))
	require.Equal(t, workbenchPayload.ContentType, contentResp.Header.Get("Content-Type"))
	require.Contains(t, contentResp.Header.Get("Content-Disposition"), "inline;")
	body, err := io.ReadAll(contentResp.Body)
	require.NoError(t, err)
	require.Equal(t, pngBody, body)

	cancel()
	require.NoError(t, <-errCh)
}

func TestServeArtifactCompareExposesLiveCompareRoute(t *testing.T) {
	cfg := testConfig(t)
	appIface, err := BuildApp(cfg)
	require.NoError(t, err)

	appImpl, ok := appIface.(*app)
	require.True(t, ok)
	require.NoError(t, appImpl.ensureDefaultProject())
	require.NoError(t, appImpl.ensureDefaultModule())

	taskDTO, err := appIface.CreateTask(command.CreateTaskCommand{
		ModuleID:   defaultModuleID,
		Title:      "Compare artifact history",
		TaskType:   "write",
		WriteScope: "repo:demo",
		Acceptance: "Compare artifact history",
		Priority:   10,
	})
	require.NoError(t, err)

	previousPath := filepath.ToSlash(filepath.Join("tasks", taskDTO.ID, "assistant_summary-previous.txt"))
	currentPath := filepath.ToSlash(filepath.Join("tasks", taskDTO.ID, "assistant_summary-current.txt"))
	require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(cfg.ArtifactRoot, filepath.FromSlash(previousPath))), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(cfg.ArtifactRoot, filepath.FromSlash(previousPath)), []byte("previous artifact\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(cfg.ArtifactRoot, filepath.FromSlash(currentPath)), []byte("current artifact\n"), 0o644))

	require.NoError(t, appImpl.runs.Save(ports.Run{
		ID:         "run-compare-1",
		TaskID:     taskDTO.ID,
		RunnerKind: "codex",
		State:      "completed",
		CreatedAt:  "2026-04-01T09:00:00.000000000Z",
	}))
	previousID, err := appImpl.artifacts.Create(taskDTO.ID, "run-compare-1", "assistant_summary", previousPath)
	require.NoError(t, err)

	require.NoError(t, appImpl.runs.Save(ports.Run{
		ID:         "run-compare-2",
		TaskID:     taskDTO.ID,
		RunnerKind: "codex",
		State:      "completed",
		CreatedAt:  "2026-04-01T10:00:00.000000000Z",
	}))
	currentID, err := appImpl.artifacts.Create(taskDTO.ID, "run-compare-2", "assistant_summary", currentPath)
	require.NoError(t, err)
	require.NotEqual(t, previousID, currentID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- appIface.Serve(ctx)
	}()
	waitForHTTP(t, cfg.HTTPAddr)

	compareResp, err := stdhttp.Get("http://" + cfg.HTTPAddr + "/api/manager/artifacts/" + currentID + "/compare")
	require.NoError(t, err)
	t.Cleanup(func() { _ = compareResp.Body.Close() })
	require.Equal(t, stdhttp.StatusOK, compareResp.StatusCode)

	var comparePayload struct {
		Status   string `json:"status"`
		Current  struct {
			ArtifactID string `json:"artifact_id"`
			RunID      string `json:"run_id"`
			TaskID     string `json:"task_id"`
			Kind       string `json:"kind"`
		} `json:"current"`
		Previous *struct {
			ArtifactID string `json:"artifact_id"`
			RunID      string `json:"run_id"`
		} `json:"previous"`
		Diff *struct {
			Format  string `json:"format"`
			Content string `json:"content"`
		} `json:"diff"`
		Limits struct {
			MaxCompareBytes int `json:"max_compare_bytes"`
		} `json:"limits"`
		Navigation struct {
			CurrentWorkbenchURL  string `json:"current_workbench_url"`
			PreviousWorkbenchURL string `json:"previous_workbench_url"`
			BackToRunURL         string `json:"back_to_run_url"`
		} `json:"navigation"`
	}
	require.NoError(t, json.NewDecoder(compareResp.Body).Decode(&comparePayload))
	require.Equal(t, "ready", comparePayload.Status)
	require.Equal(t, currentID, comparePayload.Current.ArtifactID)
	require.Equal(t, "run-compare-2", comparePayload.Current.RunID)
	require.Equal(t, taskDTO.ID, comparePayload.Current.TaskID)
	require.Equal(t, "assistant_summary", comparePayload.Current.Kind)
	require.NotNil(t, comparePayload.Previous)
	require.Equal(t, previousID, comparePayload.Previous.ArtifactID)
	require.Equal(t, "run-compare-1", comparePayload.Previous.RunID)
	require.NotNil(t, comparePayload.Diff)
	require.Equal(t, "text/unified-diff", comparePayload.Diff.Format)
	require.Contains(t, comparePayload.Diff.Content, "previous:"+previousID)
	require.Contains(t, comparePayload.Diff.Content, "current:"+currentID)
	require.Equal(t, 64*1024, comparePayload.Limits.MaxCompareBytes)
	require.Equal(t, "/board/artifacts/workbench?artifact_id="+currentID, comparePayload.Navigation.CurrentWorkbenchURL)
	require.Equal(t, "/board/artifacts/workbench?artifact_id="+previousID, comparePayload.Navigation.PreviousWorkbenchURL)
	require.Equal(t, "/board/runs/workbench?run_id=run-compare-2", comparePayload.Navigation.BackToRunURL)

	cancel()
	require.NoError(t, <-errCh)
}

func testConfig(t *testing.T) Config {
	t.Helper()

	runtimeRoot := t.TempDir()
	writeFakeCodex(t, runtimeRoot)
	t.Setenv("PATH", runtimeRoot+string(os.PathListSeparator)+os.Getenv("PATH"))
	addr := reserveLocalAddr(t)

	return Config{
		RuntimeRoot:  runtimeRoot,
		DBPath:       runtimeRoot + "/foreman.db",
		ArtifactRoot: runtimeRoot + "/artifacts",
		HTTPAddr:     addr,
	}
}

func writeFakeCodex(t *testing.T, dir string) {
	t.Helper()

	path := dir + "/codex"
	content := "#!/bin/sh\nexit 0\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o755))
}

func reserveLocalAddr(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = l.Close() }()

	return l.Addr().String()
}

func waitForHTTP(t *testing.T, addr string) {
	t.Helper()

	client := &stdhttp.Client{Timeout: 200 * time.Millisecond}
	deadline := time.Now().Add(5 * time.Second)
	url := fmt.Sprintf("http://%s/board/tasks?project_id=demo", addr)

	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			return
		}

		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("server at %s did not become ready", addr)
}
