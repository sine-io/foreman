package logging

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Configure() zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339

	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
	log.Logger = logger

	return logger
}
