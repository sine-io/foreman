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
