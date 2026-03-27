package cli

import (
	"fmt"

	"github.com/sine-io/foreman/internal/app/command"
	"github.com/sine-io/foreman/internal/bootstrap"
	"github.com/spf13/cobra"
)

func init() {}

func newApproveCommand(app bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "approve <task-id>",
		Short: "Approve a task action",
		RunE: func(cmd *cobra.Command, args []string) error {
			state, err := app.ApproveTask(command.ApproveTaskCommand{TaskID: args[0]})
			if err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "task %s state=%s\n", args[0], state)
			return err
		},
		Args: cobra.ExactArgs(1),
	}
}

func newRetryCommand(app bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "retry <task-id>",
		Short: "Retry a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			state, err := app.RetryTask(command.RetryTaskCommand{TaskID: args[0]})
			if err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "task %s state=%s\n", args[0], state)
			return err
		},
		Args: cobra.ExactArgs(1),
	}
}

func newCancelCommand(app bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "cancel <task-id>",
		Short: "Cancel a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			state, err := app.CancelTask(command.CancelTaskCommand{TaskID: args[0]})
			if err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "task %s state=%s\n", args[0], state)
			return err
		},
		Args: cobra.ExactArgs(1),
	}
}

func newReprioritizeCommand(app bootstrap.App) *cobra.Command {
	var priority int

	cmd := &cobra.Command{
		Use:   "reprioritize <task-id>",
		Short: "Reprioritize a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			state, err := app.ReprioritizeTask(command.ReprioritizeTaskCommand{
				TaskID:   args[0],
				Priority: priority,
			})
			if err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "task %s state=%s priority=%d\n", args[0], state, priority)
			return err
		},
		Args: cobra.ExactArgs(1),
	}

	cmd.Flags().IntVar(&priority, "priority", 0, "Updated priority")
	_ = cmd.MarkFlagRequired("priority")

	return cmd
}
