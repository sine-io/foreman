package command

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/sine-io/foreman/internal/domain/approval"
	modulepkg "github.com/sine-io/foreman/internal/domain/module"
	domainpolicy "github.com/sine-io/foreman/internal/domain/policy"
	"github.com/sine-io/foreman/internal/domain/task"
	"github.com/sine-io/foreman/internal/ports"
	"github.com/stretchr/testify/require"
)

func TestCreateTaskPersistsReadyTask(t *testing.T) {
	repo := newFakeTaskRepo()
	modules := newFakeModuleRepo()
	require.NoError(t, modules.Save(modulepkg.New("module-1", "project-1", "Module 1", "")))
	handler := NewCreateTaskHandler(modules, repo)

	out, err := handler.Handle(CreateTaskCommand{
		ModuleID:   "module-1",
		Title:      "Implement board query",
		TaskType:   "write",
		WriteScope: "repo:project-1",
		Acceptance: "Board query returns module columns",
		Priority:   10,
	})
	require.NoError(t, err)
	require.Equal(t, "ready", out.State)

	saved, err := repo.Get(out.ID)
	require.NoError(t, err)
	require.Equal(t, "Implement board query", saved.Summary)
	require.Equal(t, 10, saved.Priority)
}

func TestCreateTaskReturnsErrorWhenModuleMissing(t *testing.T) {
	handler := NewCreateTaskHandler(newFakeModuleRepo(), newFakeTaskRepo())

	_, err := handler.Handle(CreateTaskCommand{
		ModuleID:   "module-missing",
		Title:      "Implement board query",
		TaskType:   "write",
		WriteScope: "repo:project-1",
		Acceptance: "Board query returns module columns",
		Priority:   10,
	})
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestApproveTaskApprovesPendingApprovalAndDispatchesByTaskID(t *testing.T) {
	approvals := &fakeApprovalRepo{
		byTaskID: map[string]approval.Approval{
			"task-1": approval.New("approval-1", "task-1", "git push origin main"),
		},
	}
	tasks := newFakeTaskRepo()
	repoTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "git push origin main", "repo:project-1")
	repoTask.State = task.TaskStateWaitingApproval
	require.NoError(t, tasks.Save(repoTask))

	runs := &fakeRunRepo{}
	artifacts := &fakeArtifactRepo{}
	tx := newFakeTransactor(tasks, approvals, runs, artifacts)
	dispatch := NewDispatchTaskHandler(
		tx,
		tasks,
		&fakeLeaseRepo{},
		fakePolicy{
			decision: domainpolicy.Decision{
				RequiresApproval: true,
				Reason:           "git push origin main requires approval",
			},
		},
		&fakeRunner{},
		approvals,
		runs,
		artifacts,
	)
	handler := NewApproveTaskHandler(tx, approvals, tasks, dispatch)
	err := handler.Handle(ApproveTaskCommand{TaskID: "task-1"})
	require.NoError(t, err)

	_, err = approvals.FindPendingByTask("task-1")
	require.Error(t, err)
	require.Equal(t, approval.StatusApproved, approvals.saved.Status)

	savedTask, err := tasks.Get("task-1")
	require.NoError(t, err)
	require.Equal(t, task.TaskStateCompleted, savedTask.State)
}

