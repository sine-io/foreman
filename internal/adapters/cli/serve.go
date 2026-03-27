package cli

import (
	"github.com/sine-io/foreman/internal/bootstrap"
	"github.com/spf13/cobra"
)

func newServeCommand(app bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the embedded control plane",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.Serve(cmd.Context())
		},
	}
}
