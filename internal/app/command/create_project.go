package command

import (
	"fmt"
	"time"

	"github.com/sine-io/foreman/internal/domain/project"
	"github.com/sine-io/foreman/internal/ports"
)

type CreateProjectCommand struct {
	ID       string
	Name     string
	RepoRoot string
}

type CreateProjectHandler struct {
	Projects ports.ProjectRepository
}

func NewCreateProjectHandler(projects ports.ProjectRepository) *CreateProjectHandler {
	return &CreateProjectHandler{Projects: projects}
}

func (h *CreateProjectHandler) Handle(cmd CreateProjectCommand) (project.Project, error) {
	id := cmd.ID
	if id == "" {
		id = nextID("project")
	}

	record := project.New(id, cmd.Name, cmd.RepoRoot)
	if err := h.Projects.Save(record); err != nil {
		return project.Project{}, err
	}

	return record, nil
}

func nextID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}
