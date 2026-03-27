package command

import (
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
	Modules ports.ModuleRepository
}

func NewCreateModuleHandler(modules ports.ModuleRepository) *CreateModuleHandler {
	return &CreateModuleHandler{Modules: modules}
}

func (h *CreateModuleHandler) Handle(cmd CreateModuleCommand) (modulepkg.Module, error) {
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
