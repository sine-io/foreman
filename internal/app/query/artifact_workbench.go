package query

import (
	"mime"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/sine-io/foreman/internal/ports"
)

const artifactWorkbenchPreviewMaxBytes = 64 * 1024

type ArtifactWorkbenchSibling struct {
	ArtifactID string `json:"artifact_id"`
	Kind       string `json:"kind"`
	Summary    string `json:"summary"`
	Selected   bool   `json:"selected"`
}

type ArtifactWorkbenchView struct {
	ArtifactID       string                     `json:"artifact_id"`
	RunID            string                     `json:"run_id"`
	TaskID           string                     `json:"task_id"`
	ProjectID        string                     `json:"project_id"`
	ModuleID         string                     `json:"module_id"`
	Kind             string                     `json:"kind"`
	Summary          string                     `json:"summary"`
	Path             string                     `json:"path"`
	ContentType      string                     `json:"content_type,omitempty"`
	Preview          string                     `json:"preview,omitempty"`
	PreviewTruncated bool                       `json:"preview_truncated"`
	RunWorkbenchURL  string                     `json:"run_workbench_url"`
	RawContentURL    string                     `json:"raw_content_url"`
	Siblings         []ArtifactWorkbenchSibling `json:"siblings"`
}

type ArtifactWorkbenchRepository interface {
	GetArtifactWorkbench(artifactID string) (ports.ArtifactWorkbenchRow, error)
}

type ArtifactWorkbenchQuery struct {
	Repo  ArtifactWorkbenchRepository
	Store ports.ArtifactStore
}

func NewArtifactWorkbenchQuery(repo ArtifactWorkbenchRepository, store ports.ArtifactStore) *ArtifactWorkbenchQuery {
	return &ArtifactWorkbenchQuery{Repo: repo, Store: store}
}

func (q *ArtifactWorkbenchQuery) Execute(artifactID string) (ArtifactWorkbenchView, error) {
	row, err := q.Repo.GetArtifactWorkbench(artifactID)
	if err != nil {
		return ArtifactWorkbenchView{}, err
	}

	contentType := artifactWorkbenchContentType(row.Kind, row.Path)
	view := ArtifactWorkbenchView{
		ArtifactID:      row.ArtifactID,
		RunID:           row.RunID,
		TaskID:          row.TaskID,
		ProjectID:       row.ProjectID,
		ModuleID:        row.ModuleID,
		Kind:            row.Kind,
		Summary:         row.Summary,
		Path:            row.Path,
		ContentType:     contentType,
		RunWorkbenchURL: runWorkbenchURL(row.RunID),
		RawContentURL:   artifactRawContentURL(row.ArtifactID),
		Siblings:        make([]ArtifactWorkbenchSibling, 0, len(row.Siblings)),
	}

	if artifactWorkbenchPreviewable(contentType) && q.Store != nil {
		previewPath := row.StoragePath
		if previewPath == "" {
			previewPath = row.Path
		}
		preview, truncated, err := q.Store.ReadPreview(previewPath, artifactWorkbenchPreviewMaxBytes)
		if err == nil {
			view.Preview = preview
			view.PreviewTruncated = truncated
		}
	}

	for _, sibling := range row.Siblings {
		view.Siblings = append(view.Siblings, ArtifactWorkbenchSibling{
			ArtifactID: sibling.ID,
			Kind:       sibling.Kind,
			Summary:    sibling.Summary,
			Selected:   sibling.ID == row.ArtifactID,
		})
	}

	return view, nil
}

func artifactWorkbenchContentType(kind, path string) string {
	switch ext := strings.ToLower(filepath.Ext(path)); ext {
	case ".txt", ".log", ".diff", ".patch":
		return "text/plain; charset=utf-8"
	case ".md":
		return "text/markdown; charset=utf-8"
	case ".json":
		return "application/json"
	case ".yml", ".yaml":
		return "application/x-yaml"
	case ".xml":
		return "application/xml"
	case ".csv":
		return "text/csv; charset=utf-8"
	}

	if contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(path))); contentType != "" {
		return contentType
	}

	if isTextLikeArtifactKind(kind) {
		return "text/plain; charset=utf-8"
	}

	return ""
}

func artifactWorkbenchPreviewable(contentType string) bool {
	return strings.HasPrefix(contentType, "text/") ||
		contentType == "application/json" ||
		contentType == "application/xml" ||
		contentType == "application/x-yaml"
}

func isTextLikeArtifactKind(kind string) bool {
	if isRawLogArtifact(kind) {
		return true
	}
	switch kind {
	case "assistant_summary", "command_result", "diff_summary":
		return true
	default:
		return false
	}
}

func artifactWorkbenchURL(artifactID string) string {
	if artifactID == "" {
		return ""
	}
	return "/board/artifacts/workbench?" + url.Values{
		"artifact_id": []string{artifactID},
	}.Encode()
}

func runWorkbenchURL(runID string) string {
	if runID == "" {
		return ""
	}
	return "/board/runs/workbench?" + url.Values{
		"run_id": []string{runID},
	}.Encode()
}

func artifactRawContentURL(artifactID string) string {
	if artifactID == "" {
		return ""
	}
	return "/api/manager/artifacts/" + url.PathEscape(artifactID) + "/content"
}
