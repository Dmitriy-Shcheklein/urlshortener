// Package logger provides a global zerolog logger instance for the application.
package logger

import (
	"os"

	"github.com/rs/zerolog"
)

// Logger is the global logger instance used throughout the application.
// It must be initialized via [InitLogger] before use.
var Logger *zerolog.Logger

// InitLogger configures the global logger with the specified log level
// and sets it to write to stdout with timestamps.
func InitLogger(level zerolog.Level) {
	zerolog.SetGlobalLevel(level)
	log := zerolog.New(os.Stdout).With().Timestamp().Logger()
	Logger = &log
}
