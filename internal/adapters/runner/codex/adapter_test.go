package codex

import (
	"testing"

	"github.com/sine-io/foreman/internal/ports"
	"github.com/stretchr/testify/require"
)

func TestDispatchWritableTaskStartsCodexRun(t *testing.T) {
	exec := &fakeExecutor{}
	runner := NewCodexAdapter(exec)

	run, err := runner.Dispatch(ports.RunRequest{
		TaskID:     "task-1",
		Command:    "Implement board query",
		WriteScope: "repo:project-1",
	})
	require.NoError(t, err)
	require.Equal(t, "codex", run.RunnerKind)
	require.Equal(t, "codex", exec.name)
	require.Equal(t, []string{"exec", "-C", ".", "Implement board query"}, exec.args)
	require.NotEmpty(t, run.AssistantSummaryPath)
}

type fakeExecutor struct {
	name string
	args []string
}

func (f *fakeExecutor) Run(name string, args ...string) error {
	f.name = name
	f.args = args
	return nil
}
