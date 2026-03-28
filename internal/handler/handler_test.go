package handler

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}

type MockService struct {
	mock.Mock
}

func (s *MockService) GetById(ID string) ([]byte, error) {
	args := s.Called(ID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}
func (s *MockService) CreateShort(originalUrl []byte) ([]byte, error) {
	args := s.Called(originalUrl)
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
	service := new(MockService)
	config := new(MockConfig)
	handler := New(service, config)
	assert.NotNil(t, handler, "Указатель на обработчик не должен быть nil")
	assert.NotNil(t, handler.GetService(), "Указатель на сервис не должен быть nil")

}

func TestNewWithPanic(t *testing.T) {
	assert.PanicsWithValue(
		t, "Handler service must be not nil", func() {
			New(nil, new(MockConfig))
		},
	)
	assert.PanicsWithValue(
		t, "Handler config must be not nil", func() {
			New(new(MockService), nil)
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
		service = new(MockService)
		config = new(MockConfig)

		handler = New(service, config)
	}

	teardown := func() {
		service.AssertExpectations(t)
	}

	t.Run(
		"Должен выполниться без ошибок", func(t *testing.T) {
			setup()
			defer teardown()

			service.On("GetById", path).Return(fullLink, nil)

			assert.NotPanics(
				t, func() {
					handler.GetByd(writer, request)
				},
			)
		},
	)

	t.Run(
		"Должен установить заголовки ответа", func(t *testing.T) {
			setup()

			service.On("GetById", path).Return(fullLink, nil)

			handler.GetByd(writer, request)

			assert.Equal(t, "text/plain", writer.Header().Get("Content-Type"))
			assert.Equal(t, "fullLink", writer.Header().Get("Location"))
			assert.Equal(t, http.StatusTemporaryRedirect, writer.Code)
		},
	)

	t.Run(
		"Ошибка, не указана короткая ссылка", func(t *testing.T) {
			setup()
			request = httptest.NewRequest(http.MethodGet, "/", nil)

			handler.GetByd(writer, request)

			assert.Equal(t, http.StatusBadRequest, writer.Code)
			assert.Equal(t, "ID parameter is required\n", writer.Body.String())

		},
	)

	t.Run(
		"Ошибка при получении ссылки из сервиса", func(t *testing.T) {
			setup()
			defer teardown()
			service.On("GetById", path).Return(nil, assert.AnError)

			handler.GetByd(writer, request)

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

	setup := func() {
		path = "/"
		fullLink = "https://ya.ru"
		shortLink = []byte("short")
		body = strings.NewReader(fullLink)
		request = httptest.NewRequest(http.MethodGet, path, body)
		request.Header.Set("Content-Type", "text/plain")
		writer = httptest.NewRecorder()
		service = new(MockService)
		config = new(MockConfig)

		handler = New(service, config)
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