func TestApproveTaskMarksApprovedPendingDispatchWhenDelegatedDispatchFails(t *testing.T) {
	approvals := &fakeApprovalRepo{
		byTaskID: map[string]approval.Approval{
			"task-1": approval.New("approval-1", "task-1", "git push origin main"),
		},
	}
	tasks := newFakeTaskRepo()
	repoTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "Implement board", "repo:project-1")
	repoTask.State = task.TaskStateWaitingApproval
	require.NoError(t, tasks.Save(repoTask))

	runs := &fakeRunRepo{}
	artifacts := &fakeArtifactRepo{}
	tx := newFakeTransactor(tasks, approvals, runs, artifacts)
	dispatch := NewDispatchTaskHandler(
		tx,
		tasks,
		&fakeLeaseRepo{},
		fakePolicy{
			decision: domainpolicy.Decision{
				RequiresApproval: true,
				Reason:           "git push origin main requires approval",
			},
		},
		&fakeRunner{dispatchErr: errors.New("runner unavailable")},
		approvals,
		runs,
		artifacts,
	)
	handler := NewApproveTaskHandler(tx, approvals, tasks, dispatch)

	err := handler.Handle(ApproveTaskCommand{TaskID: "task-1"})
	require.EqualError(t, err, "runner unavailable")

	latest, err := approvals.FindLatestByTask("task-1")
	require.NoError(t, err)
	require.Equal(t, approval.StatusApproved, latest.Status)

	savedTask, err := tasks.Get("task-1")
	require.NoError(t, err)
	require.Equal(t, task.TaskStateApprovedPendingDispatch, savedTask.State)
}

func TestApproveTaskIsIdempotentAfterApprovalAlreadyResolved(t *testing.T) {
	approvals := &fakeApprovalRepo{
		byTaskID: map[string]approval.Approval{
			"task-1": {
				ID:     "approval-1",
				TaskID: "task-1",
				Reason: "git push origin main",
				Status: approval.StatusApproved,
			},
		},
	}
	tasks := newFakeTaskRepo()
	repoTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "Implement board", "repo:project-1")
	repoTask.State = task.TaskStateCompleted
	require.NoError(t, tasks.Save(repoTask))

	runs := &fakeRunRepo{
		saved: ports.Run{
			ID:     "run-1",
			TaskID: "task-1",
			State:  "completed",
		},
	}
	tx := newFakeTransactor(tasks, approvals, runs, &fakeArtifactRepo{})
	handler := NewApproveTaskHandler(tx, approvals, tasks, NewDispatchTaskHandler(
		tx,
		tasks,
		&fakeLeaseRepo{},
		fakePolicy{decision: domainpolicy.Decision{}},
		&fakeRunner{},
		approvals,
		runs,
		&fakeArtifactRepo{},
	))

	err := handler.Handle(ApproveTaskCommand{TaskID: "task-1"})
	require.NoError(t, err)
}

func TestApproveTaskReturnsLatestApprovalLookupError(t *testing.T) {
	approvals := &fakeApprovalRepo{
		findLatestErr: errors.New("latest lookup failed"),
	}
	tasks := newFakeTaskRepo()
	repoTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "Implement board", "repo:project-1")
	repoTask.State = task.TaskStateWaitingApproval
	require.NoError(t, tasks.Save(repoTask))

	tx := newFakeTransactor(tasks, approvals, &fakeRunRepo{}, &fakeArtifactRepo{})
	handler := NewApproveTaskHandler(tx, approvals, tasks, NewDispatchTaskHandler(
		tx,
		tasks,
		&fakeLeaseRepo{},
		fakePolicy{decision: domainpolicy.Decision{}},
		&fakeRunner{},
		approvals,
		tx.runs,
		tx.artifacts,
	))

	err := handler.Handle(ApproveTaskCommand{TaskID: "task-1"})
	require.EqualError(t, err, "latest lookup failed")
}

func TestRetryTaskMovesFailedTaskToReady(t *testing.T) {
	tasks := newFakeTaskRepo()
	failedTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "Retry me", "repo:project-1")
	failedTask.State = task.TaskStateFailed
	require.NoError(t, tasks.Save(failedTask))

	handler := NewRetryTaskHandler(tasks)
	err := handler.Handle(RetryTaskCommand{TaskID: "task-1"})
	require.NoError(t, err)

	savedTask, err := tasks.Get("task-1")
	require.NoError(t, err)
	require.Equal(t, task.TaskStateReady, savedTask.State)
}

