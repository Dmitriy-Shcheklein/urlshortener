package healthcheck

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthcheckHandler(t *testing.T) {
	var (
		mockService *ServiceMock
		handler     *Handler
		response    *httptest.ResponseRecorder
		request     *http.Request
	)

	setup := func(t *testing.T) {
		ctrl := minimock.NewController(t)
		mockService = NewServiceMock(ctrl)
		request = httptest.NewRequest(http.MethodGet, "/ping", nil)
		response = httptest.NewRecorder()

		handler = &Handler{service: mockService}
	}

	t.Run(
		"Тест создания хэндлера", func(t *testing.T) {
			t.Run(
				"Должен создать экземпляр без ошибок", func(t *testing.T) {
					h, err := New(&ServiceMock{})

					require.NoError(t, err)
					assert.NotNil(t, h)
				},
			)

			t.Run(
				"Ошибка, не передан сервис", func(t *testing.T) {
					_, err := New(nil)

					require.Error(t, err)
				},
			)
		},
	)

	t.Run(
		"Тест метода PingDB", func(t *testing.T) {
			t.Run(
				"Должен выполниться без ошибок", func(t *testing.T) {
					setup(t)

					mockService.PingDBMock.Expect().Return(nil)

					handler.PingDB(response, request)

					assert.Equal(t, http.StatusOK, response.Code)
				},
			)

			t.Run(
				"Ошибка при выполнении", func(t *testing.T) {
					setup(t)

					expectedErr := assert.AnError
					mockService.PingDBMock.Expect().Return(expectedErr)

					handler.PingDB(response, request)

					assert.Equal(t, http.StatusInternalServerError, response.Code)
				},
			)
		},
	)
}
