package cli

import (
	"fmt"

	"github.com/sine-io/foreman/internal/bootstrap"
	"github.com/spf13/cobra"
)

func newProjectCommand(app bootstrap.App) *cobra.Command {
	_ = app

	return &cobra.Command{
		Use:   "project",
		Short: "Project and module commands",
	}
}

func newActionCommand(app bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Task lifecycle commands",
	}

	cmd.AddCommand(
		newApproveCommand(app),
		newRetryCommand(app),
		newCancelCommand(app),
		newReprioritizeCommand(app),
	)

	return cmd
}

func notWiredError(name string) error {
	return fmt.Errorf("%s command is not wired yet", name)
}
