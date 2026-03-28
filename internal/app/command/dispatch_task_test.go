package command

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/sine-io/foreman/internal/domain/approval"
	domainpolicy "github.com/sine-io/foreman/internal/domain/policy"
	"github.com/sine-io/foreman/internal/domain/task"
	"github.com/sine-io/foreman/internal/ports"
	"github.com/stretchr/testify/require"
)

func TestDispatchAcquiresRepoLeaseAndStartsRun(t *testing.T) {
	tasks := newFakeTaskRepo()
	repoTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "Implement board query", "repo:project-1")
	require.NoError(t, tasks.Save(repoTask))

	leases := &fakeLeaseRepo{}
	runs := &fakeRunRepo{}
	artifacts := &fakeArtifactRepo{}
	tx := newFakeTransactor(tasks, &fakeApprovalRepo{}, runs, artifacts)
	handler := NewDispatchTaskHandler(
		tx,
		tasks,
		leases,
		fakePolicy{decision: domainpolicy.Decision{}},
		&fakeRunner{},
		tx.approvals,
		runs,
		artifacts,
	)

	out, err := handler.Handle(DispatchTaskCommand{TaskID: "task-1"})
	require.NoError(t, err)
	require.Equal(t, "completed", out.RunState)
	require.Equal(t, "repo:project-1", leases.scopeKey)
	require.Equal(t, "repo:project-1", leases.releasedScopeKey)
	require.Equal(t, "task-1", runs.saved.TaskID)

	savedTask, err := tasks.Get("task-1")
	require.NoError(t, err)
	require.Equal(t, task.TaskStateCompleted, savedTask.State)
}

func TestDispatchCreatesApprovalWhenRiskyActionDetected(t *testing.T) {
	tasks := newFakeTaskRepo()
	riskyTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "git push origin main", "repo:project-1")
	require.NoError(t, tasks.Save(riskyTask))

	approvals := &fakeApprovalRepo{}
	tx := newFakeTransactor(tasks, approvals, &fakeRunRepo{}, &fakeArtifactRepo{})
	handler := NewDispatchTaskHandler(
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
		tx.runs,
		tx.artifacts,
	)

	out, err := handler.Handle(DispatchTaskCommand{TaskID: "task-1"})
	require.NoError(t, err)
	require.Equal(t, "waiting_approval", out.TaskState)
	require.Equal(t, "git push origin main requires approval", approvals.saved.Reason)
	require.Equal(t, 1, approvals.saveCount)
}

func TestDispatchCopiesPolicyMetadataIntoCreatedApproval(t *testing.T) {
	tasks := newFakeTaskRepo()
	riskyTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "git push origin main", "repo:project-1")
	require.NoError(t, tasks.Save(riskyTask))

	approvals := &fakeApprovalRepo{}
	tx := newFakeTransactor(tasks, approvals, &fakeRunRepo{}, &fakeArtifactRepo{})
	handler := NewDispatchTaskHandler(
		tx,
		tasks,
		&fakeLeaseRepo{},
		fakePolicy{
			decision: domainpolicy.Decision{
				RequiresApproval: true,
				Reason:           "git push origin main requires approval",
				RiskLevel:        approval.RiskHigh,
				PolicyRule:       "strict.git_push",
			},
		},
		&fakeRunner{},
		approvals,
		tx.runs,
		tx.artifacts,
	)

	out, err := handler.Handle(DispatchTaskCommand{TaskID: "task-1"})
	require.NoError(t, err)
	require.Equal(t, "waiting_approval", out.TaskState)
	require.Equal(t, approval.RiskHigh, approvals.saved.RiskLevel)
	require.Equal(t, "strict.git_push", approvals.saved.PolicyRule)
}

func TestDispatchReusesExistingPendingApprovalWhenRetried(t *testing.T) {
	tasks := newFakeTaskRepo()
	riskyTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "git push origin main", "repo:project-1")
	require.NoError(t, tasks.Save(riskyTask))

	approvals := &fakeApprovalRepo{}
	tx := newFakeTransactor(tasks, approvals, &fakeRunRepo{}, &fakeArtifactRepo{})
	handler := NewDispatchTaskHandler(
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
		tx.runs,
		tx.artifacts,
	)

	first, err := handler.Handle(DispatchTaskCommand{TaskID: "task-1"})
	require.NoError(t, err)
	second, err := handler.Handle(DispatchTaskCommand{TaskID: "task-1"})
	require.NoError(t, err)

	require.Equal(t, first.ApprovalID, second.ApprovalID)
	require.Equal(t, 1, approvals.saveCount)
}

