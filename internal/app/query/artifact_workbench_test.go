package query

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sine-io/foreman/internal/infrastructure/store/artifactfs"
	sqlitestore "github.com/sine-io/foreman/internal/infrastructure/store/sqlite"
	"github.com/sine-io/foreman/internal/ports"
	"github.com/stretchr/testify/require"
)

func TestArtifactWorkbenchReturnsLinkedArtifactView(t *testing.T) {
	harness := newArtifactWorkbenchHarness(t)
	harness.seedTask(t, "project-1", "module-1", "task-1")
	harness.saveRun(t, ports.Run{
		ID:         "run-1",
		TaskID:     "task-1",
		RunnerKind: "codex",
		State:      "completed",
	}, "2026-03-31T09:00:00.000000000Z")
	harness.writeArtifactFile(t, "tasks/task-1/assistant_summary.txt", "Artifact preview content")
	harness.saveArtifact(t, artifactSeed{
		ID:          "artifact-1",
		TaskID:      "task-1",
		RunID:       "run-1",
		Kind:        "assistant_summary",
		Path:        "tasks/task-1/assistant_summary.txt",
		StoragePath: "tasks/task-1/assistant_summary.txt",
		Summary:     "Assistant summary for the completed run",
		CreatedAt:   "2026-03-31T09:01:00.000000000Z",
	})
	harness.saveArtifact(t, artifactSeed{
		ID:          "artifact-2",
		TaskID:      "task-1",
		RunID:       "run-1",
		Kind:        "command_result",
		Path:        "tasks/task-1/command.txt",
		StoragePath: "tasks/task-1/command.txt",
		Summary:     "Command output summary",
		CreatedAt:   "2026-03-31T09:02:00.000000000Z",
	})

	view, err := harness.query.Execute("artifact-1")
	require.NoError(t, err)
	require.Equal(t, "artifact-1", view.ArtifactID)
	require.Equal(t, "run-1", view.RunID)
	require.Equal(t, "task-1", view.TaskID)
	require.Equal(t, "project-1", view.ProjectID)
	require.Equal(t, "module-1", view.ModuleID)
	require.Equal(t, "assistant_summary", view.Kind)
	require.Equal(t, "Assistant summary for the completed run", view.Summary)
	require.Equal(t, "tasks/task-1/assistant_summary.txt", view.Path)
	require.Equal(t, "text/plain; charset=utf-8", view.ContentType)
	require.Equal(t, "Artifact preview content", view.Preview)
	require.False(t, view.PreviewTruncated)
	require.Len(t, view.Siblings, 2)
	require.True(t, artifactWorkbenchSiblingByID(t, view.Siblings, "artifact-1").Selected)
	require.False(t, artifactWorkbenchSiblingByID(t, view.Siblings, "artifact-2").Selected)
}

func TestArtifactWorkbenchReturnsConflictForLegacyUnlinkedArtifact(t *testing.T) {
	harness := newArtifactWorkbenchHarness(t)
	harness.seedTask(t, "project-1", "module-1", "task-1")
	harness.saveArtifact(t, artifactSeed{
		ID:          "artifact-legacy",
		TaskID:      "task-1",
		Kind:        "assistant_summary",
		Path:        "tasks/task-1/legacy.txt",
		StoragePath: "tasks/task-1/legacy.txt",
		Summary:     "Legacy artifact without run linkage",
		CreatedAt:   "2026-03-31T09:01:00.000000000Z",
	})

	_, err := harness.query.Execute("artifact-legacy")
	require.ErrorIs(t, err, ports.ErrArtifactRunLinkageConflict)
}

func TestArtifactWorkbenchReturnsErrorForBrokenLinkage(t *testing.T) {
	harness := newArtifactWorkbenchHarness(t)
	harness.seedTask(t, "project-1", "module-1", "task-1")
	harness.seedTask(t, "project-1", "module-1", "task-2")
	harness.saveRun(t, ports.Run{
		ID:         "run-2",
		TaskID:     "task-2",
		RunnerKind: "codex",
		State:      "failed",
	}, "2026-03-31T09:00:00.000000000Z")
	harness.saveArtifact(t, artifactSeed{
		ID:          "artifact-broken",
		TaskID:      "task-1",
		RunID:       "run-2",
		Kind:        "assistant_summary",
		Path:        "tasks/task-1/broken.txt",
		StoragePath: "tasks/task-1/broken.txt",
		Summary:     "Broken linkage artifact",
		CreatedAt:   "2026-03-31T09:01:00.000000000Z",
	})

	_, err := harness.query.Execute("artifact-broken")
	require.ErrorIs(t, err, ports.ErrArtifactBrokenLinkage)
	require.NotErrorIs(t, err, sql.ErrNoRows)
}

