package bootstrap

import "context"

type App interface {
	Serve(context.Context) error
}

type app struct {
	Config Config
}

func BuildApp(cfg Config) (App, error) {
	if err := PrepareRuntime(cfg); err != nil {
		return nil, err
	}

	return &app{Config: cfg}, nil
}

func (a *app) Serve(context.Context) error {
	return nil
}
