// Package fsobserver implements an audit Observer that writes audit messages
// to a local file as newline-delimited JSON.
package fsobserver

import (
	"bufio"
	"encoding/json"
	"os"
	"sync"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/model"
	"github.com/rs/zerolog"
)

// New creates a new file system audit observer that appends audit messages
// to the specified file path. The file is created if it does not exist.
func New(logger *zerolog.Logger, filePath string) (*Observer, error) {
	// #nosec G304
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	return &Observer{
		logger: logger, file: file,
		writer: bufio.NewWriter(file),
	}, nil
}

// Observer implements the auditor.Observer interface by appending audit messages
// as JSON lines to a local file.
type Observer struct {
	logger *zerolog.Logger
	writer *bufio.Writer
	file   *os.File
	mu     sync.Mutex
}

// HandleMessage marshals the audit message to JSON and appends it to the
// configured file. Errors are logged but not propagated.
func (o *Observer) HandleMessage(msg model.AuditMsg) {
	o.mu.Lock()
	defer o.mu.Unlock()

	encoder := json.NewEncoder(o.writer)
	if err := encoder.Encode(&msg); err != nil {
		o.logger.Error().Err(err).Msg("error while audit encode message")
		return
	}
	if err := o.writer.Flush(); err != nil {
		o.logger.Error().Err(err).Msg("error while flush audit file")
	}
}
