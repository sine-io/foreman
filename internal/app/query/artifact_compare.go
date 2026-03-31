package query

import (
	"fmt"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/sine-io/foreman/internal/ports"
)

const artifactCompareMaxBytes = 64 * 1024

const (
	artifactCompareStatusReady       = "ready"
	artifactCompareStatusNoPrevious  = "no_previous"
	artifactCompareStatusUnsupported = "unsupported"
	artifactCompareStatusTooLarge    = "too_large"
)

type ArtifactCompareArtifact struct {
	ArtifactID  string `json:"artifact_id"`
	RunID       string `json:"run_id"`
	TaskID      string `json:"task_id"`
	Kind        string `json:"kind"`
	ContentType string `json:"content_type"`
	CreatedAt   string `json:"created_at"`
}

type ArtifactCompareDiff struct {
	Format  string `json:"format"`
	Content string `json:"content"`
}

type ArtifactCompareLimits struct {
	MaxCompareBytes int `json:"max_compare_bytes"`
}

type ArtifactCompareMessages struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

type ArtifactCompareNavigation struct {
	CurrentWorkbenchURL  string `json:"current_workbench_url"`
	PreviousWorkbenchURL string `json:"previous_workbench_url,omitempty"`
	BackToRunURL         string `json:"back_to_run_url"`
}

type ArtifactCompareView struct {
	Current    ArtifactCompareArtifact   `json:"current"`
	Previous   *ArtifactCompareArtifact  `json:"previous"`
	Status     string                    `json:"status"`
	Diff       *ArtifactCompareDiff      `json:"diff,omitempty"`
	Limits     ArtifactCompareLimits     `json:"limits"`
	Messages   ArtifactCompareMessages   `json:"messages"`
	Navigation ArtifactCompareNavigation `json:"navigation"`
}

type ArtifactCompareQuery struct {
	Repo  ports.BoardQueryRepository
	Store ports.ArtifactStore
}

func NewArtifactCompareQuery(repo ports.BoardQueryRepository, store ports.ArtifactStore) *ArtifactCompareQuery {
	return &ArtifactCompareQuery{Repo: repo, Store: store}
}

func (q *ArtifactCompareQuery) Execute(artifactID string) (ArtifactCompareView, error) {
	row, err := q.Repo.GetArtifactCompare(artifactID)
	if err != nil {
		return ArtifactCompareView{}, err
	}

	view := ArtifactCompareView{
		Current: artifactCompareViewArtifact(row.Current),
		Status:  artifactCompareStatusNoPrevious,
		Limits: ArtifactCompareLimits{
			MaxCompareBytes: artifactCompareMaxBytes,
		},
		Navigation: ArtifactCompareNavigation{
			CurrentWorkbenchURL: artifactWorkbenchURL(row.Current.ArtifactID),
			BackToRunURL:        runWorkbenchURL(row.Current.RunID),
		},
	}

	if row.Previous == nil {
		if !artifactCompareSupported(view.Current.ContentType) {
			view.Status = artifactCompareStatusUnsupported
		}
		view.Messages = artifactCompareMessages(view.Status, artifactCompareMaxBytes)
		return view, nil
	}

	previous := artifactCompareViewArtifact(*row.Previous)
	view.Previous = &previous
	view.Navigation.PreviousWorkbenchURL = artifactWorkbenchURL(previous.ArtifactID)

	if !artifactCompareSupported(view.Current.ContentType) || !artifactCompareSupported(previous.ContentType) {
		view.Status = artifactCompareStatusUnsupported
		view.Messages = artifactCompareMessages(view.Status, artifactCompareMaxBytes)
		return view, nil
	}

	if q.Store == nil {
		return ArtifactCompareView{}, fmt.Errorf("artifact compare requires an artifact store")
	}

	previousContent, previousTruncated, err := artifactCompareReadContent(q.Store, *row.Previous)
	if err != nil {
		return ArtifactCompareView{}, err
	}
	currentContent, currentTruncated, err := artifactCompareReadContent(q.Store, row.Current)
	if err != nil {
		return ArtifactCompareView{}, err
	}

	if previousTruncated || currentTruncated {
		view.Status = artifactCompareStatusTooLarge
		view.Messages = artifactCompareMessages(view.Status, artifactCompareMaxBytes)
		return view, nil
	}

	diffContent, err := artifactCompareUnifiedDiff(*row.Previous, row.Current, previousContent, currentContent)
	if err != nil {
		return ArtifactCompareView{}, err
	}

	view.Status = artifactCompareStatusReady
	view.Diff = &ArtifactCompareDiff{
		Format:  "text/unified-diff",
		Content: diffContent,
	}
	view.Messages = artifactCompareMessages(view.Status, artifactCompareMaxBytes)

	return view, nil
}

func artifactCompareViewArtifact(row ports.ArtifactCompareArtifactRow) ArtifactCompareArtifact {
	return ArtifactCompareArtifact{
		ArtifactID:  row.ArtifactID,
		RunID:       row.RunID,
		TaskID:      row.TaskID,
		Kind:        row.Kind,
		ContentType: artifactWorkbenchContentType(row.Kind, artifactWorkbenchContentTypePath(row.Path, row.StoragePath)),
		CreatedAt:   row.CreatedAt,
	}
}

func artifactCompareSupported(contentType string) bool {
	return artifactWorkbenchPreviewable(contentType)
}

func artifactCompareReadContent(store ports.ArtifactStore, row ports.ArtifactCompareArtifactRow) (string, bool, error) {
	path := strings.TrimSpace(row.StoragePath)
	if path == "" {
		path = strings.TrimSpace(row.Path)
	}
	return store.ReadPreview(path, artifactCompareMaxBytes)
}

func artifactCompareMessages(status string, maxCompareBytes int) ArtifactCompareMessages {
	switch status {
	case artifactCompareStatusReady:
		return ArtifactCompareMessages{
			Title:  "Compare ready",
			Detail: "Showing a unified diff between the current artifact and the previous artifact.",
		}
	case artifactCompareStatusUnsupported:
		return ArtifactCompareMessages{
			Title:  "Compare unavailable",
			Detail: "Artifact compare currently supports text and structured-text content only.",
		}
	case artifactCompareStatusTooLarge:
		return ArtifactCompareMessages{
			Title:  "Compare too large",
			Detail: fmt.Sprintf("One or both artifacts exceed the %d byte compare limit.", maxCompareBytes),
		}
	default:
		return ArtifactCompareMessages{
			Title:  "No previous artifact",
			Detail: "No earlier artifact with the same task and kind is available for compare.",
		}
	}
}

func artifactCompareUnifiedDiff(previous, current ports.ArtifactCompareArtifactRow, previousContent, currentContent string) (string, error) {
	return difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(artifactCompareDiffInput(previousContent)),
		B:        difflib.SplitLines(artifactCompareDiffInput(currentContent)),
		FromFile: "previous:" + previous.ArtifactID,
		ToFile:   "current:" + current.ArtifactID,
		Context:  3,
	})
}

func artifactCompareDiffInput(content string) string {
	if content == "" || strings.HasSuffix(content, "\n") {
		return content
	}
	return content + "\n"
}
