package command

import (
	"database/sql"

	modulepkg "github.com/sine-io/foreman/internal/domain/module"
	"github.com/sine-io/foreman/internal/ports"
)

type CreateModuleCommand struct {
	ID          string
	ProjectID   string
	Name        string
	Description string
}

type CreateModuleHandler struct {
	Projects ports.ProjectRepository
	Modules  ports.ModuleRepository
}

func NewCreateModuleHandler(projects ports.ProjectRepository, modules ports.ModuleRepository) *CreateModuleHandler {
	return &CreateModuleHandler{
		Projects: projects,
		Modules:  modules,
	}
}

func (h *CreateModuleHandler) Handle(cmd CreateModuleCommand) (modulepkg.Module, error) {
	if _, err := h.Projects.Get(cmd.ProjectID); err != nil {
		if err == sql.ErrNoRows {
			return modulepkg.Module{}, err
		}
		return modulepkg.Module{}, err
	}

	id := cmd.ID
	if id == "" {
		id = nextID("module")
	}

	record := modulepkg.New(id, cmd.ProjectID, cmd.Name, cmd.Description)
	if err := h.Modules.Save(record); err != nil {
		return modulepkg.Module{}, err
	}

	return record, nil
}
