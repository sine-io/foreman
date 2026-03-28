package manageragent

import (
	"context"
	"database/sql"
	"testing"

	"github.com/sine-io/foreman/internal/app/command"
	"github.com/sine-io/foreman/internal/app/query"
	"github.com/sine-io/foreman/internal/domain/approval"
	modulepkg "github.com/sine-io/foreman/internal/domain/module"
	domainpolicy "github.com/sine-io/foreman/internal/domain/policy"
	projectpkg "github.com/sine-io/foreman/internal/domain/project"
	"github.com/sine-io/foreman/internal/domain/task"
	"github.com/sine-io/foreman/internal/ports"
	"github.com/stretchr/testify/require"
)

func TestHandleCreateProjectReturnsCreatedProject(t *testing.T) {
	harness := newHarness()
	svc := harness.newService()

	out, err := svc.Handle(context.Background(), Request{
		Kind:      "create_project",
		SessionID: "mgr-1",
		ProjectID: "project-1",
		Name:      "Alpha",
		RepoRoot:  "/tmp/alpha",
	})
	require.NoError(t, err)
	require.Equal(t, "project_created", out.Kind)
	require.Equal(t, "project-1", out.ProjectID)

	saved, err := harness.projects.Get("project-1")
	require.NoError(t, err)
	require.Equal(t, "Alpha", saved.Name)
	require.Equal(t, "/tmp/alpha", saved.RepoRoot)
}

func TestHandleCreateModuleReturnsCreatedModule(t *testing.T) {
	harness := newHarness()
	svc := harness.newService()

	out, err := svc.Handle(context.Background(), Request{
		Kind:        "create_module",
		SessionID:   "mgr-1",
		ProjectID:   "project-1",
		ModuleID:    "module-1",
		Name:        "Inbox",
		Description: "OpenClaw ingress module",
	})
	require.NoError(t, err)
	require.Equal(t, "module_created", out.Kind)
	require.Equal(t, "project-1", out.ProjectID)
	require.Equal(t, "module-1", out.ModuleID)

	saved, err := harness.modules.Get("module-1")
	require.NoError(t, err)
	require.Equal(t, "project-1", saved.ProjectID)
	require.Equal(t, "Inbox", saved.Name)
}

func TestHandleCreateTaskReturnsCompletionWhenDispatchFinishes(t *testing.T) {
	harness := newHarness()
	svc := harness.newService()

	out, err := svc.Handle(context.Background(), Request{
		Kind:      "create_task",
		SessionID: "mgr-1",
		ProjectID: "project-1",
		ModuleID:  "module-1",
		Summary:   "Summarize the module status",
	})
	require.NoError(t, err)
	require.Equal(t, "completion", out.Kind)
	require.NotEmpty(t, out.TaskID)

	saved, err := harness.tasks.Get(out.TaskID)
	require.NoError(t, err)
	require.Equal(t, "module-1", saved.ModuleID)
	require.Equal(t, "repo:project-1", saved.WriteScope)
}

func TestHandleCreateTaskReturnsApprovalNeededWhenPolicyRequiresApproval(t *testing.T) {
	harness := newHarness()
	harness.policyDecision = domainpolicy.Decision{
		RequiresApproval: true,
		Reason:           "git push origin main requires approval",
	}
	svc := harness.newService()

	out, err := svc.Handle(context.Background(), Request{
		Kind:      "create_task",
		SessionID: "mgr-2",
		ProjectID: "project-1",
		ModuleID:  "module-1",
		Summary:   "git push origin main",
	})
	require.NoError(t, err)
	require.Equal(t, "approval_needed", out.Kind)
	require.NotEmpty(t, out.Summary)
}

func TestHandleDispatchTaskReturnsCompletionForExistingTask(t *testing.T) {
	harness := newHarness()
	require.NoError(t, harness.tasks.Save(task.NewTask("task-1", "module-1", task.TaskTypeWrite, "Dispatch me", "repo:project-1")))
	svc := harness.newService()

	out, err := svc.Handle(context.Background(), Request{
		Kind:    "dispatch_task",
		TaskID:  "task-1",
		Summary: "Dispatch me",
	})
	require.NoError(t, err)
	require.Equal(t, "completion", out.Kind)
	require.Equal(t, "task-1", out.TaskID)
}

