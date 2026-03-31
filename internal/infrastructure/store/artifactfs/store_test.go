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

func TestArtifactStoreResolveDisplayPathStripsSymlinkedArtifactRoot(t *testing.T) {
	parent := t.TempDir()
	realRoot := filepath.Join(parent, "real-runtime-artifacts")
	rootLink := filepath.Join(parent, "runtime-artifacts")
	store := New(rootLink)

	require.NoError(t, os.MkdirAll(filepath.Join(realRoot, "tasks", "task-1"), 0o755))
	require.NoError(t, os.Symlink(realRoot, rootLink))

	displayPath, err := store.ResolveDisplayPath(filepath.Join(rootLink, "tasks", "task-1", "assistant_summary.txt"))
	require.NoError(t, err)
	require.Equal(t, "tasks/task-1/assistant_summary.txt", displayPath)
	require.NotContains(t, displayPath, "..")
}

func TestArtifactStoreResolveDisplayPathAcceptsRealPathWhenRootUsesSymlinkAlias(t *testing.T) {
	parent := t.TempDir()
	realRoot := filepath.Join(parent, "real-runtime-artifacts")
	rootLink := filepath.Join(parent, "runtime-artifacts")
	store := New(rootLink)

	require.NoError(t, os.MkdirAll(filepath.Join(realRoot, "tasks", "task-1"), 0o755))
	require.NoError(t, os.Symlink(realRoot, rootLink))

	displayPath, err := store.ResolveDisplayPath(filepath.Join(realRoot, "tasks", "task-1", "assistant_summary.txt"))
	require.NoError(t, err)
	require.Equal(t, "tasks/task-1/assistant_summary.txt", displayPath)
}

func TestArtifactStoreReadPreviewAcceptsSymlinkAliasWhenRootUsesRealPath(t *testing.T) {
	parent := t.TempDir()
	realRoot := filepath.Join(parent, "real-runtime-artifacts")
	rootLink := filepath.Join(parent, "runtime-artifacts")
	store := New(realRoot)

	require.NoError(t, os.MkdirAll(filepath.Join(realRoot, "tasks", "task-1"), 0o755))
	require.NoError(t, os.Symlink(realRoot, rootLink))
	require.NoError(t, os.WriteFile(filepath.Join(realRoot, "tasks", "task-1", "assistant_summary.txt"), []byte("preview"), 0o644))

	preview, truncated, err := store.ReadPreview(filepath.Join(rootLink, "tasks", "task-1", "assistant_summary.txt"), 16)
	require.NoError(t, err)
	require.Equal(t, "preview", preview)
	require.False(t, truncated)
}

func TestArtifactStoreReadPreviewRejectsSymlinkEscape(t *testing.T) {
	root := filepath.Join(t.TempDir(), "runtime-artifacts")
	outsideDir := t.TempDir()
	store := New(root)

	require.NoError(t, os.MkdirAll(filepath.Join(root, "tasks", "task-1"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(outsideDir, "outside.txt"), []byte("outside"), 0o644))
	require.NoError(t, os.Symlink(filepath.Join(outsideDir, "outside.txt"), filepath.Join(root, "tasks", "task-1", "escape.txt")))

	_, _, err := store.ReadPreview(filepath.Join(root, "tasks", "task-1", "escape.txt"), 8)
	require.Error(t, err)
	require.Contains(t, err.Error(), "escapes root")
}
