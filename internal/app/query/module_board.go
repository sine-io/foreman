package query

import "github.com/sine-io/foreman/internal/ports"

type ModuleCard struct {
	ID    string
	Name  string
	State string
}

type ModuleBoardView struct {
	Columns map[string][]ModuleCard
}

type ModuleBoardQuery struct {
	Repo ports.BoardQueryRepository
}

func NewModuleBoardQuery(repo ports.BoardQueryRepository) *ModuleBoardQuery {
	return &ModuleBoardQuery{Repo: repo}
}

func (q *ModuleBoardQuery) Execute(projectID string) (ModuleBoardView, error) {
	rows, err := q.Repo.ListModules(projectID)
	if err != nil {
		return ModuleBoardView{}, err
	}

	view := ModuleBoardView{
		Columns: map[string][]ModuleCard{},
	}

	for _, row := range rows {
		column := moduleColumnName(row.BoardState)
		view.Columns[column] = append(view.Columns[column], ModuleCard{
			ID:    row.ModuleID,
			Name:  row.Name,
			State: row.BoardState,
		})
	}

	return view, nil
}

func moduleColumnName(state string) string {
	switch state {
	case "planned":
		return "Backlog"
	case "active":
		return "Implementing"
	case "blocked":
		return "Blocked"
	case "done":
		return "Done"
	default:
		return state
	}
}