func TestTaskStatusReturnsPersistedRunAndApprovalState(t *testing.T) {
	harness := newHarness()
	harness.board.tasksByProject["project-1"] = []ports.TaskBoardRow{
		{
			TaskID:          "task-1",
			ModuleID:        "module-1",
			Summary:         "Summarize the module status",
			State:           "waiting_approval",
			Priority:        10,
			PendingApproval: true,
		},
	}
	svc := harness.newService()

	view, err := svc.TaskStatus(context.Background(), "task-1")
	require.NoError(t, err)
	require.Equal(t, "task-1", view.TaskID)
	require.Equal(t, "waiting_approval", view.State)
	require.True(t, view.PendingApproval)
}

func TestBoardSnapshotReturnsModuleAndTaskColumns(t *testing.T) {
	harness := newHarness()
	harness.board.modulesByProject["project-1"] = []ports.ModuleBoardRow{
		{ModuleID: "module-1", Name: "Inbox", BoardState: "active"},
	}
	harness.board.tasksByProject["project-1"] = []ports.TaskBoardRow{
		{TaskID: "task-1", ModuleID: "module-1", Summary: "Bootstrap board", State: "ready", Priority: 10},
	}
	svc := harness.newService()

	view, err := svc.BoardSnapshot(context.Background(), "project-1")
	require.NoError(t, err)
	require.Equal(t, "project-1", view.ProjectID)
	require.Len(t, view.Modules["Implementing"], 1)
	require.Len(t, view.Tasks["Ready"], 1)
	require.Equal(t, "task-1", view.Tasks["Ready"][0].TaskID)
}

type serviceHarness struct {
	projects       *fakeProjectRepo
	modules        *fakeModuleRepo
	tasks          *fakeTaskRepo
	approvals      *fakeApprovalRepo
	runs           *fakeRunRepo
	board          *fakeBoardQueryRepo
	policyDecision domainpolicy.Decision
}

func newHarness() *serviceHarness {
	return &serviceHarness{
		projects:  newFakeProjectRepo(),
		modules:   newFakeModuleRepo(),
		tasks:     newFakeTaskRepo(),
		approvals: &fakeApprovalRepo{byTaskID: map[string]approval.Approval{}},
		runs:      &fakeRunRepo{},
		board: &fakeBoardQueryRepo{
			modulesByProject: map[string][]ports.ModuleBoardRow{},
			tasksByProject:   map[string][]ports.TaskBoardRow{},
		},
	}
}

func (h *serviceHarness) newService() *Service {
	return NewService(Dependencies{
		CreateProject: command.NewCreateProjectHandler(h.projects),
		CreateModule:  command.NewCreateModuleHandler(h.modules),
		CreateTask:    command.NewCreateTaskHandler(h.tasks),
		DispatchTask: command.NewDispatchTaskHandler(
			h.tasks,
			&fakeLeaseRepo{},
			fakePolicy{decision: h.policyDecision},
			fakeRunner{},
			h.approvals,
			h.runs,
			&fakeArtifactRepo{},
		),
		QueryModuleBoard: query.NewModuleBoardQuery(h.board),
		QueryTaskBoard:   query.NewTaskBoardQuery(h.board),
		Defaults: Defaults{
			ProjectID: "project-1",
			ModuleID:  "module-1",
		},
	})
}

type fakeProjectRepo struct {
	byID map[string]projectpkg.Project
}

func newFakeProjectRepo() *fakeProjectRepo {
	return &fakeProjectRepo{byID: map[string]projectpkg.Project{}}
}

func (f *fakeProjectRepo) Save(value projectpkg.Project) error {
	f.byID[value.ID] = value
	return nil
}

func (f *fakeProjectRepo) Get(id string) (projectpkg.Project, error) {
	value, ok := f.byID[id]
	if !ok {
		return projectpkg.Project{}, sql.ErrNoRows
	}
	return value, nil
}

type fakeModuleRepo struct {
	byID map[string]modulepkg.Module
}

func newFakeModuleRepo() *fakeModuleRepo {
	return &fakeModuleRepo{byID: map[string]modulepkg.Module{}}
}

func (f *fakeModuleRepo) Save(value modulepkg.Module) error {
	f.byID[value.ID] = value
	return nil
}

