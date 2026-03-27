package codex

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/sine-io/foreman/internal/ports"
)

type executor interface {
	Run(name string, args ...string) error
}

type shellExecutor struct{}

func (shellExecutor) Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

type Adapter struct {
	exec executor
}

func NewCodexAdapter(exec executor) *Adapter {
	if exec == nil {
		exec = shellExecutor{}
	}

	return &Adapter{exec: exec}
}

func (a *Adapter) Dispatch(req ports.RunRequest) (ports.Run, error) {
	if err := a.exec.Run("codex", "exec", "-C", ".", req.Command); err != nil {
		return ports.Run{}, err
	}

	return ports.Run{
		ID:                   fmt.Sprintf("run-%d", time.Now().UnixNano()),
		TaskID:               req.TaskID,
		RunnerKind:           "codex",
		State:                "running",
		AssistantSummaryPath: filepath.Join("artifacts", "tasks", req.TaskID, "assistant_summary.txt"),
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
