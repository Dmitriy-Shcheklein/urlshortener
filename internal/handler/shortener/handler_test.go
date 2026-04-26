package shortener

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/logger"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/middlewares"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/model"
	"github.com/Dmitriy-Shcheklein/urlshortener/internal/repository/postgres"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

type errorReader struct {
	err error
}

func (r *errorReader) Read(_ []byte) (int, error) {
	return 0, r.err
}

func TestNew(t *testing.T) {
	service := NewMockService(t)
	config := NewMockConfig(t)
	deleteWorker := NewMockDeleteWorker(t)

	handler, _ := New(service, config, deleteWorker)
	assert.NotNil(t, handler, "Указатель на обработчик не должен быть nil")
	assert.NotNil(t, handler.service, "Указатель на сервис не должен быть nil")
}

func TestNewErrors(t *testing.T) {
	t.Run(
		"Ошибка, сервис не инициализирован", func(t *testing.T) {
			_, err := New(nil, NewMockConfig(t), NewMockDeleteWorker(t))
			assert.Equal(t, errors.New("handler service must be not nil"), err)
		},
	)
	t.Run(
		"Ошибка, конфиг не инициализирован", func(t *testing.T) {
			_, err := New(NewMockService(t), nil, NewMockDeleteWorker(t))
			assert.Equal(t, errors.New("handler config must be not nil"), err)
		},
	)
	t.Run(
		"Ошибка, deleteWorker не инициализирован", func(t *testing.T) {
			_, err := New(NewMockService(t), NewMockConfig(t), nil)
			assert.Equal(t, errors.New("deleteWorker must be not nil"), err)
		},
	)
}

func TestGetById(t *testing.T) {
	var (
		handler      *Handler
		service      *MockService
		config       *MockConfig
		writer       *httptest.ResponseRecorder
		request      *http.Request
		fullLink     []byte
		path         string
		deleteWorker *MockDeleteWorker
	)

	setup := func(t *testing.T) {
		path = "test"
		fullLink = []byte("fullLink")
		request = httptest.NewRequest(http.MethodGet, "/"+path, nil)
		writer = httptest.NewRecorder()
		service = NewMockService(t)
		deleteWorker = NewMockDeleteWorker(t)
		config = NewMockConfig(t)
		handler, _ = New(service, config, deleteWorker)
		logger.Logger = new(zerolog.Nop())
	}

	t.Run(
		"Должен выполниться без ошибок", func(t *testing.T) {
			setup(t)

			service.EXPECT().GetByID(path).Return(fullLink, nil)

			handler.GetByID(writer, request)

			assert.NotPanics(
				t, func() {
					handler.GetByID(writer, request)
				},
			)
		},
	)

	t.Run(
		"Должен установить заголовки ответа", func(t *testing.T) {
			setup(t)

			service.EXPECT().GetByID(path).Return(fullLink, nil)

			handler.GetByID(writer, request)

			assert.Equal(t, "text/plain", writer.Header().Get("Content-Type"))
			assert.Equal(t, "fullLink", writer.Header().Get("Location"))
			assert.Equal(t, http.StatusTemporaryRedirect, writer.Code)
		},
	)

	t.Run(
		"Ошибка, не указана короткая ссылка", func(t *testing.T) {
			setup(t)
			request = httptest.NewRequest(http.MethodGet, "/", nil)

			handler.GetByID(writer, request)

			assert.Equal(t, http.StatusBadRequest, writer.Code)
			assert.Equal(t, "ID parameter is required\n", writer.Body.String())
		},
	)

	t.Run(
		"Ошибка при получении ссылки из сервиса", func(t *testing.T) {
			setup(t)

			service.EXPECT().GetByID(path).Return(fullLink, assert.AnError)

			handler.GetByID(writer, request)

			assert.Equal(t, http.StatusBadRequest, writer.Code)
			assert.Equal(t, "Error while getting url\n", writer.Body.String())
		},
	)

	t.Run(
		"Должен вернуть 410 код", func(t *testing.T) {
			setup(t)

			service.EXPECT().GetByID(path).Return(nil, nil)

			handler.GetByID(writer, request)

			assert.Equal(t, http.StatusGone, writer.Code)
		},
	)
}

