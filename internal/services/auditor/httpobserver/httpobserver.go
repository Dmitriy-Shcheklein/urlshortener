// Package httpobserver implements an audit Observer that sends audit messages
// to a remote HTTP endpoint via POST requests.
package httpobserver

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/model"
	"github.com/rs/zerolog"
)

// HttpClient defines the HTTP client interface used by the observer.
// This allows for easy mocking in tests.
type HttpClient interface {
	// Post sends an HTTP POST request to the specified URL.
	Post(url string, contentType string, body io.Reader) (resp *http.Response, err error)
}

// New creates a new HTTP audit observer that sends audit messages to the specified URL.
// The client is used for making HTTP requests; auditURL is the destination endpoint.
func New(logger *zerolog.Logger, client HttpClient, auditURL string) *Observer {
	return &Observer{logger: logger, httpClient: client, url: auditURL}
}

// Observer implements the auditor.Observer interface by sending audit messages
// as JSON POST requests to a configured HTTP endpoint.
type Observer struct {
	logger     *zerolog.Logger
	httpClient HttpClient
	url        string
}

// HandleMessage marshals the audit message to JSON and sends it to the
// configured HTTP endpoint. Errors are logged but not propagated.
func (o *Observer) HandleMessage(msg model.AuditMsg) {
	jsonData, err := json.Marshal(&msg)
	if err != nil {
		o.logger.Error().Err(err).Msg("error while marshalling http audit message")
	}
	_, err = o.httpClient.Post(o.url, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		o.logger.Error().Err(err).Msg("error while send http audit message")
	}
}
