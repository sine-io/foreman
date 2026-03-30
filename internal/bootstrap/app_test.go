package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	stdhttp "net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sine-io/foreman/internal/app/command"
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