func TestCreateShort(t *testing.T) {
	var (
		handler      *Handler
		service      *MockService
		deleteWorker *MockDeleteWorker
		config       *MockConfig
		writer       *httptest.ResponseRecorder
		request      *http.Request
		fullLink     string
		path         string
		body         io.Reader
		shortLink    []byte
		baseAddress  []byte
		userID       []byte
	)

	logger.InitLogger(zerolog.Disabled)

	setup := func(t *testing.T) {
		userID = []byte("userID")
		path = "/"
		fullLink = "https://ya.ru"
		shortLink = []byte("short")
		body = strings.NewReader(fullLink)
		request = httptest.NewRequest(http.MethodGet, path, body)
		request.Header.Set("Content-Type", "text/plain")
		request = request.WithContext(context.WithValue(request.Context(), middlewares.UserIDKey, userID))
		writer = httptest.NewRecorder()
		service = NewMockService(t)
		deleteWorker = NewMockDeleteWorker(t)
		config = NewMockConfig(t)

		handler, _ = New(service, config, deleteWorker)
	}

	t.Run(
		"Должен выполниться без ошибок", func(t *testing.T) {
			setup(t)

			service.EXPECT().CreateShort([]byte(fullLink), userID).Return(shortLink, nil)
			config.EXPECT().GetBaseAddress().Return(baseAddress)

			assert.NotPanics(
				t, func() {
					handler.CreateShort(writer, request)
				},
			)
		},
	)

	t.Run(
		"Должен установить заголовки и тело ответа", func(t *testing.T) {
			tests := []struct {
				baseAddress []byte
				body        string
			}{
				{baseAddress: []byte{}, body: "http://example.com/short"},
				{baseAddress: []byte("https://ya.ru"), body: "https://ya.ru/short"},
			}

			for _, test := range tests {
				setup(t)

				service.EXPECT().CreateShort([]byte(fullLink), userID).Return(shortLink, nil)
				config.EXPECT().GetBaseAddress().Return(test.baseAddress)

				handler.CreateShort(writer, request)

				assert.Equal(t, "text/plain", writer.Header().Get("Content-Type"))
				assert.Equal(t, http.StatusCreated, writer.Code)
				assert.Equal(t, test.body, writer.Body.String())
			}
		},
	)

	t.Run(
		"Должен установить заголовки и тело ответа - конфликт", func(t *testing.T) {
			setup(t)

			shLink := []byte("short")

			service.EXPECT().CreateShort([]byte(fullLink), userID).Return(
				shLink, postgres.NewConflictError([]byte(fullLink), shLink),
			)
			config.EXPECT().GetBaseAddress().Return([]byte{})

			handler.CreateShort(writer, request)

			assert.Equal(t, "text/plain", writer.Header().Get("Content-Type"))
			assert.Equal(t, http.StatusConflict, writer.Code)
			assert.Equal(t, "http://example.com/short", writer.Body.String())
		},
	)

	t.Run(
		"Ошибка, некорректный content-type", func(t *testing.T) {
			setup(t)
			request.Header.Set("Content-Type", "application/json")

			handler.CreateShort(writer, request)

			assert.Equal(t, http.StatusBadRequest, writer.Code)
			assert.Equal(t, "Invalid content-type\n", writer.Body.String())
		},
	)

	t.Run(
		"Ошибка, пустое тело запрос", func(t *testing.T) {
			setup(t)
			request = httptest.NewRequest(http.MethodGet, path, nil)
			request.Header.Set("Content-Type", "text/plain")

			handler.CreateShort(writer, request)

			assert.Equal(t, http.StatusBadRequest, writer.Code)
			assert.Equal(t, "Empty body\n", writer.Body.String())
		},
	)

	t.Run(
		"Ошибка создания короткой ссылки", func(t *testing.T) {
			setup(t)

			service.EXPECT().CreateShort([]byte(fullLink), userID).Return(nil, assert.AnError)

			handler.CreateShort(writer, request)

			assert.Equal(t, http.StatusInternalServerError, writer.Code)
			assert.Equal(t, "Error while create short url\n", writer.Body.String())
		},
	)

	t.Run(
		"Ошибка при чтении тела запроса", func(t *testing.T) {
			setup(t)
			request = httptest.NewRequest(http.MethodGet, path, &errorReader{err: errors.New("read error")})
			request.Header.Set("Content-Type", "text/plain")

			handler.CreateShort(writer, request)

			assert.Equal(t, http.StatusBadRequest, writer.Code)
			assert.Equal(t, "Error while parse body\n", writer.Body.String())
		},
	)
}

