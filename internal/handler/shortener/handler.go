package shortener

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/logger"
	"github.com/go-playground/validator/v10"
)

type Service interface {
	GetByID(ID string) ([]byte, error)
	CreateShort(originalURL []byte) ([]byte, error)
	CreateMany(values []CreateManyBodyRaw) ([]CreateManyResponseRaw, error)
}

type Config interface {
	GetBaseAddress() []byte
}

type Handler struct {
	service Service
	config  Config
}

type CreateShortBody struct {
	URL string `json:"url" validate:"required,min=3"`
}
type CreateShortResponse struct {
	Result string `json:"result"`
}
type CreateManyBodyRaw struct {
	CorrelationID string `json:"correlation_id" validate:"required"`
	OriginalUrl   string `json:"original_url" validate:"required"`
}
type CreateManyResponseRaw struct {
	CorrelationId string `json:"correlation_id" validate:"required"`
	ShortURL      string `json:"short_url" validate:"required"`
}

func New(service Service, config Config) (*Handler, error) {
	if service == nil {
		return &Handler{}, errors.New("handler service must be not nil")
	}
	if config == nil {
		return &Handler{}, errors.New("handler config must be not nil")
	}

	return &Handler{service: service, config: config}, nil
}

func (h *Handler) GetByID(writer http.ResponseWriter, request *http.Request) {
	strID := strings.TrimPrefix(request.URL.Path, "/")

	if strID == "" {
		http.Error(writer, "ID parameter is required", http.StatusBadRequest)
		return
	}

	link, err := h.service.GetByID(strID)
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
	defer func() {
		if err = request.Body.Close(); err != nil {
			logger.Logger.Error().Err(err).Msg("error while close body")
		}
	}()
	if len(body) == 0 {
		http.Error(writer, "Empty body", http.StatusBadRequest)
		return
	}
	short, err := h.service.CreateShort(body)
	if err != nil {
		http.Error(writer, "Error while create short url", http.StatusInternalServerError)
		return
	}
	result := h.prepareRequest(request.Host, short)

	writer.Header().Add("Content-Type", "text/plain")
	writer.WriteHeader(http.StatusCreated)
	// #nosec G705
	_, err = writer.Write(result)
	if err != nil {
		http.Error(writer, "Error while write body", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) CreateFromJSONBody(writer http.ResponseWriter, request *http.Request) {
	if contentType := request.Header.Get("Content-Type"); contentType != "application/json" {
		http.Error(writer, "Invalid content-type", http.StatusBadRequest)
		return
	}

	var body CreateShortBody
	validate := validator.New()

	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		http.Error(writer, "Error while decode body", http.StatusBadRequest)
		return
	}

	if err := validate.Struct(body); err != nil {
		http.Error(writer, "Error while validate body", http.StatusBadRequest)
		return
	}

	short, err := h.service.CreateShort([]byte(body.URL))
	if err != nil {
		http.Error(writer, "Error while create short url", http.StatusInternalServerError)
		return
	}
	result := h.prepareRequest(request.Host, short)
	response := &CreateShortResponse{
		Result: string(result),
	}

	resp, err := json.Marshal(response)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	writer.Header().Add("Content-Type", "application/json")
	writer.WriteHeader(http.StatusCreated)
	_, err = writer.Write(resp)
	if err != nil {
		http.Error(writer, "Error while write body", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) CreateMany(writer http.ResponseWriter, request *http.Request) {
	if contentType := request.Header.Get("Content-Type"); contentType != "application/json" {
		http.Error(writer, "Invalid content-type", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer func() {
		if err = request.Body.Close(); err != nil {
			logger.Logger.Error().Err(err).Msg("error while close body")
		}
	}()

	var deserialized []CreateManyBodyRaw
	validate := validator.New()

	err = json.Unmarshal(body, &deserialized)
	if err != nil {
		http.Error(writer, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	//if err := json.NewDecoder(request.Body).Decode(&deserialized); err != nil {
	//	http.Error(writer, "Error while decode body", http.StatusBadRequest)
	//	return
	//}
	if len(body) == 0 {
		http.Error(writer, "empty body values", http.StatusBadRequest)
		return
	}
	if err = validate.Struct(deserialized); err != nil {
		http.Error(writer, "Error while validate body", http.StatusBadRequest)
		return
	}

	shorts, err := h.service.CreateMany(deserialized)
	if err != nil {
		http.Error(writer, "Error while create short url", http.StatusInternalServerError)
		return
	}

	for i := range shorts {
		res := h.prepareRequest(request.Host, []byte(shorts[i].ShortURL))
		shorts[i].ShortURL = string(res)
	}

	resp, err := json.Marshal(shorts)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	writer.Header().Add("Content-Type", "application/json")
	writer.WriteHeader(http.StatusCreated)
	_, err = writer.Write(resp)
	if err != nil {
		http.Error(writer, "Error while write body", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) prepareRequest(host string, short []byte) []byte {
	var result []byte
	if len(h.config.GetBaseAddress()) != 0 {
		result = append(h.config.GetBaseAddress(), "/"...)
		result = append(result, short...)
	} else {
		result = append(result, "http://"...)
		result = append(result, host...)
		result = append(result, "/"...)
		result = append(result, short...)
	}

	return result
}
