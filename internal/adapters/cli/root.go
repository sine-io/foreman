package cli

import (
	"fmt"

	"github.com/sine-io/foreman/internal/bootstrap"
	"github.com/spf13/cobra"
)

func NewRootCommand(app bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "foreman",
		Short: "Foreman embedded control plane",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = cmd.Help()
			return fmt.Errorf("subcommand required")
		},
	}

	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.AddCommand(newServeCommand(app))
	cmd.AddCommand(newProjectCommand(app))
	cmd.AddCommand(newActionCommand(app))

	return cmd
}
