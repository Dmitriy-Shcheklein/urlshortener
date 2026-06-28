// Package middlewares provides HTTP middleware for authentication, logging,
// and gzip compression for the URL shortener service.
package middlewares

import (
	"github.com/rs/zerolog"
)

// Config provides application configuration needed by the middleware layer.
type Config interface {
	// GetSalt returns the secret key used for JWT token signing.
	GetSalt() []byte
}

// AppMiddleware holds shared dependencies for all middleware functions.
// Use [NewAppMiddleware] to create an instance.
type AppMiddleware struct {
	logger *zerolog.Logger
	cfg    Config
}

// NewAppMiddleware creates a new AppMiddleware with the given logger and configuration.
func NewAppMiddleware(logger *zerolog.Logger, cfg Config) *AppMiddleware {
	return &AppMiddleware{logger: logger, cfg: cfg}
}
