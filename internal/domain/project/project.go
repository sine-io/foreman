package project

type Project struct {
	ID                   string
	Name                 string
	RepoRoot             string
	DefaultPolicyProfile string
	ManagerProfile       string
}

func New(id, name, repoRoot string) Project {
	return Project{
		ID:       id,
		Name:     name,
		RepoRoot: repoRoot,
	}
}
