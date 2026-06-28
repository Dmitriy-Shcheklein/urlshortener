// Package fsobserver implements an audit Observer that writes audit messages
// to a local file as newline-delimited JSON.
package fsobserver

import (
	"encoding/json"
	"os"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/model"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// New creates a new file system audit observer that appends audit messages
// to the specified file path. The file is created if it does not exist.
func New(logger *zerolog.Logger, filePath string) *Observer {
	return &Observer{logger: logger, path: filePath}
}

// Observer implements the auditor.Observer interface by appending audit messages
// as JSON lines to a local file.
type Observer struct {
	logger *zerolog.Logger
	path   string
}

// HandleMessage marshals the audit message to JSON and appends it to the
// configured file. Errors are logged but not propagated.
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