func TestDispatchRunsApprovedTaskEvenWhenInitialTaskReadIsStale(t *testing.T) {
	tasks := newFakeTaskRepo()
	riskyTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "git push origin main", "repo:project-1")
	riskyTask.State = task.TaskStateWaitingApproval
	require.NoError(t, tasks.Save(riskyTask))

	approvals := &fakeApprovalRepo{
		byTaskID: map[string]approval.Approval{
			"task-1": {
				ID:     "approval-1",
				TaskID: "task-1",
				Reason: "git push origin main requires approval",
				Status: approval.StatusApproved,
			},
		},
	}
	tasks.afterGet = func(value task.Task) {
		if value.ID != "task-1" || value.State != task.TaskStateWaitingApproval {
			return
		}
		updated := value
		updated.State = task.TaskStateLeased
		require.NoError(t, tasks.Save(updated))
		tasks.afterGet = nil
	}

	tx := newFakeTransactor(tasks, approvals, &fakeRunRepo{}, &fakeArtifactRepo{})
	runner := &fakeRunner{}
	handler := NewDispatchTaskHandler(
		tx,
		tasks,
		&fakeLeaseRepo{},
		fakePolicy{
			decision: domainpolicy.Decision{
				RequiresApproval: true,
				Reason:           "git push origin main requires approval",
			},
		},
		runner,
		approvals,
		tx.runs,
		tx.artifacts,
	)

	out, err := handler.Handle(DispatchTaskCommand{TaskID: "task-1"})
	require.NoError(t, err)
	require.Equal(t, "completed", out.RunState)
	require.Equal(t, 1, runner.dispatchCount)
	require.Equal(t, 0, approvals.saveCount)
}

func TestDispatchDoesNotHideUnexpectedApprovalSaveErrors(t *testing.T) {
	tasks := newFakeTaskRepo()
	riskyTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "git push origin main", "repo:project-1")
	require.NoError(t, tasks.Save(riskyTask))

	approvals := &fakeApprovalRepo{
		byTaskID: map[string]approval.Approval{},
		saveErr:  errors.New("db down"),
	}
	tx := newFakeTransactor(tasks, approvals, &fakeRunRepo{}, &fakeArtifactRepo{})
	handler := NewDispatchTaskHandler(
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
		tx.runs,
		tx.artifacts,
	)

	_, err := handler.Handle(DispatchTaskCommand{TaskID: "task-1"})
	require.EqualError(t, err, "db down")
}

func TestDispatchDoesNotLeaveDuplicatePendingApprovalsUnderRetry(t *testing.T) {
	tasks := newFakeTaskRepo()
	riskyTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "git push origin main", "repo:project-1")
	require.NoError(t, tasks.Save(riskyTask))

	approvals := &fakeApprovalRepo{}
	tx := newFakeTransactor(tasks, approvals, &fakeRunRepo{}, &fakeArtifactRepo{})
	tx.failCommitCount = 1

	handler := NewDispatchTaskHandler(
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
		tx.runs,
		tx.artifacts,
	)

	_, err := handler.Handle(DispatchTaskCommand{TaskID: "task-1"})
	require.Error(t, err)

	out, err := handler.Handle(DispatchTaskCommand{TaskID: "task-1"})
	require.NoError(t, err)
	require.Equal(t, "waiting_approval", out.TaskState)
	require.Equal(t, 1, approvals.pendingCount())
	require.Equal(t, 1, approvals.saveCount)

	savedTask, err := tasks.Get("task-1")
	require.NoError(t, err)
	require.Equal(t, task.TaskStateWaitingApproval, savedTask.State)
}

func TestDispatchIndexesAssistantSummaryArtifact(t *testing.T) {
	tasks := newFakeTaskRepo()
	repoTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "Implement board query", "repo:project-1")
	require.NoError(t, tasks.Save(repoTask))

	tx := newFakeTransactor(tasks, &fakeApprovalRepo{}, &fakeRunRepo{}, &fakeArtifactRepo{})
	handler := NewDispatchTaskHandler(
		tx,
		tasks,
		&fakeLeaseRepo{},
		fakePolicy{decision: domainpolicy.Decision{}},
		&fakeRunner{},
		tx.approvals,
		tx.runs,
		tx.artifacts,
	)

	out, err := handler.Handle(DispatchTaskCommand{TaskID: "task-1"})
	require.NoError(t, err)
	require.NotEmpty(t, out.ArtifactIDs)
}

