// Package shortener provides HTTP handlers for the URL shortener REST API.
//
// The package exposes endpoints for creating, retrieving, and deleting shortened
// URLs. It supports both plain-text and JSON request/response formats, as well
// as batch operations.
package shortener

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/logger"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/model"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/repository/postgres"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
)

// Service defines the core URL shortening operations used by the HTTP handler.
// Implementations must be safe for concurrent use.
type Service interface {
	// GetByID retrieves the original URL by its short identifier.
	// Returns pgx.ErrNoRows if the link has been deleted or does not exist.
	GetByID(ID string) ([]byte, error)
	// CreateShort creates a shortened URL for the given original URL.
	// Returns a postgres.ConflictError if the URL was already shortened.
	CreateShort(originalURL []byte, userID []byte) ([]byte, error)
	// CreateMany performs a batch URL shortening operation.
	CreateMany(values []model.CreateManyBodyRaw, userID []byte) ([]model.CreateManyResponseRaw, error)
	// FindByUserID returns all non-deleted shortened URLs owned by the given user.
	FindByUserID(userID []byte) ([]model.LinkRow, error)
}

// Config provides access to application-level configuration needed by the handler.
type Config interface {
	// GetBaseAddress returns the configured base address for building shortened URLs
	// (e.g., "http://localhost:8080"). Returns nil if not configured.
	GetBaseAddress() []byte
}

// DeleteWorker defines the interface for asynchronous link deletion.
type DeleteWorker interface {
	// AddToQueue enqueues a batch of short URL identifiers for deferred deletion
	// under the given user ID.
	AddToQueue(urls []string, userID string)
}

// AuthService provides user authentication context for request handlers.
type AuthService interface {
	// GetUserID extracts the authenticated user ID from the request context.
	GetUserID(ctx context.Context) ([]byte, error)
}

// Auditor defines the interface for emitting audit events.
type Auditor interface {
	// Audit records an auditable action. userID may be nil for anonymous actions.
	Audit(userID *string, action string, URL string)
}

// Handler implements HTTP endpoints for the URL shortener service.
// Use [New] to create a properly initialized instance.
type Handler struct {
	service      Service
	config       Config
	deleteWorker DeleteWorker
	authSvc      AuthService
	auditor      Auditor
	validate     *validator.Validate
}

// CreateShortBody represents the JSON request body for the POST /api/shorten endpoint.
type CreateShortBody struct {
	URL string `json:"url" validate:"required,min=3"`
}

// CreateShortResponse represents the JSON response body returned after
// successfully creating a shortened URL via POST /api/shorten.
type CreateShortResponse struct {
	Result string `json:"result"`
}

// FindByUserIDResponse represents a single entry in the GET /api/user/urls response.
type FindByUserIDResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// New creates a new Handler with the required dependencies.
// All parameters must be non-nil; an error is returned if any is nil.
func New(service Service, config Config, deleteWorker DeleteWorker, authService AuthService, auditor Auditor) (
	*Handler, error,
) {
	if service == nil {
		return &Handler{}, errors.New("handler service must be not nil")
	}
	if config == nil {
		return &Handler{}, errors.New("handler config must be not nil")
	}
	if deleteWorker == nil {
		return &Handler{}, errors.New("deleteWorker must be not nil")
	}
	if authService == nil {
		return &Handler{}, errors.New("authService must be not nil")
	}
	if auditor == nil {
		return &Handler{}, errors.New("auditor must be not nil")
	}

	return &Handler{
		service: service, config: config, deleteWorker: deleteWorker, authSvc: authService, auditor: auditor,
		validate: validator.New(),
	}, nil
}

// GetByID handles GET /{id} requests. It resolves a short URL identifier to
// the original URL and responds with an HTTP 307 redirect.
//
// Response codes:
//   - 307 Temporary Redirect with Location header on success
//   - 410 Gone if the link was deleted
//   - 400 Bad Request if the ID is missing
func (h *Handler) GetByID(writer http.ResponseWriter, request *http.Request) {
	strID := strings.TrimPrefix(request.URL.Path, "/")

	if strID == "" {
		http.Error(writer, "ID parameter is required", http.StatusBadRequest)
		return
	}

	link, err := h.service.GetByID(strID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writer.WriteHeader(http.StatusGone)
			return
		}

		http.Error(writer, "Error while getting url", http.StatusBadRequest)
		return
	}

	writer.Header().Add("Content-Type", "text/plain")
	writer.Header().Add("Location", string(link))
	writer.WriteHeader(http.StatusTemporaryRedirect)
	h.auditor.Audit(nil, "follow", string(link))
}