func TestCancelTaskMovesTaskToCanceled(t *testing.T) {
	tasks := newFakeTaskRepo()
	require.NoError(t, tasks.Save(task.NewTask("task-1", "module-1", task.TaskTypeWrite, "Cancel me", "repo:project-1")))

	handler := NewCancelTaskHandler(tasks)
	err := handler.Handle(CancelTaskCommand{TaskID: "task-1"})
	require.NoError(t, err)

	savedTask, err := tasks.Get("task-1")
	require.NoError(t, err)
	require.Equal(t, task.TaskStateCanceled, savedTask.State)
}

func TestReprioritizeTaskUpdatesPriority(t *testing.T) {
	tasks := newFakeTaskRepo()
	repoTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "Prioritize me", "repo:project-1")
	require.NoError(t, tasks.Save(repoTask))

	handler := NewReprioritizeTaskHandler(tasks)
	err := handler.Handle(ReprioritizeTaskCommand{TaskID: "task-1", Priority: 42})
	require.NoError(t, err)

	savedTask, err := tasks.Get("task-1")
	require.NoError(t, err)
	require.Equal(t, 42, savedTask.Priority)
}

type fakeTaskRepo struct {
	byID             map[string]task.Task
	failSaveForState task.TaskState
	failSaveCount    int
	afterGet         func(task.Task)
}

func newFakeTaskRepo() *fakeTaskRepo {
	return &fakeTaskRepo{
		byID: map[string]task.Task{},
	}
}

func (f *fakeTaskRepo) Save(value task.Task) error {
	if f.failSaveCount > 0 && value.State == f.failSaveForState {
		f.failSaveCount--
		return errors.New("save failed")
	}
	f.byID[value.ID] = value
	return nil
}

func (f *fakeTaskRepo) Get(id string) (task.Task, error) {
	value, ok := f.byID[id]
	if !ok {
		return task.Task{}, sql.ErrNoRows
	}

	if f.afterGet != nil {
		f.afterGet(value)
	}

	return value, nil
}

func (f *fakeTaskRepo) snapshot() fakeTaskRepoSnapshot {
	copyByID := make(map[string]task.Task, len(f.byID))
	for id, value := range f.byID {
		copyByID[id] = value
	}

	return fakeTaskRepoSnapshot{
		byID:             copyByID,
		failSaveForState: f.failSaveForState,
		failSaveCount:    f.failSaveCount,
	}
}

func (f *fakeTaskRepo) restore(snapshot fakeTaskRepoSnapshot) {
	f.byID = snapshot.byID
	f.failSaveForState = snapshot.failSaveForState
	f.failSaveCount = snapshot.failSaveCount
}

type fakeTaskRepoSnapshot struct {
	byID             map[string]task.Task
	failSaveForState task.TaskState
	failSaveCount    int
}

type fakeModuleRepo struct {
	byID map[string]modulepkg.Module
}

func newFakeModuleRepo() *fakeModuleRepo {
	return &fakeModuleRepo{
		byID: map[string]modulepkg.Module{},
	}
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

type fakeApprovalRepo struct {
	byTaskID        map[string]approval.Approval
	saved           approval.Approval
	saveCount       int
	saveErr         error
	findLatestErr   error
	findPendingErrs []error
}

func (f *fakeApprovalRepo) Save(value approval.Approval) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	if f.byTaskID == nil {
		f.byTaskID = map[string]approval.Approval{}
	}
	f.saveCount++
	f.saved = value
	f.byTaskID[value.TaskID] = value
	return nil
}

func (f *fakeApprovalRepo) Get(id string) (approval.Approval, error) {
	for _, value := range f.byTaskID {
		if value.ID == id {
			return value, nil
		}
	}

	if f.saved.ID == id {
		return f.saved, nil
	}

	return approval.Approval{}, sql.ErrNoRows
}

