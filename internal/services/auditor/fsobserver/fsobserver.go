package fsobserver

import (
	"encoding/json"
	"os"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/model"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func New(logger *zerolog.Logger, filePath string) *Observer {
	return &Observer{logger: logger, path: filePath}
}

type Observer struct {
	logger *zerolog.Logger
	path   string
}

func (o *Observer) HandleMessage(msg model.AuditMsg) {
	file, err := os.OpenFile(o.path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		o.logger.Error().Err(err).Msg("error while audit open files")
		return
	}
	defer func() {
		if err = file.Close(); err != nil {
			log.Err(err).Msg("error while close file")
		}
	}()
	encoder := json.NewEncoder(file)
	if err = encoder.Encode(&msg); err != nil {
		o.logger.Error().Err(err).Msg("error while audit encode message")
		return
	}
}
