package module

type BoardState string

const (
	BoardStatePlanned BoardState = "planned"
	BoardStateActive  BoardState = "active"
	BoardStateBlocked BoardState = "blocked"
	BoardStateDone    BoardState = "done"
)

type Module struct {
	ID                 string
	ProjectID          string
	Name               string
	Description        string
	State              BoardState
	CompletionCriteria []string
}

func New(id, projectID, name, description string) Module {
	return Module{
		ID:          id,
		ProjectID:   projectID,
		Name:        name,
		Description: description,
		State:       BoardStatePlanned,
	}
}
