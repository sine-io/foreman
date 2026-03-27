package task

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTaskAllowsReadyToRunningPathThroughLease(t *testing.T) {
	task := NewTask(
		"task-1",
		"module-1",
		TaskTypeWrite,
		"Add SQLite store",
		"repo:project-1",
	)

	require.True(t, task.CanTransition(TaskStateLeased))
}