func TestCreateFromJSONBody(t *testing.T) {
	var (
		handler      *Handler
		service      *MockService
		deleteWorker *MockDeleteWorker
		config       *MockConfig
		writer       *httptest.ResponseRecorder
		request      *http.Request
		fullLink     string
		path         string
		body         io.Reader
		shortLink    []byte
		baseAddress  []byte
		userID       []byte
	)

	setup := func(t *testing.T) {
		userID = []byte("userID")
		path = "/"
		fullLink = "https://practicum.yandex.ru"
		shortLink = []byte("short")
		body = strings.NewReader("{\"url\": \"https://practicum.yandex.ru\"}")
		request = httptest.NewRequest(http.MethodPost, path, body)
		request.Header.Set("Content-Type", "application/json")
		request = request.WithContext(context.WithValue(request.Context(), middlewares.UserIDKey, userID))
		writer = httptest.NewRecorder()
		service = NewMockService(t)
		deleteWorker = NewMockDeleteWorker(t)
		config = NewMockConfig(t)

		handler, _ = New(service, config, deleteWorker)
	}

	t.Run(
		"Должен выполниться без ошибок", func(t *testing.T) {
			setup(t)

			service.EXPECT().CreateShort([]byte(fullLink), userID).Return(shortLink, nil)
			config.EXPECT().GetBaseAddress().Return(baseAddress)

			assert.NotPanics(
				t, func() {
					handler.CreateFromJSONBody(writer, request)
				},
			)
		},
	)

	t.Run(
		"Должен установить заголовки и тело ответа", func(t *testing.T) {
			tests := []struct {
				baseAddress []byte
				body        string
			}{
				{baseAddress: []byte{}, body: "{\"result\":\"http://example.com/short\"}"},
				{baseAddress: []byte("https://ya.ru"), body: "{\"result\":\"https://ya.ru/short\"}"},
			}

			for _, test := range tests {
				setup(t)

				service.EXPECT().CreateShort([]byte(fullLink), userID).Return(shortLink, nil)
				config.EXPECT().GetBaseAddress().Return(test.baseAddress)

				handler.CreateFromJSONBody(writer, request)

				assert.Equal(t, "application/json", writer.Header().Get("Content-Type"))
				assert.Equal(t, http.StatusCreated, writer.Code)
				assert.Equal(t, test.body, writer.Body.String())
			}
		},
	)

	t.Run(
		"Должен установить заголовки и тело ответа - конфликт", func(t *testing.T) {
			setup(t)
			originalUrl := []byte(fullLink)
			expectedBody := "{\"result\":\"https://ya.ru/short\"}"

			service.EXPECT().CreateShort(originalUrl, userID).Return(
				shortLink, postgres.NewConflictError(originalUrl, shortLink),
			)
			config.EXPECT().GetBaseAddress().Return([]byte("https://ya.ru"))

			handler.CreateFromJSONBody(writer, request)

			assert.Equal(t, "application/json", writer.Header().Get("Content-Type"))
			assert.Equal(t, http.StatusConflict, writer.Code)
			assert.Equal(t, expectedBody, writer.Body.String())
		},
	)

	t.Run(
		"Ошибка, некорректный content-type", func(t *testing.T) {
			setup(t)
			request.Header.Set("Content-Type", "text/plain")

			handler.CreateFromJSONBody(writer, request)

			assert.Equal(t, http.StatusBadRequest, writer.Code)
			assert.Equal(t, "Invalid content-type\n", writer.Body.String())
		},
	)

	t.Run(
		"Ошибка, пустое тело запрос", func(t *testing.T) {
			setup(t)
			request = httptest.NewRequest(http.MethodGet, path, nil)
			request.Header.Set("Content-Type", "application/json")

			handler.CreateFromJSONBody(writer, request)

			assert.Equal(t, http.StatusBadRequest, writer.Code)
			assert.Equal(t, "Error while decode body\n", writer.Body.String())
		},
	)

	t.Run(
		"Ошибка создания короткой ссылки", func(t *testing.T) {
			setup(t)

			service.EXPECT().CreateShort([]byte(fullLink), userID).Return(nil, assert.AnError)

			handler.CreateFromJSONBody(writer, request)

			assert.Equal(t, http.StatusInternalServerError, writer.Code)
			assert.Equal(t, "Error while create short url\n", writer.Body.String())
		},
	)

	t.Run(
		"Ошибка при чтении тела запроса", func(t *testing.T) {
			setup(t)
			request = httptest.NewRequest(http.MethodGet, path, &errorReader{err: errors.New("read error")})
			request.Header.Set("Content-Type", "application/json")

			handler.CreateFromJSONBody(writer, request)

			assert.Equal(t, http.StatusBadRequest, writer.Code)
			assert.Equal(t, "Error while decode body\n", writer.Body.String())
		},
	)
}

