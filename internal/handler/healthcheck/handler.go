// Package healthcheck provides HTTP handlers for service health checks.
package healthcheck

import (
	"errors"
	"net/http"
)

// Service defines the health check operations used by the handler.
type Service interface {
	// PingDB checks the database connectivity.
	PingDB() error
}

// Handler implements the /ping health check endpoint.
type Handler struct {
	service Service
}

// New creates a new health check Handler. The service parameter must not be nil.
func New(service Service) (*Handler, error) {
	handler := &Handler{}
	if service == nil {
		return handler, errors.New("service must be not nil")
	}
	handler.service = service
	return handler, nil
}

// PingDB handles GET /ping requests. It checks database connectivity
// and responds with 200 OK on success or 500 Internal Server Error on failure.
func (h *Handler) PingDB(writer http.ResponseWriter, _ *http.Request) {
	if err := h.service.PingDB(); err != nil {
		http.Error(writer, "error while ping DB", http.StatusInternalServerError)
	} else {
		writer.WriteHeader(http.StatusOK)
	}
}
