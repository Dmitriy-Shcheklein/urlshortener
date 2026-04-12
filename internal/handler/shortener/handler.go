package shortener

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/logger"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/model"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/repository/postgres"
	"github.com/go-playground/validator/v10"
)

type Service interface {
	GetByID(ID string) ([]byte, error)
	CreateShort(originalURL []byte) ([]byte, error)
	CreateMany(values []model.CreateManyBodyRaw) ([]model.CreateManyResponseRaw, error)
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
	headers := map[string]string{"Content-Type": "text/plain"}
	if conflictError, ok := errors.AsType[*postgres.ConflictError](err); ok {
		prepareResponse(
			writer, headers, http.StatusConflict, h.prepareRequest(request.Host, conflictError.Shorten),
		)
		return
	}

	if err != nil {
		http.Error(writer, "Error while create short url", http.StatusInternalServerError)
		return
	}
	result := h.prepareRequest(request.Host, short)
	prepareResponse(writer, headers, http.StatusCreated, result)
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
	headers := map[string]string{"Content-Type": "application/json"}
	if conflictError, ok := errors.AsType[*postgres.ConflictError](err); ok {
		h.prepareJSONResponse(writer, request.Host, conflictError.Shorten, http.StatusConflict, headers)
		return
	}
	if err != nil {
		http.Error(writer, "Error while create short url", http.StatusInternalServerError)
		return
	}
	h.prepareJSONResponse(writer, request.Host, short, http.StatusCreated, headers)
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

	var deserialized []model.CreateManyBodyRaw
	validate := validator.New()

	err = json.Unmarshal(body, &deserialized)
	if err != nil {
		http.Error(writer, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	if len(body) == 0 {
		http.Error(writer, "empty body values", http.StatusBadRequest)
		return
	}
	for i := range deserialized {
		if err = validate.Struct(deserialized[i]); err != nil {
			logger.Logger.Error().Err(err).Msg("error while validate body\n")
			http.Error(writer, "Error while validate body", http.StatusBadRequest)
			return
		}
	}

	shorts, err := h.service.CreateMany(deserialized)
	if err != nil {
		logger.Logger.Error().Err(err).Msg("error while create short url\n")
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

	headers := map[string]string{"Content-Type": "application/json"}
	prepareResponse(writer, headers, http.StatusCreated, resp)
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

func (h *Handler) prepareJSONResponse(
	w http.ResponseWriter, host string, res []byte, status int, headers map[string]string,
) {
	result := h.prepareRequest(host, res)
	response := &CreateShortResponse{
		Result: string(result),
	}

	resp, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	prepareResponse(w, headers, status, resp)
}

func prepareResponse(w http.ResponseWriter, headers map[string]string, statusCode int, body []byte) {
	for key, value := range headers {
		w.Header().Add(key, value)
	}
	w.WriteHeader(statusCode)
	// #nosec G705
	_, err := w.Write(body)
	if err != nil {
		http.Error(w, "Error while write body", http.StatusInternalServerError)
		return
	}
}
