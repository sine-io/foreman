package cli

import (
	"fmt"
	"path/filepath"

	"github.com/sine-io/foreman/internal/app/command"
	"github.com/sine-io/foreman/internal/bootstrap"
	"github.com/spf13/cobra"
)

func newProjectCommand(app bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Project and module commands",
	}

	cmd.AddCommand(newProjectCreateCommand(app))
	cmd.AddCommand(newModuleCommand(app))

	return cmd
}

func newProjectCreateCommand(app bootstrap.App) *cobra.Command {
	var createCmd command.CreateProjectCommand

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			record, err := app.CreateProject(createCmd)
			if err != nil {
				return err
			}

			_, err = fmt.Fprintf(
				cmd.OutOrStdout(),
				"created project %s (%s)\n",
				record.ID,
				record.Name,
			)
			return err
		},
	}

	cmd.Flags().StringVar(&createCmd.ID, "id", "", "Project ID")
	cmd.Flags().StringVar(&createCmd.Name, "name", "", "Project name")
	cmd.Flags().StringVar(&createCmd.RepoRoot, "repo-root", ".", "Repository root")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func newModuleCommand(app bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "module",
		Short: "Module commands",
	}

	cmd.AddCommand(newModuleCreateCommand(app))

	return cmd
}

func newModuleCreateCommand(app bootstrap.App) *cobra.Command {
	var createCmd command.CreateModuleCommand

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a module",
		RunE: func(cmd *cobra.Command, args []string) error {
			record, err := app.CreateModule(createCmd)
			if err != nil {
				return err
			}

			_, err = fmt.Fprintf(
				cmd.OutOrStdout(),
				"created module %s (%s)\n",
				record.ID,
				record.Name,
			)
			return err
		},
	}

	cmd.Flags().StringVar(&createCmd.ID, "id", "", "Module ID")
	cmd.Flags().StringVar(&createCmd.ProjectID, "project-id", "", "Project ID")
	cmd.Flags().StringVar(&createCmd.Name, "name", "", "Module name")
	cmd.Flags().StringVar(&createCmd.Description, "description", "", "Module description")
	_ = cmd.MarkFlagRequired("project-id")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func newActionCommand(app bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Task lifecycle commands",
	}

	cmd.AddCommand(
		newTaskCreateCommand(app),
		newApproveCommand(app),
		newRetryCommand(app),
		newCancelCommand(app),
		newReprioritizeCommand(app),
	)

	return cmd
}

func newTaskCreateCommand(app bootstrap.App) *cobra.Command {
	var createCmd command.CreateTaskCommand

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			record, err := app.CreateTask(createCmd)
			if err != nil {
				return err
			}

			_, err = fmt.Fprintf(
				cmd.OutOrStdout(),
				"created task %s state=%s\n",
				record.ID,
				record.State,
			)
			return err
		},
	}

	cmd.Flags().StringVar(&createCmd.ID, "id", "", "Task ID")
	cmd.Flags().StringVar(&createCmd.ModuleID, "module-id", "", "Module ID")
	cmd.Flags().StringVar(&createCmd.Title, "title", "", "Task title")
	cmd.Flags().StringVar(&createCmd.TaskType, "task-type", "write", "Task type")
	cmd.Flags().StringVar(&createCmd.WriteScope, "write-scope", "", "Writable scope")
	cmd.Flags().StringVar(&createCmd.Acceptance, "acceptance", "", "Acceptance criteria")
	cmd.Flags().IntVar(&createCmd.Priority, "priority", 0, "Task priority")
	_ = cmd.MarkFlagRequired("module-id")
	_ = cmd.MarkFlagRequired("title")

	return cmd
}

func defaultRepoRoot() string {
	root, err := filepath.Abs(".")
	if err != nil {
		return "."
	}

	return root
}
