package ports

type RunRequest struct {
	TaskID     string
	Command    string
	WriteScope string
}

type Runner interface {
	Dispatch(RunRequest) (Run, error)
	Observe(runID string) (Run, error)
	Stop(runID string) error
}