// CreateShort handles POST / requests with Content-Type: text/plain.
// The request body must contain the original URL as plain text.
//
// Response codes:
//   - 201 Created with the shortened URL in the body on success
//   - 409 Conflict if the URL was already shortened (returns existing short URL)
//   - 400 Bad Request on invalid content type or empty body
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
	userID, err := h.authSvc.GetUserID(request.Context())
	if err != nil {
		logger.Logger.Error().Err(err).Msg("error while get UserID")
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	short, err := h.service.CreateShort(body, userID)
	headers := map[string]string{"Content-Type": "text/plain"}
	if conflictError, ok := errors.AsType[*postgres.ConflictError](err); ok {
		prepareResponse(
			writer, headers, http.StatusConflict, h.prepareURL(request.Host, conflictError.Shorten),
		)
		return
	}

	if err != nil {
		logger.Logger.Error().Err(err).Msg("error while create short url")
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	result := h.prepareURL(request.Host, short)
	prepareResponse(writer, headers, http.StatusCreated, result)
	h.auditor.Audit(new(string(userID)), "shorten", string(body))
}

// CreateFromJSONBody handles POST /api/shorten requests with Content-Type: application/json.
// It accepts a JSON body with a "url" field and returns a JSON response with the
// shortened URL in the "result" field.
//
// Request body example:
//
//	{"url": "https://example.com/very/long/path"}
//
// Response body example:
//
//	{"result": "http://localhost:8080/abc123"}
//
// Response codes:
//   - 201 Created with JSON result on success
//   - 409 Conflict with JSON result if URL already shortened
//   - 400 Bad Request on invalid JSON or validation failure
func (h *Handler) CreateFromJSONBody(writer http.ResponseWriter, request *http.Request) {
	if contentType := request.Header.Get("Content-Type"); contentType != "application/json" {
		http.Error(writer, "Invalid content-type", http.StatusBadRequest)
		return
	}

	bodyBytes, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, "Error while read body", http.StatusBadRequest)
		return
	}
	defer func() {
		if err = request.Body.Close(); err != nil {
			logger.Logger.Error().Err(err).Msg("error while close body")
		}
	}()

	var body CreateShortBody

	if err = json.Unmarshal(bodyBytes, &body); err != nil {
		http.Error(writer, "Error while decode body", http.StatusBadRequest)
		return
	}

	if err = h.validate.Struct(body); err != nil {
		http.Error(writer, "Error while validate body", http.StatusBadRequest)
		return
	}
	userID, err := h.authSvc.GetUserID(request.Context())
	if err != nil {
		logger.Logger.Error().Err(err).Msg("error while get UserID")
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	short, err := h.service.CreateShort([]byte(body.URL), userID)
	headers := map[string]string{"Content-Type": "application/json"}
	if conflictError, ok := errors.AsType[*postgres.ConflictError](err); ok {
		h.prepareJSONResponse(writer, request.Host, conflictError.Shorten, http.StatusConflict, headers)
		return
	}
	if err != nil {
		logger.Logger.Error().Err(err).Msg("error while create short url")
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	h.prepareJSONResponse(writer, request.Host, short, http.StatusCreated, headers)
	h.auditor.Audit(new(string(userID)), "shorten", body.URL)
}

// CreateMany handles POST /api/shorten/batch requests for batch URL shortening.
// It accepts a JSON array of objects with "correlation_id" and "original_url" fields,
// and returns a JSON array with "correlation_id" and "short_url" fields.
//
// Request body example:
//
//	[{"correlation_id": "1", "original_url": "https://example.com/a"}]
//
// Response body example:
//
//	[{"correlation_id": "1", "short_url": "http://localhost:8080/abc123"}]
//
// Response codes:
//   - 201 Created with JSON array of shortened URLs on success
//   - 400 Bad Request on invalid JSON, empty body, or validation failure
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
		if err = h.validate.Struct(deserialized[i]); err != nil {
			http.Error(writer, "Error while validate body", http.StatusBadRequest)
			return
		}
	}
	userID, err := h.authSvc.GetUserID(request.Context())
	if err != nil {
		logger.Logger.Error().Err(err).Msg("error while get UserID")
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	shorts, err := h.service.CreateMany(deserialized, userID)
	if err != nil {
		logger.Logger.Error().Err(err).Msg("error while create short url")
		http.Error(writer, "Error while create short url", http.StatusInternalServerError)
		return
	}

	for i := range shorts {
		res := h.prepareURL(request.Host, []byte(shorts[i].ShortURL))
		shorts[i].ShortURL = string(res)
	}

	resp, err := json.Marshal(shorts)
	if err != nil {
		logger.Logger.Error().Err(err).Msg("error while prepare JSON")
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	headers := map[string]string{"Content-Type": "application/json"}
	prepareResponse(writer, headers, http.StatusCreated, resp)
}

// GetByUserID handles GET /api/user/urls requests. It returns all shortened URLs
// owned by the authenticated user.
//
// Response body example:
//
//	[{"short_url": "http://localhost:8080/abc", "original_url": "https://example.com"}]
//
// Response codes:
//   - 200 OK with JSON array of user's URLs
//   - 204 No Content if the user has no URLs
//   - 500 Internal Server Error on auth or service failure
func (h *Handler) GetByUserID(w http.ResponseWriter, r *http.Request) {
	userID, err := h.authSvc.GetUserID(r.Context())
	if err != nil {
		logger.Logger.Error().Err(err).Msg("error while get UserID")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	res, err := h.service.FindByUserID(userID)
	if err != nil {
		logger.Logger.Error().Err(err).Msg("error while FindByUserID")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var status int
	if len(res) == 0 {
		status = http.StatusNoContent
	} else {
		status = http.StatusOK
	}
	headers := map[string]string{"Content-Type": "application/json"}
	h.prepareFindByUserIDResponse(w, r.Host, res, status, headers)
}

// DeleteLinks handles DELETE /api/user/urls requests. It accepts a JSON array of
// short URL identifiers and enqueues them for asynchronous deletion. Only URLs
// owned by the authenticated user will be deleted.
//
// Request body example:
//
//	["abc123", "def456"]
//
// Response codes:
//   - 202 Accepted (deletion scheduled, may not complete immediately)
//   - 400 Bad Request on invalid JSON or empty body
func (h *Handler) DeleteLinks(w http.ResponseWriter, r *http.Request) {
	if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
		http.Error(w, "Invalid content-type", http.StatusBadRequest)
		return
	}
	userID, err := h.authSvc.GetUserID(r.Context())
	if err != nil {
		logger.Logger.Error().Err(err).Msg("error while get UserID")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer func() {
		if err = r.Body.Close(); err != nil {
			logger.Logger.Error().Err(err).Msg("error while close body")
		}
	}()

	var deserialized []string

	err = json.Unmarshal(body, &deserialized)
	if err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	if len(deserialized) == 0 {
		http.Error(w, "empty body values", http.StatusBadRequest)
		return
	}
	for i := range deserialized {
		if len(deserialized[i]) == 0 {
			http.Error(w, "Invalid url format", http.StatusBadRequest)
			return
		}
	}
	h.deleteWorker.AddToQueue(deserialized, string(userID))

	prepareResponse(w, make(map[string]string), http.StatusAccepted, []byte{})
}

func (h *Handler) prepareURL(host string, short []byte) []byte {
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

func (h *Handler) prepareFindByUserIDResponse(
	w http.ResponseWriter, host string, res []model.LinkRow, status int, headers map[string]string,
) {
	if status == http.StatusNoContent {
		w.WriteHeader(status)
		return
	}

	var output []FindByUserIDResponse
	if len(res) != 0 {
		for _, value := range res {
			shorten := h.prepareURL(host, []byte(value.ShortURL))
			output = append(
				output, FindByUserIDResponse{
					ShortURL:    string(shorten),
					OriginalURL: value.OriginalURL,
				},
			)
		}
	}

	resp, err := json.Marshal(output)
	if err != nil {
		logger.Logger.Error().Err(err).Msg("error while serialized body")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	prepareResponse(w, headers, status, resp)
}

func (h *Handler) prepareJSONResponse(
	w http.ResponseWriter, host string, res []byte, status int, headers map[string]string,
) {
	result := h.prepareURL(host, res)
	response := &CreateShortResponse{
		Result: string(result),
	}

	resp, err := json.Marshal(response)
	if err != nil {
		logger.Logger.Error().Err(err).Msg("error while serialized body")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	prepareResponse(w, headers, status, resp)
}

func prepareResponse(w http.ResponseWriter, headers map[string]string, statusCode int, body []byte) {
	for key, value := range headers {
		w.Header().Add(key, value)
	}
	w.WriteHeader(statusCode)
	if len(body) != 0 {
		// #nosec G705
		_, err := w.Write(body)
		if err != nil {
			logger.Logger.Error().Err(err).Msg("error while write body")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}
}
