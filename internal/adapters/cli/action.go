package cli

import (
	"github.com/sine-io/foreman/internal/bootstrap"
	"github.com/spf13/cobra"
)

func init() {}

func newApproveCommand(app bootstrap.App) *cobra.Command {
	_ = app

	return &cobra.Command{
		Use:   "approve <task-id>",
		Short: "Approve a task action",
		RunE: func(cmd *cobra.Command, args []string) error {
			return notWiredError("approve")
		},
	}
}

func newRetryCommand(app bootstrap.App) *cobra.Command {
	_ = app

	return &cobra.Command{
		Use:   "retry <task-id>",
		Short: "Retry a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			return notWiredError("retry")
		},
	}
}

func newCancelCommand(app bootstrap.App) *cobra.Command {
	_ = app

	return &cobra.Command{
		Use:   "cancel <task-id>",
		Short: "Cancel a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			return notWiredError("cancel")
		},
	}
}

func newReprioritizeCommand(app bootstrap.App) *cobra.Command {
	_ = app

	return &cobra.Command{
		Use:   "reprioritize <task-id>",
		Short: "Reprioritize a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			return notWiredError("reprioritize")
		},
	}
}