func TestCreateMany(t *testing.T) {
	var (
		handler      *Handler
		service      *MockService
		deleteWorker *MockDeleteWorker
		config       *MockConfig
		writer       *httptest.ResponseRecorder
		request      *http.Request
		svcIncoming  []model.CreateManyBodyRaw
		svcResult    []model.CreateManyResponseRaw
		path         string
		body         io.Reader
		baseAddress  []byte
		userID       []byte
	)

	setup := func(t *testing.T) {
		userID = []byte("userID")
		path = "/"
		svcIncoming = []model.CreateManyBodyRaw{{OriginalURL: "https://practicum.yandex.ru", CorrelationID: "id"}}
		svcResult = []model.CreateManyResponseRaw{{CorrelationID: "id", ShortURL: "url"}}
		body = strings.NewReader("[{\"original_url\": \"https://practicum.yandex.ru\", \"correlation_id\": \"id\"}]")
		request = httptest.NewRequest(http.MethodPost, path, body)
		request.Header.Set("Content-Type", "application/json")
		request = request.WithContext(context.WithValue(request.Context(), middlewares.UserIDKey, userID))
		writer = httptest.NewRecorder()
		service = NewMockService(t)
		deleteWorker = NewMockDeleteWorker(t)
		config = NewMockConfig(t)

		handler, _ = New(service, config, deleteWorker)
		logger.Logger = new(zerolog.Nop())
	}

	t.Run(
		"Должен выполниться без ошибок", func(t *testing.T) {
			setup(t)

			service.EXPECT().CreateMany(svcIncoming, userID).Return(svcResult, nil)
			config.EXPECT().GetBaseAddress().Return(baseAddress)

			assert.NotPanics(
				t, func() {
					handler.CreateMany(writer, request)
				},
			)
		},
	)

	t.Run(
		"Должен установить заголовки и тело ответа", func(t *testing.T) {
			tests := []struct {
				baseAddress []byte
				body        string
			}{
				{
					baseAddress: []byte{},
					body:        "[{\"correlation_id\":\"id\",\"short_url\":\"http://example.com/url\"}]",
				},
				{
					baseAddress: []byte("https://ya.ru"),
					body:        "[{\"correlation_id\":\"id\",\"short_url\":\"https://ya.ru/url\"}]",
				},
			}

			for _, test := range tests {
				setup(t)

				service.EXPECT().CreateMany(svcIncoming, userID).Return(svcResult, nil)
				config.EXPECT().GetBaseAddress().Return(test.baseAddress)

				handler.CreateMany(writer, request)

				assert.Equal(t, "application/json", writer.Header().Get("Content-Type"))
				assert.Equal(t, http.StatusCreated, writer.Code)
				assert.Equal(t, test.body, writer.Body.String())
			}
		},
	)
	t.Run(
		"Ошибка, некорректный content-type", func(t *testing.T) {
			setup(t)
			request.Header.Set("Content-Type", "text/plain")

			handler.CreateMany(writer, request)

			assert.Equal(t, http.StatusBadRequest, writer.Code)
			assert.Equal(t, "Invalid content-type\n", writer.Body.String())
		},
	)

	t.Run(
		"Ошибка, пустое тело запрос", func(t *testing.T) {
			setup(t)
			request = httptest.NewRequest(http.MethodGet, path, nil)
			request.Header.Set("Content-Type", "application/json")

			handler.CreateMany(writer, request)

			assert.Equal(t, http.StatusBadRequest, writer.Code)
			assert.Equal(t, "Invalid JSON format\n", writer.Body.String())
		},
	)

	t.Run(
		"Ошибка создания короткой ссылки", func(t *testing.T) {
			setup(t)

			service.EXPECT().CreateMany(svcIncoming, userID).Return(nil, assert.AnError)

			handler.CreateMany(writer, request)

			assert.Equal(t, http.StatusInternalServerError, writer.Code)
			assert.Equal(t, "Error while create short url\n", writer.Body.String())
		},
	)

	t.Run(
		"Ошибка при чтении тела запроса", func(t *testing.T) {
			setup(t)
			request = httptest.NewRequest(http.MethodGet, path, &errorReader{err: errors.New("read error")})
			request.Header.Set("Content-Type", "application/json")

			handler.CreateMany(writer, request)

			assert.Equal(t, http.StatusBadRequest, writer.Code)
			assert.Equal(t, "Failed to read body\n", writer.Body.String())
		},
	)
}

