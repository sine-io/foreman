package artifactfs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArtifactStoreReadPreviewBoundsTextArtifacts(t *testing.T) {
	root := t.TempDir()
	store := New(root)

	fullPath := filepath.Join(root, "tasks", "task-1", "assistant_summary.txt")
	require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
	require.NoError(t, os.WriteFile(fullPath, []byte("abcdefghijklmnopqrstuvwxyz"), 0o644))

	preview, truncated, err := store.ReadPreview(fullPath, 8)
	require.NoError(t, err)
	require.Equal(t, "abcdefgh", preview)
	require.True(t, truncated)
}

func TestArtifactStoreResolveDisplayPathStripsArtifactRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "runtime-artifacts")
	store := New(root)

	displayPath, err := store.ResolveDisplayPath(filepath.Join(root, "tasks", "task-1", "assistant_summary.txt"))
	require.NoError(t, err)
	require.Equal(t, "tasks/task-1/assistant_summary.txt", displayPath)
}
