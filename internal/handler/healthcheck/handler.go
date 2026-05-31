package healthcheck

import (
	"errors"
	"net/http"
)

type Service interface {
	PingDB() error
}

type Handler struct {
	service Service
}

func New(service Service) (*Handler, error) {
	handler := &Handler{}
	if service == nil {
		return handler, errors.New("service must be not nil")
	}
	handler.service = service
	return handler, nil
}

func (h *Handler) PingDB(writer http.ResponseWriter, _ *http.Request) {
	if err := h.service.PingDB(); err != nil {
		http.Error(writer, "error while ping DB", http.StatusInternalServerError)
	} else {
		writer.WriteHeader(http.StatusOK)
	}
}