func TestHandler_GetByUserID(t *testing.T) {
	var (
		handler      *Handler
		service      *MockService
		deleteWorker *MockDeleteWorker
		config       *MockConfig
		writer       *httptest.ResponseRecorder
		request      *http.Request
		path         string
		userID       []byte
		urls         []model.LinkRow
		baseAddress  []byte
	)

	setup := func(t *testing.T) {
		userID = []byte("userID")
		path = "test"
		urls = []model.LinkRow{
			{
				ID:          "1",
				ShortURL:    "short1",
				OriginalURL: "original1",
				UserID:      "user1",
			},
			{
				ID:          "2",
				ShortURL:    "short2",
				OriginalURL: "original2",
				UserID:      "user2",
			},
		}
		baseAddress = []byte{}

		request = httptest.NewRequest(http.MethodGet, "/"+path, nil)
		request = request.WithContext(context.WithValue(request.Context(), middlewares.UserIDKey, userID))
		writer = httptest.NewRecorder()
		service = NewMockService(t)
		deleteWorker = NewMockDeleteWorker(t)
		config = NewMockConfig(t)
		handler, _ = New(service, config, deleteWorker)
		logger.Logger = new(zerolog.Nop())
	}

	t.Run(
		"Должен выполниться без ошибок", func(t *testing.T) {
			setup(t)

			service.EXPECT().FindByUserID(userID).Return(urls, nil)
			config.EXPECT().GetBaseAddress().Return(baseAddress)

			handler.GetByUserID(writer, request)

			assert.Equal(
				t,
				"[{\"short_url\":\"http://example.com/short1\",\"original_url\":\"original1\"},{\"short_url\":\"http://example.com/short2\",\"original_url\":\"original2\"}]",
				writer.Body.String(),
			)
			assert.Equal(t, http.StatusOK, writer.Code)
			assert.Equal(t, "application/json", writer.Header().Get("Content-Type"))
		},
	)

	t.Run(
		"Ошибка получения идентификатора юзера", func(t *testing.T) {
			setup(t)
			request = httptest.NewRequest(http.MethodGet, "/"+path, nil)

			handler.GetByUserID(writer, request)

			assert.Equal(t, http.StatusInternalServerError, writer.Code)
			assert.Equal(t, "error while getting UserID\n", writer.Body.String())
		},
	)

	t.Run(
		"Ошибка получения данных из сервиса", func(t *testing.T) {
			setup(t)

			testError := assert.AnError
			service.EXPECT().FindByUserID(userID).Return(nil, testError)

			handler.GetByUserID(writer, request)

			assert.Equal(t, http.StatusInternalServerError, writer.Code)
			assert.Equal(t, testError.Error()+"\n", writer.Body.String())
		},
	)

	t.Run(
		"Установлен статус 204", func(t *testing.T) {
			setup(t)

			service.EXPECT().FindByUserID(userID).Return([]model.LinkRow{}, nil)

			handler.GetByUserID(writer, request)

			assert.Equal(t, http.StatusNoContent, writer.Code)
		},
	)
}