func TestDispatchReleasesLeaseIfPersistenceFailsAfterRunnerReturns(t *testing.T) {
	tasks := newFakeTaskRepo()
	repoTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "Implement board query", "repo:project-1")
	require.NoError(t, tasks.Save(repoTask))

	leases := &fakeLeaseRepo{}
	runner := &fakeRunner{}
	runs := &fakeRunRepo{}
	artifacts := &fakeArtifactRepo{}
	tx := newFakeTransactor(tasks, &fakeApprovalRepo{}, runs, artifacts)
	tx.failCommitCount = 1

	handler := NewDispatchTaskHandler(
		tx,
		tasks,
		leases,
		fakePolicy{decision: domainpolicy.Decision{}},
		runner,
		tx.approvals,
		runs,
		artifacts,
	)

	_, err := handler.Handle(DispatchTaskCommand{TaskID: "task-1"})
	require.Error(t, err)
	require.Equal(t, "repo:project-1", leases.releasedScopeKey)
	require.Equal(t, 1, runner.dispatchCount)

	_, err = runs.FindByTask("task-1")
	require.ErrorIs(t, err, sql.ErrNoRows)

	savedTask, err := tasks.Get("task-1")
	require.NoError(t, err)
	require.Equal(t, task.TaskStateReady, savedTask.State)
}

func TestDispatchRetriesLeaseReleaseAfterCompletedRunWasPersisted(t *testing.T) {
	tasks := newFakeTaskRepo()
	repoTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "Implement board query", "repo:project-1")
	require.NoError(t, tasks.Save(repoTask))

	leases := &fakeLeaseRepo{
		releaseErrs: []error{errors.New("release failed")},
	}
	runner := &fakeRunner{}
	runs := &fakeRunRepo{}
	artifacts := &fakeArtifactRepo{}
	tx := newFakeTransactor(tasks, &fakeApprovalRepo{}, runs, artifacts)

	handler := NewDispatchTaskHandler(
		tx,
		tasks,
		leases,
		fakePolicy{decision: domainpolicy.Decision{}},
		runner,
		tx.approvals,
		runs,
		artifacts,
	)

	_, err := handler.Handle(DispatchTaskCommand{TaskID: "task-1"})
	require.EqualError(t, err, "release failed")
	require.Equal(t, 1, runner.dispatchCount)
	require.Equal(t, 1, leases.releaseCount)

	out, err := handler.Handle(DispatchTaskCommand{TaskID: "task-1"})
	require.NoError(t, err)
	require.Equal(t, "completed", out.RunState)
	require.Equal(t, 1, runner.dispatchCount)
	require.Equal(t, 2, leases.releaseCount)
	require.Equal(t, "repo:project-1", leases.releasedScopeKey)
}

func TestDispatchDoesNotReinvokeRunnerWhenTaskAlreadyHasPersistedRun(t *testing.T) {
	tasks := newFakeTaskRepo()
	repoTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "Implement board query", "repo:project-1")
	repoTask.State = task.TaskStateCompleted
	require.NoError(t, tasks.Save(repoTask))

	runs := &fakeRunRepo{
		saved: ports.Run{
			ID:                   "run-1",
			TaskID:               "task-1",
			RunnerKind:           "codex",
			State:                "completed",
			AssistantSummaryPath: "artifacts/tasks/task-1/assistant_summary.txt",
		},
	}
	artifacts := &fakeArtifactRepo{}
	leases := &fakeLeaseRepo{}
	runner := &fakeRunner{}
	tx := newFakeTransactor(tasks, &fakeApprovalRepo{}, runs, artifacts)

	handler := NewDispatchTaskHandler(
		tx,
		tasks,
		leases,
		fakePolicy{decision: domainpolicy.Decision{}},
		runner,
		tx.approvals,
		runs,
		artifacts,
	)

	out, err := handler.Handle(DispatchTaskCommand{TaskID: "task-1"})
	require.NoError(t, err)
	require.Equal(t, "completed", out.TaskState)
	require.Equal(t, "completed", out.RunState)
	require.Equal(t, 0, runner.dispatchCount)
	require.Zero(t, leases.acquireCount)
}

