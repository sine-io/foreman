package codex

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sine-io/foreman/internal/ports"
	"github.com/stretchr/testify/require"
)

func TestDispatchWritableTaskStartsCodexRun(t *testing.T) {
	workdir := t.TempDir()
	artifactRoot := filepath.Join(workdir, "runtime-artifacts")
	exec := &fakeExecutor{output: "smoke-ok\n"}
	runner := NewCodexAdapter(exec, workdir, artifactRoot)

	run, err := runner.Dispatch(ports.RunRequest{
		TaskID:     "task-1",
		Command:    "Implement board query",
		WriteScope: "repo:project-1",
	})
	require.NoError(t, err)
	require.Equal(t, "codex", run.RunnerKind)
	require.Equal(t, "codex", exec.name)
	require.Equal(t, []string{"exec", "-C", workdir, "Implement board query"}, exec.args)
	require.NotEmpty(t, run.AssistantSummaryPath)
	require.Equal(t, "completed", run.State)

	require.Equal(t, filepath.Join(artifactRoot, "tasks", "task-1", "assistant_summary.txt"), run.AssistantSummaryPath)

	content, err := os.ReadFile(run.AssistantSummaryPath)
	require.NoError(t, err)
	require.Equal(t, "smoke-ok\n", string(content))
}

type fakeExecutor struct {
	name   string
	args   []string
	output string
}

func (f *fakeExecutor) Run(name string, args ...string) ([]byte, error) {
	f.name = name
	f.args = args
	return []byte(f.output), nil
}
