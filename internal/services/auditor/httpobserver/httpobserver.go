package httpobserver

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/model"
	"github.com/rs/zerolog"
)

type HttpClient interface {
	Post(url string, contentType string, body io.Reader) (resp *http.Response, err error)
}

func New(logger *zerolog.Logger, client HttpClient, auditURL string) *Observer {
	return &Observer{logger: logger, httpClient: client, url: auditURL}
}

type Observer struct {
	logger     *zerolog.Logger
	httpClient HttpClient
	url        string
}

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