func TestArtifactWorkbenchScopesSiblingsToSameRun(t *testing.T) {
	harness := newArtifactWorkbenchHarness(t)
	harness.seedTask(t, "project-1", "module-1", "task-1")
	harness.saveRun(t, ports.Run{
		ID:         "run-1",
		TaskID:     "task-1",
		RunnerKind: "codex",
		State:      "failed",
	}, "2026-03-31T09:00:00.000000000Z")
	harness.saveRun(t, ports.Run{
		ID:         "run-2",
		TaskID:     "task-1",
		RunnerKind: "codex",
		State:      "completed",
	}, "2026-03-31T10:00:00.000000000Z")
	harness.writeArtifactFile(t, "tasks/task-1/assistant_summary.txt", "preview")
	harness.saveArtifact(t, artifactSeed{
		ID:          "artifact-target",
		TaskID:      "task-1",
		RunID:       "run-1",
		Kind:        "assistant_summary",
		Path:        "tasks/task-1/assistant_summary.txt",
		StoragePath: "tasks/task-1/assistant_summary.txt",
		Summary:     "Failed run summary",
		CreatedAt:   "2026-03-31T09:01:00.000000000Z",
	})
	harness.saveArtifact(t, artifactSeed{
		ID:          "artifact-sibling",
		TaskID:      "task-1",
		RunID:       "run-1",
		Kind:        "command_result",
		Path:        "tasks/task-1/command.txt",
		StoragePath: "tasks/task-1/command.txt",
		Summary:     "Command output from the same run",
		CreatedAt:   "2026-03-31T09:02:00.000000000Z",
	})
	harness.saveArtifact(t, artifactSeed{
		ID:          "artifact-other-run",
		TaskID:      "task-1",
		RunID:       "run-2",
		Kind:        "assistant_summary",
		Path:        "tasks/task-1/newer.txt",
		StoragePath: "tasks/task-1/newer.txt",
		Summary:     "A different run summary",
		CreatedAt:   "2026-03-31T10:01:00.000000000Z",
	})

	view, err := harness.query.Execute("artifact-target")
	require.NoError(t, err)
	require.Len(t, view.Siblings, 2)
	require.Equal(t, []string{"artifact-sibling", "artifact-target"}, artifactWorkbenchSiblingIDs(view.Siblings))
}

func TestArtifactWorkbenchIncludesBoundedPreviewAndTruncationFlag(t *testing.T) {
	harness := newArtifactWorkbenchHarness(t)
	harness.seedTask(t, "project-1", "module-1", "task-1")
	harness.saveRun(t, ports.Run{
		ID:         "run-1",
		TaskID:     "task-1",
		RunnerKind: "codex",
		State:      "completed",
	}, "2026-03-31T09:00:00.000000000Z")
	body := strings.Repeat("a", 64*1024+128)
	harness.writeArtifactFile(t, "tasks/task-1/large.txt", body)
	harness.saveArtifact(t, artifactSeed{
		ID:          "artifact-large",
		TaskID:      "task-1",
		RunID:       "run-1",
		Kind:        "assistant_summary",
		Path:        "tasks/task-1/large.txt",
		StoragePath: "tasks/task-1/large.txt",
		Summary:     "Large preview artifact",
		CreatedAt:   "2026-03-31T09:01:00.000000000Z",
	})

	view, err := harness.query.Execute("artifact-large")
	require.NoError(t, err)
	require.Len(t, view.Preview, 64*1024)
	require.True(t, view.PreviewTruncated)
}

func TestArtifactWorkbenchFallsBackForNonTextArtifact(t *testing.T) {
	harness := newArtifactWorkbenchHarness(t)
	harness.seedTask(t, "project-1", "module-1", "task-1")
	harness.saveRun(t, ports.Run{
		ID:         "run-1",
		TaskID:     "task-1",
		RunnerKind: "codex",
		State:      "completed",
	}, "2026-03-31T09:00:00.000000000Z")
	harness.writeArtifactFile(t, "tasks/task-1/screenshot.png", string([]byte{0x89, 0x50, 0x4e, 0x47}))
	harness.saveArtifact(t, artifactSeed{
		ID:          "artifact-image",
		TaskID:      "task-1",
		RunID:       "run-1",
		Kind:        "screenshot",
		Path:        "tasks/task-1/screenshot.png",
		StoragePath: "tasks/task-1/screenshot.png",
		Summary:     "PNG screenshot artifact",
		CreatedAt:   "2026-03-31T09:01:00.000000000Z",
	})

	view, err := harness.query.Execute("artifact-image")
	require.NoError(t, err)
	require.Equal(t, "image/png", view.ContentType)
	require.Empty(t, view.Preview)
	require.False(t, view.PreviewTruncated)
	require.Equal(t, "/api/manager/artifacts/artifact-image/content", view.RawContentURL)
}

