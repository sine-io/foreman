package query

import "github.com/sine-io/foreman/internal/ports"

type ArtifactSummary struct {
	ID      string
	Kind    string
	Path    string
	Summary string
}

type RunDetailView struct {
	ID          string
	TaskID      string
	RunnerKind  string
	State       string
	TaskSummary string
	Artifacts   []ArtifactSummary
}

type RunDetailQuery struct {
	Repo ports.BoardQueryRepository
}

func NewRunDetailQuery(repo ports.BoardQueryRepository) *RunDetailQuery {
	return &RunDetailQuery{Repo: repo}
}

func (q *RunDetailQuery) Execute(runID string) (RunDetailView, error) {
	record, err := q.Repo.GetRunDetail(runID)
	if err != nil {
		return RunDetailView{}, err
	}

	view := RunDetailView{
		ID:          record.Run.ID,
		TaskID:      record.Run.TaskID,
		RunnerKind:  record.Run.RunnerKind,
		State:       record.Run.State,
		TaskSummary: record.TaskSummary,
	}

	for _, artifact := range record.Artifacts {
		view.Artifacts = append(view.Artifacts, ArtifactSummary{
			ID:      artifact.ID,
			Kind:    artifact.Kind,
			Path:    artifact.Path,
			Summary: artifact.Summary,
		})
	}

	return view, nil
}
