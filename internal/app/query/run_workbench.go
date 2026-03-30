package query

import (
	"strings"

	"github.com/sine-io/foreman/internal/ports"
)

type RunWorkbenchArtifact struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Path    string `json:"path"`
	Summary string `json:"summary"`
}

type RunWorkbenchView struct {
	RunID              string                 `json:"run_id"`
	TaskID             string                 `json:"task_id"`
	ProjectID          string                 `json:"project_id"`
	ModuleID           string                 `json:"module_id"`
	TaskSummary        string                 `json:"task_summary"`
	RunState           string                 `json:"run_state"`
	RunnerKind         string                 `json:"runner_kind"`
	PrimarySummary     string                 `json:"primary_summary"`
	RunCreatedAt       string                 `json:"run_created_at,omitempty"`
	TaskWorkbenchURL   string                 `json:"task_workbench_url"`
	ArtifactTargetURLs map[string]string      `json:"artifact_target_urls"`
	Artifacts          []RunWorkbenchArtifact `json:"artifacts"`
}

type RunWorkbenchRepository interface {
	GetRunWorkbench(runID string) (ports.RunWorkbenchRow, error)
}

type RunWorkbenchQuery struct {
	Repo RunWorkbenchRepository
}

func NewRunWorkbenchQuery(repo RunWorkbenchRepository) *RunWorkbenchQuery {
	return &RunWorkbenchQuery{Repo: repo}
}

func (q *RunWorkbenchQuery) Execute(runID string) (RunWorkbenchView, error) {
	row, err := q.Repo.GetRunWorkbench(runID)
	if err != nil {
		return RunWorkbenchView{}, err
	}

	view := RunWorkbenchView{
		RunID:              row.RunID,
		TaskID:             row.TaskID,
		ProjectID:          row.ProjectID,
		ModuleID:           row.ModuleID,
		TaskSummary:        row.TaskSummary,
		RunState:           row.RunState,
		RunnerKind:         row.RunnerKind,
		PrimarySummary:     runWorkbenchPrimarySummary(row.RunState, row.Artifacts),
		RunCreatedAt:       row.RunCreatedAt,
		TaskWorkbenchURL:   taskWorkbenchURL(row.ProjectID, row.TaskID),
		ArtifactTargetURLs: map[string]string{},
		Artifacts:          make([]RunWorkbenchArtifact, 0, len(row.Artifacts)),
	}

	for _, artifact := range row.Artifacts {
		view.Artifacts = append(view.Artifacts, RunWorkbenchArtifact{
			ID:      artifact.ID,
			Kind:    artifact.Kind,
			Path:    artifact.Path,
			Summary: artifact.Summary,
		})
		if artifact.ID != "" {
			view.ArtifactTargetURLs[artifact.ID] = runWorkbenchArtifactTargetURL(artifact.ID)
		}
	}

	return view, nil
}

func runWorkbenchPrimarySummary(runState string, artifacts []ports.ArtifactRecord) string {
	if summary := assistantArtifactSummary(artifacts); summary != "" {
		return summary
	}

	parts := make([]string, 0, len(artifacts)+1)
	if runState != "" {
		parts = append(parts, runState)
	}
	for _, artifact := range artifacts {
		if artifact.Summary == "" || artifact.Kind == "assistant_summary" || isRawLogArtifact(artifact.Kind) {
			continue
		}
		if artifact.Kind == "" {
			parts = append(parts, artifact.Summary)
			continue
		}
		parts = append(parts, artifact.Kind+": "+artifact.Summary)
	}

	return strings.Join(parts, " — ")
}

func assistantArtifactSummary(artifacts []ports.ArtifactRecord) string {
	for _, artifact := range artifacts {
		if artifact.Kind == "assistant_summary" && artifact.Summary != "" {
			return artifact.Summary
		}
	}
	return ""
}

func isRawLogArtifact(kind string) bool {
	return kind == "run_log" || strings.HasSuffix(kind, "_log")
}

func runWorkbenchArtifactTargetURL(artifactID string) string {
	return "#artifact-" + artifactID
}

func taskWorkbenchURL(projectID, taskID string) string {
	if projectID == "" || taskID == "" {
		return ""
	}
	return "/board/tasks/workbench?project_id=" + projectID + "&task_id=" + taskID
}