func TestArtifactWorkbenchIncludesRunAndRawContentURLs(t *testing.T) {
	harness := newArtifactWorkbenchHarness(t)
	harness.seedTask(t, "project-1", "module-1", "task-1")
	harness.saveRun(t, ports.Run{
		ID:         "run-1",
		TaskID:     "task-1",
		RunnerKind: "codex",
		State:      "completed",
	}, "2026-03-31T09:00:00.000000000Z")
	harness.writeArtifactFile(t, "tasks/task-1/assistant_summary.txt", "preview")
	harness.saveArtifact(t, artifactSeed{
		ID:          "artifact-1",
		TaskID:      "task-1",
		RunID:       "run-1",
		Kind:        "assistant_summary",
		Path:        "tasks/task-1/assistant_summary.txt",
		StoragePath: "tasks/task-1/assistant_summary.txt",
		Summary:     "Artifact with navigation URLs",
		CreatedAt:   "2026-03-31T09:01:00.000000000Z",
	})

	view, err := harness.query.Execute("artifact-1")
	require.NoError(t, err)
	require.Equal(t, "/board/runs/workbench?run_id=run-1", view.RunWorkbenchURL)
	require.Equal(t, "/api/manager/artifacts/artifact-1/content", view.RawContentURL)
}

type artifactWorkbenchHarness struct {
	db    *sql.DB
	root  string
	query *ArtifactWorkbenchQuery
}

func newArtifactWorkbenchHarness(t *testing.T) *artifactWorkbenchHarness {
	t.Helper()

	db, err := sqlitestore.Open(filepath.Join(t.TempDir(), "foreman.db"))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	root := t.TempDir()
	store := artifactfs.New(root)
	repo := sqlitestore.NewBoardQueryRepository(db)

	return &artifactWorkbenchHarness{
		db:    db,
		root:  root,
		query: NewArtifactWorkbenchQuery(repo, store),
	}
}

func (h *artifactWorkbenchHarness) seedTask(t *testing.T, projectID, moduleID, taskID string) {
	t.Helper()

	h.mustExec(t, `insert or ignore into projects (id, name, repo_root) values (?, ?, ?)`, projectID, projectID, "/tmp/"+projectID)
	h.mustExec(t, `insert or ignore into modules (id, project_id, name, board_state) values (?, ?, ?, ?)`, moduleID, projectID, moduleID, "planned")
	h.mustExec(t, `insert into tasks (id, module_id, summary, acceptance, priority, task_type, state, write_scope) values (?, ?, ?, ?, ?, ?, ?, ?)`,
		taskID,
		moduleID,
		taskID+" summary",
		taskID+" acceptance",
		10,
		"write",
		"completed",
		"repo:"+projectID,
	)
}

func (h *artifactWorkbenchHarness) saveRun(t *testing.T, run ports.Run, createdAt string) {
	t.Helper()
	h.mustExec(t, `insert into runs (id, task_id, runner_kind, state, created_at) values (?, ?, ?, ?, ?)`,
		run.ID,
		run.TaskID,
		run.RunnerKind,
		run.State,
		createdAt,
	)
}

type artifactSeed struct {
	ID          string
	TaskID      string
	RunID       string
	Kind        string
	Path        string
	StoragePath string
	Summary     string
	CreatedAt   string
}

func (h *artifactWorkbenchHarness) saveArtifact(t *testing.T, artifact artifactSeed) {
	t.Helper()

	var runID any
	if artifact.RunID != "" {
		runID = artifact.RunID
	}

	h.mustExec(t,
		`insert into artifacts (id, task_id, run_id, kind, path, storage_path, summary, created_at) values (?, ?, ?, ?, ?, ?, ?, ?)`,
		artifact.ID,
		artifact.TaskID,
		runID,
		artifact.Kind,
		artifact.Path,
		artifact.StoragePath,
		artifact.Summary,
		artifact.CreatedAt,
	)
}

func (h *artifactWorkbenchHarness) writeArtifactFile(t *testing.T, relativePath string, content string) {
	t.Helper()

	fullPath := filepath.Join(h.root, filepath.FromSlash(relativePath))
	require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
	require.NoError(t, os.WriteFile(fullPath, []byte(content), 0o644))
}

func (h *artifactWorkbenchHarness) mustExec(t *testing.T, query string, args ...any) {
	t.Helper()

	_, err := h.db.Exec(query, args...)
	require.NoError(t, err)
}

func artifactWorkbenchSiblingByID(t *testing.T, siblings []ArtifactWorkbenchSibling, artifactID string) ArtifactWorkbenchSibling {
	t.Helper()

	for _, sibling := range siblings {
		if sibling.ArtifactID == artifactID {
			return sibling
		}
	}
	t.Fatalf("missing sibling %s", artifactID)
	return ArtifactWorkbenchSibling{}
}

func artifactWorkbenchSiblingIDs(siblings []ArtifactWorkbenchSibling) []string {
	ids := make([]string, 0, len(siblings))
	for _, sibling := range siblings {
		ids = append(ids, sibling.ArtifactID)
	}
	return ids
}
