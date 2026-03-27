package main

import (
	"github.com/sine-io/foreman/internal/adapters/cli"
	"github.com/sine-io/foreman/internal/bootstrap"
	"github.com/sine-io/foreman/internal/infrastructure/logging"
	"github.com/rs/zerolog/log"
)

func main() {
	logging.Configure()

	cfg, err := bootstrap.LoadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("load config")
	}

	app, err := bootstrap.BuildApp(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("build app")
	}

	if err := cli.NewRootCommand(app).Execute(); err != nil {
		log.Fatal().Err(err).Msg("run command")
	}
}
