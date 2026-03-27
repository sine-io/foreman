package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/sine-io/foreman/internal/app/command"
	"github.com/sine-io/foreman/internal/app/query"
	modulepkg "github.com/sine-io/foreman/internal/domain/module"
	"github.com/sine-io/foreman/internal/domain/project"
	"github.com/stretchr/testify/require"
)

type fakeApp struct {
	serveCalled bool
	projectCmd  command.CreateProjectCommand
	moduleCmd   command.CreateModuleCommand
	taskCmd     command.CreateTaskCommand
	approveCmd  command.ApproveTaskCommand
	retryCmd    command.RetryTaskCommand
	cancelCmd   command.CancelTaskCommand
	reprioCmd   command.ReprioritizeTaskCommand
}

func (f *fakeApp) Serve(context.Context) error {
	f.serveCalled = true
	return nil
}

func (f *fakeApp) CreateProject(cmd command.CreateProjectCommand) (project.Project, error) {
	f.projectCmd = cmd
	return project.New(cmd.ID, cmd.Name, cmd.RepoRoot), nil
}

func (f *fakeApp) CreateModule(cmd command.CreateModuleCommand) (modulepkg.Module, error) {
	f.moduleCmd = cmd
	return modulepkg.New(cmd.ID, cmd.ProjectID, cmd.Name, cmd.Description), nil
}

func (f *fakeApp) CreateTask(cmd command.CreateTaskCommand) (command.TaskDTO, error) {
	f.taskCmd = cmd
	return command.TaskDTO{
		ID:       cmd.ID,
		ModuleID: cmd.ModuleID,
		State:    "ready",
		Summary:  cmd.Title,
		Priority: cmd.Priority,
	}, nil
}

func (f *fakeApp) ApprovalQueue(projectID string) (query.ApprovalQueueView, error) {
	return query.ApprovalQueueView{}, nil
}

func (f *fakeApp) ApproveTask(cmd command.ApproveTaskCommand) (string, error) {
	f.approveCmd = cmd
	return "leased", nil
}

func (f *fakeApp) RetryTask(cmd command.RetryTaskCommand) (string, error) {
	f.retryCmd = cmd
	return "ready", nil
}

func (f *fakeApp) CancelTask(cmd command.CancelTaskCommand) (string, error) {
	f.cancelCmd = cmd
	return "canceled", nil
}

func (f *fakeApp) ReprioritizeTask(cmd command.ReprioritizeTaskCommand) (string, error) {
	f.reprioCmd = cmd
	return "ready", nil
}

func TestRootCommandRequiresSubcommand(t *testing.T) {
	cmd := NewRootCommand(&fakeApp{})
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	require.Error(t, err)
}

func TestServeCommandCallsAppServe(t *testing.T) {
	app := &fakeApp{}
	cmd := NewRootCommand(app)
	cmd.SetArgs([]string{"serve"})

	err := cmd.Execute()

	require.NoError(t, err)
	require.True(t, app.serveCalled)
}

func TestProjectCreateCommandCallsAppCreateProject(t *testing.T) {
	app := &fakeApp{}
	cmd := NewRootCommand(app)
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"project", "create", "--id", "project-1", "--name", "Demo", "--repo-root", "/tmp/demo"})

	err := cmd.Execute()

	require.NoError(t, err)
	require.Equal(t, command.CreateProjectCommand{
		ID:       "project-1",
		Name:     "Demo",
		RepoRoot: "/tmp/demo",
	}, app.projectCmd)
	require.Contains(t, out.String(), "project-1")
}

func TestProjectModuleCreateCommandCallsAppCreateModule(t *testing.T) {
	app := &fakeApp{}
	cmd := NewRootCommand(app)
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"project", "module", "create", "--id", "module-1", "--project-id", "project-1", "--name", "Inbox", "--description", "Ingress work"})

	err := cmd.Execute()

	require.NoError(t, err)
	require.Equal(t, command.CreateModuleCommand{
		ID:          "module-1",
		ProjectID:   "project-1",
		Name:        "Inbox",
		Description: "Ingress work",
	}, app.moduleCmd)
	require.Contains(t, out.String(), "module-1")
}

func TestTaskCreateCommandCallsAppCreateTask(t *testing.T) {
	app := &fakeApp{}
	cmd := NewRootCommand(app)
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{
		"task", "create",
		"--id", "task-1",
		"--module-id", "module-1",
		"--title", "Implement board query",
		"--task-type", "write",
		"--write-scope", "repo:project-1",
		"--acceptance", "Board query returns columns",
		"--priority", "10",
	})

	err := cmd.Execute()

	require.NoError(t, err)
	require.Equal(t, command.CreateTaskCommand{
		ID:         "task-1",
		ModuleID:   "module-1",
		Title:      "Implement board query",
		TaskType:   "write",
		WriteScope: "repo:project-1",
		Acceptance: "Board query returns columns",
		Priority:   10,
	}, app.taskCmd)
	require.Contains(t, out.String(), "ready")
}

func TestTaskApproveCommandCallsAppApproveTask(t *testing.T) {
	app := &fakeApp{}
	cmd := NewRootCommand(app)
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"task", "approve", "task-1"})

	err := cmd.Execute()

	require.NoError(t, err)
	require.Equal(t, command.ApproveTaskCommand{TaskID: "task-1"}, app.approveCmd)
	require.Contains(t, out.String(), "leased")
}

func TestTaskRetryCommandCallsAppRetryTask(t *testing.T) {
	app := &fakeApp{}
	cmd := NewRootCommand(app)
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"task", "retry", "task-1"})

	err := cmd.Execute()

	require.NoError(t, err)
	require.Equal(t, command.RetryTaskCommand{TaskID: "task-1"}, app.retryCmd)
	require.Contains(t, out.String(), "ready")
}

func TestTaskCancelCommandCallsAppCancelTask(t *testing.T) {
	app := &fakeApp{}
	cmd := NewRootCommand(app)
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"task", "cancel", "task-1"})

	err := cmd.Execute()

	require.NoError(t, err)
	require.Equal(t, command.CancelTaskCommand{TaskID: "task-1"}, app.cancelCmd)
	require.Contains(t, out.String(), "canceled")
}

func TestTaskReprioritizeCommandCallsAppReprioritizeTask(t *testing.T) {
	app := &fakeApp{}
	cmd := NewRootCommand(app)
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"task", "reprioritize", "task-1", "--priority", "7"})

	err := cmd.Execute()

	require.NoError(t, err)
	require.Equal(t, command.ReprioritizeTaskCommand{TaskID: "task-1", Priority: 7}, app.reprioCmd)
	require.Contains(t, out.String(), "ready")
}