func (f *fakeModuleRepo) Get(id string) (modulepkg.Module, error) {
	value, ok := f.byID[id]
	if !ok {
		return modulepkg.Module{}, sql.ErrNoRows
	}
	return value, nil
}

type fakeTaskRepo struct {
	byID map[string]task.Task
}

func newFakeTaskRepo() *fakeTaskRepo {
	return &fakeTaskRepo{
		byID: map[string]task.Task{},
	}
}

func (f *fakeTaskRepo) Save(value task.Task) error {
	f.byID[value.ID] = value
	return nil
}

func (f *fakeTaskRepo) Get(id string) (task.Task, error) {
	value, ok := f.byID[id]
	if !ok {
		return task.Task{}, sql.ErrNoRows
	}

	return value, nil
}

type fakeApprovalRepo struct {
	byTaskID map[string]approval.Approval
	saved    approval.Approval
}

func (f *fakeApprovalRepo) Save(value approval.Approval) error {
	f.saved = value
	f.byTaskID[value.TaskID] = value
	return nil
}

func (f *fakeApprovalRepo) Get(id string) (approval.Approval, error) {
	if f.saved.ID == id {
		return f.saved, nil
	}

	for _, value := range f.byTaskID {
		if value.ID == id {
			return value, nil
		}
	}

	return approval.Approval{}, sql.ErrNoRows
}

func (f *fakeApprovalRepo) FindPendingByTask(taskID string) (approval.Approval, error) {
	value, ok := f.byTaskID[taskID]
	if !ok {
		return approval.Approval{}, sql.ErrNoRows
	}

	if value.Status != approval.StatusPending {
		return approval.Approval{}, sql.ErrNoRows
	}

	return value, nil
}

type fakeLeaseRepo struct{}

func (f *fakeLeaseRepo) Acquire(taskID, scopeKey string) error {
	return nil
}

func (f *fakeLeaseRepo) Release(scopeKey string) error {
	return nil
}

type fakePolicy struct {
	decision domainpolicy.Decision
}

func (f fakePolicy) Evaluate(action string) domainpolicy.Decision {
	return f.decision
}

type fakeRunner struct{}

func (fakeRunner) Dispatch(req ports.RunRequest) (ports.Run, error) {
	return ports.Run{
		ID:                   "run-1",
		TaskID:               req.TaskID,
		RunnerKind:           "codex",
		State:                "completed",
		AssistantSummaryPath: "artifacts/tasks/task-1/assistant_summary.txt",
	}, nil
}

func (fakeRunner) Observe(runID string) (ports.Run, error) {
	return ports.Run{}, nil
}

func (fakeRunner) Stop(runID string) error {
	return nil
}

type fakeRunRepo struct {
	saved ports.Run
}

func (f *fakeRunRepo) Save(run ports.Run) error {
	f.saved = run
	return nil
}

func (f *fakeRunRepo) Get(id string) (ports.Run, error) {
	if f.saved.ID != id {
		return ports.Run{}, sql.ErrNoRows
	}

	return f.saved, nil
}

type fakeArtifactRepo struct{}

func (f *fakeArtifactRepo) Create(taskID, kind, path string) (string, error) {
	return "artifact-1", nil
}

func (f *fakeArtifactRepo) Get(id string) (ports.ArtifactRecord, error) {
	if id == "" {
		return ports.ArtifactRecord{}, sql.ErrNoRows
	}

	return ports.ArtifactRecord{ID: id}, nil
}

type fakeBoardQueryRepo struct {
	modulesByProject map[string][]ports.ModuleBoardRow
	tasksByProject   map[string][]ports.TaskBoardRow
}

func (f *fakeBoardQueryRepo) ListModules(projectID string) ([]ports.ModuleBoardRow, error) {
	return f.modulesByProject[projectID], nil
}

func (f *fakeBoardQueryRepo) ListTasks(projectID string) ([]ports.TaskBoardRow, error) {
	return f.tasksByProject[projectID], nil
}

func (f *fakeBoardQueryRepo) GetRunDetail(runID string) (ports.RunDetailRecord, error) {
	return ports.RunDetailRecord{}, sql.ErrNoRows
}

func (f *fakeBoardQueryRepo) ListApprovals(projectID string) ([]ports.ApprovalQueueRow, error) {
	return nil, nil
}
