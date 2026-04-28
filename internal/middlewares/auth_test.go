package middlewares

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuth(t *testing.T) {
	t.Run(
		"Cookie Generation", func(t *testing.T) {
			t.Run(
				"Успешное создание cookie с корректными параметрами", func(t *testing.T) {
					cookie, err := generateNewCookie()

					require.NoError(t, err)
					assert.NotNil(t, cookie)
					assert.Equal(t, "auth", cookie.Name)
					assert.Equal(t, "/", cookie.Path)
					assert.True(t, cookie.HttpOnly)
					assert.False(t, cookie.Secure)
					assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
					assert.Equal(t, 86_400, cookie.MaxAge)
					assert.Contains(t, cookie.Value, ".")
				},
			)

			t.Run(
				"Cookie value имеет правильный формат", func(t *testing.T) {
					cookie, err := generateNewCookie()

					require.NoError(t, err)
					assert.NotNil(t, cookie)
					parts := strings.Split(cookie.Value, ".")
					assert.Equal(t, 3, len(parts), "Cookie value должен содержать 3 части, разделенные точкой")
				},
			)
		},
	)

	t.Run(
		"Token Verification", func(t *testing.T) {
			var (
				request *http.Request
				cookie  *http.Cookie
			)

			setup := func(t *testing.T) {
				request = httptest.NewRequest(http.MethodGet, "/", nil)
			}

			t.Run(
				"Невалидный формат - нет разделителя", func(t *testing.T) {
					setup(t)
					cookie = &http.Cookie{Value: "justOnePart"}

					result, err := verifyToken(request, cookie)

					assert.Error(t, err)
					assert.Contains(t, err.Error(), "invalid token format")
					assert.Equal(t, request, result)
				},
			)

			t.Run(
				"Невалидный формат - пустая строка", func(t *testing.T) {
					setup(t)
					cookie = &http.Cookie{Value: ""}

					result, err := verifyToken(request, cookie)

					assert.Error(t, err)
					assert.True(t, errors.Is(err, errInvalidUserFormat))
					assert.Contains(t, err.Error(), "invalid token format")
					assert.Equal(t, request, result)
				},
			)

			t.Run(
				"Невалидный формат - только разделители", func(t *testing.T) {
					setup(t)
					cookie = &http.Cookie{Value: "..."}

					result, err := verifyToken(request, cookie)

					assert.Error(t, err)
					assert.True(t, errors.Is(err, errInvalidUserFormat))
					assert.Contains(t, err.Error(), "invalid token format")
					assert.Equal(t, request, result)
				},
			)

			t.Run(
				"Невалидный base64 в encodedUserID", func(t *testing.T) {
					setup(t)
					cookie = &http.Cookie{Value: "not_valid_base64!!.invalid_signature"}

					result, err := verifyToken(request, cookie)

					assert.Error(t, err)
					assert.Contains(t, err.Error(), "invalid token format")
					assert.Equal(t, request, result)
				},
			)

			t.Run(
				"Проверка типа ошибки InvalidUserFormatError", func(t *testing.T) {
					setup(t)
					invalidCookie := &http.Cookie{Value: "not_valid_base64!!.invalid_signature"}

					_, err := verifyToken(request, invalidCookie)

					require.Error(t, err)
					assert.True(t, errors.Is(err, errInvalidUserFormat))
				},
			)

			t.Run(
				"Успешная верификация с валидным generated cookie", func(t *testing.T) {
					setup(t)
					validCookie, _ := generateNewCookie()

					result, err := verifyToken(request, validCookie)

					require.NoError(t, err)
					assert.NotNil(t, result)

					token := result.Context().Value(userToken)
					assert.NotNil(t, token, "token должен быть установлен в контексте")

					claims, ok := token.(Claims)
					assert.True(t, ok, "UserID должен быть типом Claims")
					assert.NotEmpty(t, claims)
				},
			)
		},
	)

	t.Run(
		"Middleware Integration", func(t *testing.T) {
			var (
				middleware  http.Handler
				nextHandler *mockHandler
				writer      *httptest.ResponseRecorder
				request     *http.Request
			)

			setup := func(t *testing.T) {
				nextHandler = &mockHandler{}
				middleware = Auth(nextHandler)
				writer = httptest.NewRecorder()
				request = httptest.NewRequest(http.MethodGet, "/", nil)
			}

			t.Run(
				"Нет cookie - успешное создание нового", func(t *testing.T) {
					setup(t)

					middleware.ServeHTTP(writer, request)

					assert.Equal(t, http.StatusOK, writer.Code)
					assert.True(t, nextHandler.called, "NextHandler должен быть вызван")

					cookies := writer.Result().Cookies()
					authCookie := findCookie(cookies, "auth")
					assert.NotNil(t, authCookie)
					assert.Equal(t, "auth", authCookie.Name)
					assert.Contains(t, authCookie.Value, ".")
				},
			)

			t.Run(
				"Нет cookie - NextHandler вызывается", func(t *testing.T) {
					setup(t)

					middleware.ServeHTTP(writer, request)

					assert.True(t, nextHandler.called, "NextHandler должен быть вызван при отсутствии cookie")
				},
			)

			t.Run(
				"Нет cookie - Response Header содержит Set-Cookie", func(t *testing.T) {
					setup(t)

					middleware.ServeHTTP(writer, request)

					setCookieHeader := writer.Header().Get("Set-Cookie")
					assert.Contains(t, setCookieHeader, "auth=")
				},
			)

			t.Run(
				"Нет cookie - Response Code не равен 500", func(t *testing.T) {
					setup(t)

					middleware.ServeHTTP(writer, request)

					assert.NotEqual(t, http.StatusInternalServerError, writer.Code)
				},
			)

			t.Run(
				"Существующий валидный cookie", func(t *testing.T) {
					setup(t)

					manualCookie, _ := generateNewCookie()
					request.AddCookie(manualCookie)

					middleware.ServeHTTP(writer, request)

					assert.True(t, nextHandler.called, "NextHandler должен быть вызван с валидным cookie")
					assert.Equal(t, http.StatusOK, writer.Code)
				},
			)

			t.Run(
				"Существующий валидный cookie - UserID в контексте", func(t *testing.T) {
					setup(t)

					manualCookie, _ := generateNewCookie()
					request.AddCookie(manualCookie)

					middleware.ServeHTTP(writer, request)

					assert.True(t, nextHandler.called)
					assert.True(t, nextHandler.contextReceived, "Handler должен получить правильный контекст")
				},
			)

			t.Run(
				"Существующий cookie - неверный формат (нет разделителя)", func(t *testing.T) {
					setup(t)
					invalidCookie := &http.Cookie{Name: "auth", Value: "invalid_format"}
					request.AddCookie(invalidCookie)

					middleware.ServeHTTP(writer, request)

					assert.False(t, nextHandler.called, "NextHandler НЕ должен быть вызван при неверном формате")
					assert.Equal(t, http.StatusUnauthorized, writer.Code)
					assert.Contains(t, writer.Body.String(), "invalid token format")
				},
			)

			t.Run(
				"Существующий cookie - неверная подпись", func(t *testing.T) {
					setup(t)
					invalidCookie := &http.Cookie{Name: "auth", Value: "valid_base64.wrong_signature"}
					request.AddCookie(invalidCookie)

					middleware.ServeHTTP(writer, request)

					assert.False(t, nextHandler.called, "NextHandler НЕ должен быть вызван при неверной подписи")
					assert.Equal(t, http.StatusUnauthorized, writer.Code)
					assert.Contains(t, writer.Body.String(), "invalid token")
				},
			)

			t.Run(
				"Проверка HTTP кода для InvalidUserFormatError", func(t *testing.T) {
					setup(t)
					invalidCookie := &http.Cookie{Name: "auth", Value: "not_valid_base64!!.invalid_signature"}
					request.AddCookie(invalidCookie)

					middleware.ServeHTTP(writer, request)

					assert.False(t, nextHandler.called, "NextHandler НЕ должен быть вызван при неверном формате")
					assert.Equal(t, http.StatusUnauthorized, writer.Code)
					assert.Contains(t, writer.Body.String(), "invalid token")
				},
			)
		},
	)
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

type mockHandler struct {
	called          bool
	contextReceived bool
	responseStatus  int
	responseBody    string
}

func (m *mockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.called = true

	if userID := r.Context().Value(userToken); userID != nil {
		m.contextReceived = true
	} else {
		m.contextReceived = false
	}

	if m.responseStatus == 0 {
		m.responseStatus = http.StatusOK
	}

	w.WriteHeader(m.responseStatus)
	_, _ = w.Write([]byte(m.responseBody))
}