func (f *fakeApprovalRepo) FindPendingByTask(taskID string) (approval.Approval, error) {
	if len(f.findPendingErrs) > 0 {
		err := f.findPendingErrs[0]
		f.findPendingErrs = f.findPendingErrs[1:]
		return approval.Approval{}, err
	}

	value, ok := f.byTaskID[taskID]
	if !ok {
		return approval.Approval{}, sql.ErrNoRows
	}

	if value.Status != approval.StatusPending {
		return approval.Approval{}, sql.ErrNoRows
	}

	return value, nil
}

func (f *fakeApprovalRepo) pendingCount() int {
	count := 0
	for _, value := range f.byTaskID {
		if value.Status == approval.StatusPending {
			count++
		}
	}
	return count
}

func (f *fakeApprovalRepo) snapshot() fakeApprovalRepoSnapshot {
	copyByTaskID := make(map[string]approval.Approval, len(f.byTaskID))
	for taskID, value := range f.byTaskID {
		copyByTaskID[taskID] = value
	}
	findPendingErrs := make([]error, len(f.findPendingErrs))
	copy(findPendingErrs, f.findPendingErrs)

	return fakeApprovalRepoSnapshot{
		byTaskID:        copyByTaskID,
		saved:           f.saved,
		saveCount:       f.saveCount,
		findLatestErr:   f.findLatestErr,
		findPendingErrs: findPendingErrs,
	}
}

func (f *fakeApprovalRepo) restore(snapshot fakeApprovalRepoSnapshot) {
	f.byTaskID = snapshot.byTaskID
	f.saved = snapshot.saved
	f.saveCount = snapshot.saveCount
	f.findLatestErr = snapshot.findLatestErr
	f.findPendingErrs = snapshot.findPendingErrs
}

type fakeApprovalRepoSnapshot struct {
	byTaskID        map[string]approval.Approval
	saved           approval.Approval
	saveCount       int
	findLatestErr   error
	findPendingErrs []error
}

func (f *fakeApprovalRepo) FindLatestByTask(taskID string) (approval.Approval, error) {
	if f.findLatestErr != nil {
		return approval.Approval{}, f.findLatestErr
	}

	value, ok := f.byTaskID[taskID]
	if !ok {
		return approval.Approval{}, sql.ErrNoRows
	}

	return value, nil
}

type fakeTransactor struct {
	tasks           *fakeTaskRepo
	approvals       *fakeApprovalRepo
	runs            *fakeRunRepo
	artifacts       *fakeArtifactRepo
	failCommitCount int
}

func newFakeTransactor(tasks *fakeTaskRepo, approvals *fakeApprovalRepo, runs *fakeRunRepo, artifacts *fakeArtifactRepo) *fakeTransactor {
	return &fakeTransactor{
		tasks:     tasks,
		approvals: approvals,
		runs:      runs,
		artifacts: artifacts,
	}
}

func (f *fakeTransactor) WithinTransaction(ctx context.Context, fn func(context.Context, ports.TransactionRepositories) error) error {
	tasksSnapshot := f.tasks.snapshot()
	approvalsSnapshot := f.approvals.snapshot()
	runsSnapshot := f.runs.snapshot()
	artifactsSnapshot := f.artifacts.snapshot()

	err := fn(ctx, ports.TransactionRepositories{
		Tasks:     f.tasks,
		Runs:      f.runs,
		Approvals: f.approvals,
		Artifacts: f.artifacts,
	})
	if err != nil {
		f.tasks.restore(tasksSnapshot)
		f.approvals.restore(approvalsSnapshot)
		f.runs.restore(runsSnapshot)
		f.artifacts.restore(artifactsSnapshot)
		return err
	}
	if f.failCommitCount > 0 {
		f.failCommitCount--
		f.tasks.restore(tasksSnapshot)
		f.approvals.restore(approvalsSnapshot)
		f.runs.restore(runsSnapshot)
		f.artifacts.restore(artifactsSnapshot)
		return errors.New("commit failed")
	}

	return nil
}