func TestDispatchDoesNotReleaseAnotherTasksActiveLeaseOnCompletedRunRetry(t *testing.T) {
	tasks := newFakeTaskRepo()
	completedTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "Implement board query", "repo:project-1")
	completedTask.State = task.TaskStateCompleted
	require.NoError(t, tasks.Save(completedTask))

	runs := &fakeRunRepo{
		saved: ports.Run{
			ID:     "run-1",
			TaskID: "task-1",
			State:  "completed",
		},
	}
	leases := &fakeLeaseRepo{
		activeByScope: map[string]string{
			"repo:project-1": "task-2",
		},
	}
	tx := newFakeTransactor(tasks, &fakeApprovalRepo{}, runs, &fakeArtifactRepo{})
	handler := NewDispatchTaskHandler(
		tx,
		tasks,
		leases,
		fakePolicy{decision: domainpolicy.Decision{}},
		&fakeRunner{},
		tx.approvals,
		runs,
		tx.artifacts,
	)

	out, err := handler.Handle(DispatchTaskCommand{TaskID: "task-1"})
	require.NoError(t, err)
	require.Equal(t, "completed", out.RunState)
	require.Equal(t, "task-2", leases.activeByScope["repo:project-1"])
}

func TestDispatchReturnsApprovalLookupErrorAfterPendingConflict(t *testing.T) {
	tasks := newFakeTaskRepo()
	riskyTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "git push origin main", "repo:project-1")
	require.NoError(t, tasks.Save(riskyTask))

	approvals := &fakeApprovalRepo{
		saveErr:         ports.ErrPendingApprovalConflict,
		findPendingErrs: []error{sql.ErrNoRows, errors.New("lookup failed")},
	}
	tx := newFakeTransactor(tasks, approvals, &fakeRunRepo{}, &fakeArtifactRepo{})
	handler := NewDispatchTaskHandler(
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
		tx.runs,
		tx.artifacts,
	)

	_, err := handler.Handle(DispatchTaskCommand{TaskID: "task-1"})
	require.EqualError(t, err, "lookup failed")
}

func TestDispatchPreservesConcurrentTaskMetadataUpdates(t *testing.T) {
	tasks := newFakeTaskRepo()
	repoTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "Implement board query", "repo:project-1")
	repoTask.Priority = 10
	require.NoError(t, tasks.Save(repoTask))

	runner := &fakeRunner{
		onDispatch: func() {
			current, err := tasks.Get("task-1")
			require.NoError(t, err)
			current.Summary = "Updated summary"
			current.Priority = 42
			require.NoError(t, tasks.Save(current))
		},
	}
	tx := newFakeTransactor(tasks, &fakeApprovalRepo{}, &fakeRunRepo{}, &fakeArtifactRepo{})
	handler := NewDispatchTaskHandler(
		tx,
		tasks,
		&fakeLeaseRepo{},
		fakePolicy{decision: domainpolicy.Decision{}},
		runner,
		tx.approvals,
		tx.runs,
		tx.artifacts,
	)

	out, err := handler.Handle(DispatchTaskCommand{TaskID: "task-1"})
	require.NoError(t, err)
	require.Equal(t, "completed", out.RunState)

	savedTask, err := tasks.Get("task-1")
	require.NoError(t, err)
	require.Equal(t, "Updated summary", savedTask.Summary)
	require.Equal(t, 42, savedTask.Priority)
	require.Equal(t, task.TaskStateCompleted, savedTask.State)
}

func TestDispatchDoesNotReinvokeRunnerWhenRunAppearsAfterLeaseAcquire(t *testing.T) {
	tasks := newFakeTaskRepo()
	repoTask := task.NewTask("task-1", "module-1", task.TaskTypeWrite, "Implement board query", "repo:project-1")
	require.NoError(t, tasks.Save(repoTask))

	runs := &fakeRunRepo{}
	leases := &fakeLeaseRepo{
		onAcquire: func() {
			_ = runs.Save(ports.Run{
				ID:     "run-1",
				TaskID: "task-1",
				State:  "completed",
			})
		},
	}
	runner := &fakeRunner{}
	tx := newFakeTransactor(tasks, &fakeApprovalRepo{}, runs, &fakeArtifactRepo{})

	handler := NewDispatchTaskHandler(
		tx,
		tasks,
		leases,
		fakePolicy{decision: domainpolicy.Decision{}},
		runner,
		tx.approvals,
		runs,
		tx.artifacts,
	)

	out, err := handler.Handle(DispatchTaskCommand{TaskID: "task-1"})
	require.NoError(t, err)
	require.Equal(t, "completed", out.RunState)
	require.Equal(t, 0, runner.dispatchCount)
	require.Equal(t, 1, leases.releaseCount)
}

