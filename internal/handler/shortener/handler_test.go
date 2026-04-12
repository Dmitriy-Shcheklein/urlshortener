package shortener

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/logger"
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
	handler, _ := New(service, config)
	assert.NotNil(t, handler, "Указатель на обработчик не должен быть nil")
	assert.NotNil(t, handler.service, "Указатель на сервис не должен быть nil")
}

func TestNewErrors(t *testing.T) {
	t.Run(
		"Ошибка, сервис не инициализирован", func(t *testing.T) {
			_, err := New(nil, NewMockConfig(t))
			assert.Equal(t, errors.New("handler service must be not nil"), err)
		},
	)
	t.Run(
		"Ошибка, конфиг не инициализирован", func(t *testing.T) {
			_, err := New(NewMockService(t), nil)
			assert.Equal(t, errors.New("handler config must be not nil"), err)
		},
	)
}

func TestGetById(t *testing.T) {
	var (
		handler  *Handler
		service  *MockService
		config   *MockConfig
		writer   *httptest.ResponseRecorder
		request  *http.Request
		fullLink []byte
		path     string
	)

	setup := func(t *testing.T) {
		path = "test"
		fullLink = []byte("fullLink")
		request = httptest.NewRequest(http.MethodGet, "/"+path, nil)
		writer = httptest.NewRecorder()
		service = NewMockService(t)
		config = NewMockConfig(t)
		handler, _ = New(service, config)
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
}

func TestCreateShort(t *testing.T) {
	var (
		handler     *Handler
		service     *MockService
		config      *MockConfig
		writer      *httptest.ResponseRecorder
		request     *http.Request
		fullLink    string
		path        string
		body        io.Reader
		shortLink   []byte
		baseAddress []byte
	)

	logger.InitLogger(zerolog.Disabled)

	setup := func(t *testing.T) {
		path = "/"
		fullLink = "https://ya.ru"
		shortLink = []byte("short")
		body = strings.NewReader(fullLink)
		request = httptest.NewRequest(http.MethodGet, path, body)
		request.Header.Set("Content-Type", "text/plain")
		writer = httptest.NewRecorder()
		service = NewMockService(t)
		config = NewMockConfig(t)

		handler, _ = New(service, config)
	}

	t.Run(
		"Должен выполниться без ошибок", func(t *testing.T) {
			setup(t)

			service.EXPECT().CreateShort([]byte(fullLink)).Return(shortLink, nil)
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

				service.EXPECT().CreateShort([]byte(fullLink)).Return(shortLink, nil)
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

			service.EXPECT().CreateShort([]byte(fullLink)).Return(
				shortLink, postgres.NewConflictError([]byte(fullLink), shortLink),
			)

			handler.CreateShort(writer, request)

			assert.Equal(t, "text/plain", writer.Header().Get("Content-Type"))
			assert.Equal(t, http.StatusConflict, writer.Code)
			assert.Equal(t, string(shortLink), writer.Body.String())
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

			service.EXPECT().CreateShort([]byte(fullLink)).Return(nil, assert.AnError)

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
		handler     *Handler
		service     *MockService
		config      *MockConfig
		writer      *httptest.ResponseRecorder
		request     *http.Request
		fullLink    string
		path        string
		body        io.Reader
		shortLink   []byte
		baseAddress []byte
	)

	setup := func(t *testing.T) {
		path = "/"
		fullLink = "https://practicum.yandex.ru"
		shortLink = []byte("short")
		body = strings.NewReader("{\"url\": \"https://practicum.yandex.ru\"}")
		request = httptest.NewRequest(http.MethodPost, path, body)
		request.Header.Set("Content-Type", "application/json")
		writer = httptest.NewRecorder()
		service = NewMockService(t)
		config = NewMockConfig(t)

		handler, _ = New(service, config)
	}

	t.Run(
		"Должен выполниться без ошибок", func(t *testing.T) {
			setup(t)

			service.EXPECT().CreateShort([]byte(fullLink)).Return(shortLink, nil)
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

				service.EXPECT().CreateShort([]byte(fullLink)).Return(shortLink, nil)
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

			service.EXPECT().CreateShort(originalUrl).Return(
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

			service.EXPECT().CreateShort([]byte(fullLink)).Return(nil, assert.AnError)

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
		handler     *Handler
		service     *MockService
		config      *MockConfig
		writer      *httptest.ResponseRecorder
		request     *http.Request
		svcIncoming []CreateManyBodyRaw
		svcResult   []CreateManyResponseRaw
		path        string
		body        io.Reader
		baseAddress []byte
	)

	setup := func(t *testing.T) {
		path = "/"
		svcIncoming = []CreateManyBodyRaw{{OriginalUrl: "https://practicum.yandex.ru", CorrelationID: "id"}}
		svcResult = []CreateManyResponseRaw{{CorrelationId: "id", ShortURL: "url"}}
		body = strings.NewReader("[{\"original_url\": \"https://practicum.yandex.ru\", \"correlation_id\": \"id\"}]")
		request = httptest.NewRequest(http.MethodPost, path, body)
		request.Header.Set("Content-Type", "application/json")
		writer = httptest.NewRecorder()
		service = NewMockService(t)
		config = NewMockConfig(t)

		handler, _ = New(service, config)
		logger.Logger = new(zerolog.Nop())
	}

	t.Run(
		"Должен выполниться без ошибок", func(t *testing.T) {
			setup(t)

			service.EXPECT().CreateMany(svcIncoming).Return(svcResult, nil)
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

				service.EXPECT().CreateMany(svcIncoming).Return(svcResult, nil)
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

			service.EXPECT().CreateMany(svcIncoming).Return(nil, assert.AnError)

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
