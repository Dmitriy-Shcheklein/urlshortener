package httpobserver

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/model"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newLogger() *zerolog.Logger {
	return new(zerolog.Nop())
}

func TestObserver(t *testing.T) {
	t.Run("Тест создания Observer", func(t *testing.T) {
		t.Run("Должен создать Observer", func(t *testing.T) {
			logger := newLogger()
			client := NewMockHttpClient(t)
			o := New(logger, client, "http://audit.example.com")

			assert.NotNil(t, o)
			assert.Equal(t, logger, o.logger)
			assert.Equal(t, client, o.httpClient)
			assert.Equal(t, "http://audit.example.com", o.url)
		})
	})

	t.Run("Тест HandleMessage", func(t *testing.T) {
		t.Run("Должен отправить POST-запрос с корректным JSON", func(t *testing.T) {
			client := NewMockHttpClient(t)
			o := New(newLogger(), client, "http://audit.example.com")

			msg := model.AuditMsg{Ts: 1234567890, Action: "create", UserID: new("user1"), URL: "http://example.com"}

			client.EXPECT().Post("http://audit.example.com", "application/json", mock.Anything).
				Run(func(url string, contentType string, body io.Reader) {
					data, err := io.ReadAll(body)
					require.NoError(t, err)

					var got model.AuditMsg
					err = json.Unmarshal(data, &got)
					require.NoError(t, err)

					assert.Equal(t, msg.Ts, got.Ts)
					assert.Equal(t, msg.Action, got.Action)
					assert.Equal(t, msg.URL, got.URL)
					assert.Equal(t, msg.UserID, got.UserID)
				}).
				Return(&http.Response{StatusCode: http.StatusOK}, nil)

			o.HandleMessage(msg)
		})

		t.Run("Должен не паниковать при ошибке HTTP-клиента", func(t *testing.T) {
			client := NewMockHttpClient(t)
			o := New(newLogger(), client, "http://audit.example.com")

			msg := model.AuditMsg{Ts: 1, Action: "create", URL: "http://example.com"}

			client.EXPECT().Post("http://audit.example.com", "application/json", mock.Anything).
				Return(nil, assert.AnError)

			assert.NotPanics(t, func() {
				o.HandleMessage(msg)
			})
		})

		t.Run("Должен отправить сообщение без UserID", func(t *testing.T) {
			client := NewMockHttpClient(t)
			o := New(newLogger(), client, "http://audit.example.com")

			msg := model.AuditMsg{Ts: 100, Action: "delete", URL: "http://example.com/123"}

			client.EXPECT().Post("http://audit.example.com", "application/json", mock.Anything).
				Return(&http.Response{StatusCode: http.StatusOK}, nil)

			o.HandleMessage(msg)
		})
	})
}
