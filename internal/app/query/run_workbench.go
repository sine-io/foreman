package query

import (
	"net/url"
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
		PrimarySummary:     runWorkbenchPrimarySummary(row.RunID, row.RunState, row.Artifacts),
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
			view.ArtifactTargetURLs[artifact.ID] = runWorkbenchArtifactTargetURL(artifact)
		}
	}

	return view, nil
}

func runWorkbenchPrimarySummary(runID, runState string, artifacts []ports.ArtifactRecord) string {
	summaryArtifacts := runWorkbenchSummaryArtifacts(runID, artifacts)
	if summary := assistantArtifactSummary(summaryArtifacts); summary != "" {
		return summary
	}

	parts := make([]string, 0, len(summaryArtifacts)+1)
	if runState != "" {
		parts = append(parts, runState)
	}
	for _, artifact := range summaryArtifacts {
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

func runWorkbenchSummaryArtifacts(runID string, artifacts []ports.ArtifactRecord) []ports.ArtifactRecord {
	if runID == "" {
		return artifacts
	}

	sameRun := make([]ports.ArtifactRecord, 0, len(artifacts))
	legacy := make([]ports.ArtifactRecord, 0, len(artifacts))
	for _, artifact := range artifacts {
		switch artifact.RunID {
		case runID:
			sameRun = append(sameRun, artifact)
		case "":
			legacy = append(legacy, artifact)
		}
	}

	if len(sameRun) > 0 {
		return append(sameRun, legacy...)
	}
	return legacy
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

func runWorkbenchArtifactTargetURL(artifact ports.ArtifactRecord) string {
	if artifact.RunID != "" {
		return artifactWorkbenchURL(artifact.ID)
	}
	artifactID := artifact.ID
	return "#artifact-" + artifactID
}

func taskWorkbenchURL(projectID, taskID string) string {
	if projectID == "" || taskID == "" {
		return ""
	}
	return "/board/tasks/workbench?" + url.Values{
		"project_id": []string{projectID},
		"task_id":    []string{taskID},
	}.Encode()
}