type fakeLeaseRepo struct {
	taskID           string
	scopeKey         string
	releasedScopeKey string
	acquireCount     int
	releaseCount     int
	onAcquire        func()
	releaseErrs      []error
	activeByScope    map[string]string
}

func (f *fakeLeaseRepo) Acquire(taskID, scopeKey string) error {
	f.taskID = taskID
	f.scopeKey = scopeKey
	f.acquireCount++
	if f.activeByScope == nil {
		f.activeByScope = map[string]string{}
	}
	f.activeByScope[scopeKey] = taskID
	if f.onAcquire != nil {
		f.onAcquire()
	}
	return nil
}

func (f *fakeLeaseRepo) Release(taskID, scopeKey string) error {
	f.releasedScopeKey = scopeKey
	f.releaseCount++
	if f.activeByScope != nil {
		activeTaskID, ok := f.activeByScope[scopeKey]
		if ok && activeTaskID == taskID {
			delete(f.activeByScope, scopeKey)
		}
	}
	if len(f.releaseErrs) > 0 {
		err := f.releaseErrs[0]
		f.releaseErrs = f.releaseErrs[1:]
		return err
	}
	return nil
}

type fakePolicy struct {
	decision domainpolicy.Decision
}

func (f fakePolicy) Evaluate(action string) domainpolicy.Decision {
	return f.decision
}

type fakeRunner struct {
	state         string
	dispatchCount int
	dispatchErr   error
	onDispatch    func()
}

func (f *fakeRunner) Dispatch(req ports.RunRequest) (ports.Run, error) {
	f.dispatchCount++
	if f.onDispatch != nil {
		f.onDispatch()
	}
	if f.dispatchErr != nil {
		return ports.Run{}, f.dispatchErr
	}
	state := f.state
	if state == "" {
		state = "completed"
	}
	return ports.Run{
		ID:                   "run-1",
		TaskID:               req.TaskID,
		RunnerKind:           "codex",
		State:                state,
		AssistantSummaryPath: "artifacts/tasks/task-1/assistant_summary.txt",
	}, nil
}

func (f *fakeRunner) Observe(runID string) (ports.Run, error) {
	return ports.Run{}, nil
}

func (f *fakeRunner) Stop(runID string) error {
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

func (f *fakeRunRepo) FindByTask(taskID string) (ports.Run, error) {
	if f.saved.TaskID != taskID {
		return ports.Run{}, sql.ErrNoRows
	}
	return f.saved, nil
}

func (f *fakeRunRepo) snapshot() fakeRunRepoSnapshot {
	return fakeRunRepoSnapshot{saved: f.saved}
}

func (f *fakeRunRepo) restore(snapshot fakeRunRepoSnapshot) {
	f.saved = snapshot.saved
}

type fakeRunRepoSnapshot struct {
	saved ports.Run
}

type fakeArtifactRepo struct {
	nextID    string
	created   []ports.ArtifactRecord
	createErr error
}

func (f *fakeArtifactRepo) Create(taskID, kind, path string) (string, error) {
	if f.createErr != nil {
		return "", f.createErr
	}
	if f.nextID == "" {
		f.nextID = "artifact-1"
	}
	f.created = append(f.created, ports.ArtifactRecord{
		ID:     f.nextID,
		TaskID: taskID,
		Kind:   kind,
		Path:   path,
	})
	return f.nextID, nil
}

func (f *fakeArtifactRepo) Get(id string) (ports.ArtifactRecord, error) {
	if id == "" {
		return ports.ArtifactRecord{}, sql.ErrNoRows
	}
	return ports.ArtifactRecord{ID: id}, nil
}

func (f *fakeArtifactRepo) snapshot() fakeArtifactRepoSnapshot {
	created := make([]ports.ArtifactRecord, len(f.created))
	copy(created, f.created)
	return fakeArtifactRepoSnapshot{
		nextID:    f.nextID,
		created:   created,
		createErr: f.createErr,
	}
}

func (f *fakeArtifactRepo) restore(snapshot fakeArtifactRepoSnapshot) {
	f.nextID = snapshot.nextID
	f.created = snapshot.created
	f.createErr = snapshot.createErr
}

type fakeArtifactRepoSnapshot struct {
	nextID    string
	created   []ports.ArtifactRecord
	createErr error
}
