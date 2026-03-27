package ports

type ManagerCommand struct {
	ProjectID string
	ModuleID  string
	TaskID    string
	Command   string
}

type ManagerAgentGateway interface {
	Name() string
	Handle(ManagerCommand) error
}
