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
type Config interface {
	GetBaseAddress() []byte
}

type Handler struct {
	service Service
	config  Config
}

func New(service Service, config Config) *Handler {
	if service == nil {
		panic("Handler service must be not nil")
	}
	if config == nil {
		panic("Handler config must be not nil")
	}

	return new(Handler{service: service, config: config})
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

	short, err := h.service.CreateShort(body)
	if err != nil {
		http.Error(writer, "Error while create short url", http.StatusBadRequest)
		log.Printf("error: %s", err)
		return
	}

	writer.Header().Add("Content-Type", "text/plain")
	writer.WriteHeader(http.StatusCreated)

	var result []byte

	if len(h.config.GetBaseAddress()) != 0 {
		result = append(h.config.GetBaseAddress(), "/"...)
		result = append(result, short...)
	} else {
		result = append(result, "http://"...)
		result = append(result, request.Host...)
		result = append(result, "/"...)
		result = append(result, short...)
	}

	_, err = writer.Write(result)
	if err != nil {
		http.Error(writer, "Error while write body", http.StatusBadRequest)
		return
	}
}
