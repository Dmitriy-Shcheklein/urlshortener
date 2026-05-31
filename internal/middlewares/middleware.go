package middlewares

import "github.com/rs/zerolog"

type Config interface {
	GetSalt() []byte
}

type AppMiddleware struct {
	logger *zerolog.Logger
	cfg    Config
}

func NewAppMiddleware(logger *zerolog.Logger, cfg Config) *AppMiddleware {
	return &AppMiddleware{logger: logger, cfg: cfg}
}
