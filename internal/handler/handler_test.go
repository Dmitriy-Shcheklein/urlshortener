package handler

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/logger"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type errorReader struct {
	err error
}

func (r *errorReader) Read(_ []byte) (int, error) {
	return 0, r.err
}

type MockService struct {
	mock.Mock
}

func (s *MockService) GetByID(id string) ([]byte, error) {
	args := s.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}
func (s *MockService) CreateShort(originalURL []byte) ([]byte, error) {
	args := s.Called(originalURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

type MockConfig struct {
	mock.Mock
}

func (c *MockConfig) GetBaseAddress() []byte {
	args := c.Called()
	if args.Get(0) == nil {
		return []byte("")
	}
	return args.Get(0).([]byte)
}

func (h *Handler) GetService() Service {
	return h.service
}

func TestNew(t *testing.T) {
	service := &MockService{}
	config := &MockConfig{}
	handler, _ := New(service, config)
	assert.NotNil(t, handler, "Указатель на обработчик не должен быть nil")
	assert.NotNil(t, handler.GetService(), "Указатель на сервис не должен быть nil")
}

func TestNewErrors(t *testing.T) {
	t.Run(
		"Ошибка, сервис не инициализирован", func(t *testing.T) {
			_, err := New(nil, &MockConfig{})
			assert.Equal(t, errors.New("handler service must be not nil"), err)
		},
	)
	t.Run(
		"Ошибка, конфиг не инициализирован", func(t *testing.T) {
			_, err := New(&MockService{}, nil)
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

	setup := func() {
		path = "test"
		fullLink = []byte("fullLink")
		request = httptest.NewRequest(http.MethodGet, "/"+path, nil)
		writer = httptest.NewRecorder()
		service = &MockService{}
		config = &MockConfig{}

		handler, _ = New(service, config)
	}

	teardown := func() {
		service.AssertExpectations(t)
	}

	t.Run(
		"Должен выполниться без ошибок", func(t *testing.T) {
			setup()
			defer teardown()

			service.On("GetByID", path).Return(fullLink, nil)

			assert.NotPanics(
				t, func() {
					handler.GetByID(writer, request)
				},
			)
		},
	)

	t.Run(
		"Должен установить заголовки ответа", func(t *testing.T) {
			setup()

			service.On("GetByID", path).Return(fullLink, nil)

			handler.GetByID(writer, request)

			assert.Equal(t, "text/plain", writer.Header().Get("Content-Type"))
			assert.Equal(t, "fullLink", writer.Header().Get("Location"))
			assert.Equal(t, http.StatusTemporaryRedirect, writer.Code)
		},
	)

	t.Run(
		"Ошибка, не указана короткая ссылка", func(t *testing.T) {
			setup()
			request = httptest.NewRequest(http.MethodGet, "/", nil)

			handler.GetByID(writer, request)

			assert.Equal(t, http.StatusBadRequest, writer.Code)
			assert.Equal(t, "ID parameter is required\n", writer.Body.String())
		},
	)

	t.Run(
		"Ошибка при получении ссылки из сервиса", func(t *testing.T) {
			setup()
			defer teardown()
			service.On("GetByID", path).Return(nil, assert.AnError)

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

	setup := func() {
		path = "/"
		fullLink = "https://ya.ru"
		shortLink = []byte("short")
		body = strings.NewReader(fullLink)
		request = httptest.NewRequest(http.MethodGet, path, body)
		request.Header.Set("Content-Type", "text/plain")
		writer = httptest.NewRecorder()
		service = &MockService{}
		config = &MockConfig{}

		handler, _ = New(service, config)
	}

	teardown := func() {
		service.AssertExpectations(t)
	}

	t.Run(
		"Должен выполниться без ошибок", func(t *testing.T) {
			setup()
			defer teardown()

			service.On("CreateShort", []byte(fullLink)).Return(shortLink, nil)
			config.On("GetBaseAddress").Return(baseAddress)

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
				setup()

				service.On("CreateShort", []byte(fullLink)).Return(shortLink, nil)
				config.On("GetBaseAddress").Return(test.baseAddress)

				handler.CreateShort(writer, request)

				assert.Equal(t, "text/plain", writer.Header().Get("Content-Type"))
				assert.Equal(t, http.StatusCreated, writer.Code)
				assert.Equal(t, test.body, writer.Body.String())
			}
		},
	)
	t.Run(
		"Ошибка, некорректный content-type", func(t *testing.T) {
			setup()
			request.Header.Set("Content-Type", "application/json")

			service.On("CreateShort", []byte(fullLink)).Return(shortLink, nil)
			config.On("GetBaseAddress").Return(baseAddress)

			handler.CreateShort(writer, request)

			assert.Equal(t, http.StatusBadRequest, writer.Code)
			assert.Equal(t, "Invalid content-type\n", writer.Body.String())
		},
	)

	t.Run(
		"Ошибка, пустое тело запрос", func(t *testing.T) {
			setup()
			request = httptest.NewRequest(http.MethodGet, path, nil)
			request.Header.Set("Content-Type", "text/plain")

			service.On("CreateShort", []byte(fullLink)).Return(shortLink, nil)
			config.On("GetBaseAddress").Return(baseAddress)

			handler.CreateShort(writer, request)

			assert.Equal(t, http.StatusBadRequest, writer.Code)
			assert.Equal(t, "Empty body\n", writer.Body.String())
		},
	)

	t.Run(
		"Ошибка создания короткой ссылки", func(t *testing.T) {
			setup()

			service.On("CreateShort", []byte(fullLink)).Return(nil, assert.AnError)
			config.On("GetBaseAddress").Return(baseAddress)

			handler.CreateShort(writer, request)

			assert.Equal(t, http.StatusBadRequest, writer.Code)
			assert.Equal(t, "Error while create short url\n", writer.Body.String())
		},
	)

	t.Run(
		"Ошибка при чтении тела запроса", func(t *testing.T) {
			setup()
			request = httptest.NewRequest(http.MethodGet, path, &errorReader{err: errors.New("read error")})
			request.Header.Set("Content-Type", "text/plain")

			service.On("CreateShort", []byte(fullLink)).Return(nil, assert.AnError)
			config.On("GetBaseAddress").Return(baseAddress)

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

	setup := func() {
		path = "/"
		fullLink = "https://practicum.yandex.ru"
		shortLink = []byte("short")
		body = strings.NewReader("{\"url\": \"https://practicum.yandex.ru\"}")
		request = httptest.NewRequest(http.MethodPost, path, body)
		request.Header.Set("Content-Type", "application/json")
		writer = httptest.NewRecorder()
		service = &MockService{}
		config = &MockConfig{}

		handler, _ = New(service, config)
	}

	teardown := func() {
		service.AssertExpectations(t)
	}

	t.Run(
		"Должен выполниться без ошибок", func(t *testing.T) {
			setup()
			defer teardown()

			service.On("CreateShort", []byte(fullLink)).Return(shortLink, nil)
			config.On("GetBaseAddress").Return(baseAddress)

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
				setup()

				service.On("CreateShort", []byte(fullLink)).Return(shortLink, nil)
				config.On("GetBaseAddress").Return(test.baseAddress)

				handler.CreateFromJSONBody(writer, request)

				assert.Equal(t, "application/json", writer.Header().Get("Content-Type"))
				assert.Equal(t, http.StatusCreated, writer.Code)
				assert.Equal(t, test.body, writer.Body.String())
			}
		},
	)
	t.Run(
		"Ошибка, некорректный content-type", func(t *testing.T) {
			setup()
			request.Header.Set("Content-Type", "text/plain")

			service.On("CreateShort", []byte(fullLink)).Return(shortLink, nil)
			config.On("GetBaseAddress").Return(baseAddress)

			handler.CreateFromJSONBody(writer, request)

			assert.Equal(t, http.StatusBadRequest, writer.Code)
			assert.Equal(t, "Invalid content-type\n", writer.Body.String())
		},
	)

	t.Run(
		"Ошибка, пустое тело запрос", func(t *testing.T) {
			setup()
			request = httptest.NewRequest(http.MethodGet, path, nil)
			request.Header.Set("Content-Type", "application/json")

			service.On("CreateShort", []byte(fullLink)).Return(shortLink, nil)
			config.On("GetBaseAddress").Return(baseAddress)

			handler.CreateFromJSONBody(writer, request)

			assert.Equal(t, http.StatusBadRequest, writer.Code)
			assert.Equal(t, "Error while decode body\n", writer.Body.String())
		},
	)

	t.Run(
		"Ошибка создания короткой ссылки", func(t *testing.T) {
			setup()

			service.On("CreateShort", []byte(fullLink)).Return(nil, assert.AnError)
			config.On("GetBaseAddress").Return(baseAddress)

			handler.CreateFromJSONBody(writer, request)

			assert.Equal(t, http.StatusBadRequest, writer.Code)
			assert.Equal(t, "Error while create short url\n", writer.Body.String())
		},
	)

	t.Run(
		"Ошибка при чтении тела запроса", func(t *testing.T) {
			setup()
			request = httptest.NewRequest(http.MethodGet, path, &errorReader{err: errors.New("read error")})
			request.Header.Set("Content-Type", "application/json")

			service.On("CreateShort", []byte(fullLink)).Return(nil, assert.AnError)
			config.On("GetBaseAddress").Return(baseAddress)

			handler.CreateFromJSONBody(writer, request)

			assert.Equal(t, http.StatusBadRequest, writer.Code)
			assert.Equal(t, "Error while decode body\n", writer.Body.String())
		},
	)
}
