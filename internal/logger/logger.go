package logger

import (
	"os"

	"github.com/rs/zerolog"
)

var Logger *zerolog.Logger

func InitLogger(level zerolog.Level) {
	log := zerolog.New(os.Stdout).With().Timestamp().Logger()
	Logger = &log
}
