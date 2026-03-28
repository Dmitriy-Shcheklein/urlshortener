package handler

import (
	"io"
	"log"
	"net/http"
	"strings"
)

type Service interface {
	GetById(ID string) ([]byte, error)
	CreateShort(originalUrl []byte) ([]byte, error)
}

type Handler struct {
	service Service
}

func New(service Service) *Handler {
	if service == nil {
		panic("Handler service must be not nil")
	}
	return new(Handler{service: service})
}

func (h *Handler) GetByd(writer http.ResponseWriter, request *http.Request) {
	strID := strings.TrimPrefix(request.URL.Path, "/")

	if strID == "" {
		http.Error(writer, "ID parameter is required", http.StatusBadRequest)
		return
	}

	link, err := h.service.GetById(strID)
	if err != nil {
		http.Error(writer, "Error while getting url", http.StatusBadRequest)
		return
	}

	writer.Header().Add("Content-Type", "text/plain")
	writer.Header().Add("Location", string(link))
	writer.WriteHeader(http.StatusTemporaryRedirect)
}

func (h *Handler) CreateShort(writer http.ResponseWriter, request *http.Request) {
	if contentType := request.Header.Get("Content-Type"); contentType != "text/plain" {
		http.Error(writer, "Invalid content-type", http.StatusBadRequest)
		return
	}
	body, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, "Error while parse body", http.StatusBadRequest)
		return
	}
	defer request.Body.Close()
	if len(body) == 0 {
		http.Error(writer, "Empty body", http.StatusBadRequest)
		return
	}

	url, err := h.service.CreateShort(body)
	if err != nil {
		http.Error(writer, "Error while create short url", http.StatusBadRequest)
		log.Printf("error: %s", err)
		return
	}

	writer.Header().Add("Content-Type", "text/plain")
	writer.WriteHeader(http.StatusCreated)

	_, err = writer.Write(url)
	if err != nil {
		http.Error(writer, "Error while write body", http.StatusBadRequest)
		return
	}
}