func TestHandler_DeleteLinks(t *testing.T) {
	var (
		handler      *Handler
		service      *MockService
		deleteWorker *MockDeleteWorker
		config       *MockConfig
		writer       *httptest.ResponseRecorder
		request      *http.Request
		userID       []byte
		urls         []string
		path         string
	)

	setup := func(t *testing.T) {
		userID = []byte("userID")
		urls = []string{
			"1", "2", "3",
		}
		path = "test"

		request = httptest.NewRequest(http.MethodDelete, "/"+path, strings.NewReader("[\"1\",\"2\",\"3\"]"))
		request = request.WithContext(context.WithValue(request.Context(), middlewares.UserIDKey, userID))
		request.Header.Set("Content-Type", "application/json")
		writer = httptest.NewRecorder()
		deleteWorker = NewMockDeleteWorker(t)
		service = NewMockService(t)
		config = NewMockConfig(t)
		handler, _ = New(service, config, deleteWorker)
		logger.Logger = new(zerolog.Nop())
	}

	t.Run(
		"Должен выполниться без ошибок", func(t *testing.T) {
			setup(t)

			deleteWorker.EXPECT().AddToQueue(urls, string(userID)).Return()

			handler.DeleteLinks(writer, request)

			assert.Equal(t, http.StatusAccepted, writer.Code)
		},
	)

	t.Run(
		"Ошибка получения идентификатора юзера", func(t *testing.T) {
			setup(t)
			request = httptest.NewRequest(http.MethodDelete, "/"+path, nil)
			request.Header.Set("Content-Type", "application/json")

			handler.DeleteLinks(writer, request)

			assert.Equal(t, http.StatusInternalServerError, writer.Code)
			assert.Equal(t, "error while getting UserID\n", writer.Body.String())
		},
	)

	t.Run(
		"Ошибка, некорректный content type", func(t *testing.T) {
			setup(t)
			request = httptest.NewRequest(http.MethodDelete, "/"+path, nil)

			handler.DeleteLinks(writer, request)

			assert.Equal(t, http.StatusBadRequest, writer.Code)
			assert.Equal(t, "Invalid content-type\n", writer.Body.String())
		},
	)

	t.Run(
		"Ошибка, пустое тело запрос", func(t *testing.T) {
			setup(t)
			request = httptest.NewRequest(http.MethodDelete, "/"+path, strings.NewReader("[]"))
			request = request.WithContext(context.WithValue(request.Context(), middlewares.UserIDKey, userID))
			request.Header.Set("Content-Type", "application/json")

			handler.DeleteLinks(writer, request)

			assert.Equal(t, http.StatusBadRequest, writer.Code)
			assert.Equal(t, "empty body values\n", writer.Body.String())
		},
	)

	t.Run(
		"Ошибка, невалидный JSON", func(t *testing.T) {
			setup(t)
			request = httptest.NewRequest(http.MethodDelete, "/"+path, strings.NewReader("[ads: 1]"))
			request = request.WithContext(context.WithValue(request.Context(), middlewares.UserIDKey, userID))
			request.Header.Set("Content-Type", "application/json")

			handler.DeleteLinks(writer, request)

			assert.Equal(t, http.StatusBadRequest, writer.Code)
			assert.Equal(t, "Invalid JSON format\n", writer.Body.String())
		},
	)
}
