package codex

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/sine-io/foreman/internal/ports"
)

type executor interface {
	Run(name string, args ...string) ([]byte, error)
}

type shellExecutor struct{}

func (shellExecutor) Run(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.CombinedOutput()
}

type Adapter struct {
	exec         executor
	workdir      string
	artifactRoot string
}

func NewCodexAdapter(exec executor, workdir, artifactRoot string) *Adapter {
	if exec == nil {
		exec = shellExecutor{}
	}

	if workdir == "" {
		workdir = "."
	}
	if artifactRoot == "" {
		artifactRoot = filepath.Join(workdir, "artifacts")
	}

	return &Adapter{
		exec:         exec,
		workdir:      workdir,
		artifactRoot: artifactRoot,
	}
}

func (a *Adapter) Dispatch(req ports.RunRequest) (ports.Run, error) {
	output, err := a.exec.Run("codex", "exec", "-C", a.workdir, req.Command)
	if err != nil {
		return ports.Run{}, err
	}

	relativeSummaryPath := filepath.Join("tasks", req.TaskID, "assistant_summary.txt")
	fullSummaryPath := filepath.Join(a.artifactRoot, relativeSummaryPath)
	if err := os.MkdirAll(filepath.Dir(fullSummaryPath), 0o755); err != nil {
		return ports.Run{}, err
	}
	if err := os.WriteFile(fullSummaryPath, output, 0o644); err != nil {
		return ports.Run{}, err
	}

	return ports.Run{
		ID:                   fmt.Sprintf("run-%d", time.Now().UnixNano()),
		TaskID:               req.TaskID,
		RunnerKind:           "codex",
		State:                "completed",
		AssistantSummaryPath: fullSummaryPath,
	}, nil
}

func (a *Adapter) Observe(runID string) (ports.Run, error) {
	return ports.Run{
		ID:         runID,
		RunnerKind: "codex",
		State:      "running",
	}, nil
}

func (a *Adapter) Stop(runID string) error {
	return nil
}
