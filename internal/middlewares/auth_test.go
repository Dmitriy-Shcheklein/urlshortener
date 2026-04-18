package middlewares

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateNewCookie(t *testing.T) {
	t.Run(
		"Успешное создание cookie с корректными параметрами", func(t *testing.T) {
			cookie, userID := generateNewCookie()

			assert.NotNil(t, cookie)
			assert.NotNil(t, userID)
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
			cookie, userID := generateNewCookie()

			assert.NotNil(t, cookie)
			assert.NotNil(t, userID)

			parts := strings.Split(cookie.Value, ".")
			assert.Equal(t, 2, len(parts), "Cookie value должен содержать 2 части, разделенные точкой")
			assert.NotEmpty(t, parts[0], "Первая часть (encodedUserID) не должна быть пустой")
			assert.NotEmpty(t, parts[1], "Вторая часть (signature) не должна быть пустой")
		},
	)
}

func TestVerifyToken(t *testing.T) {
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
			assert.Contains(t, err.Error(), "invalid cookie format")
			assert.Equal(t, request, result)
		},
	)

	t.Run(
		"Невалидный формат - пустая строка", func(t *testing.T) {
			setup(t)
			cookie = &http.Cookie{Value: ""}

			result, err := verifyToken(request, cookie)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid cookie format")
			assert.Equal(t, request, result)
		},
	)

	t.Run(
		"Невалидный формат - несколько разделителей", func(t *testing.T) {
			setup(t)
			cookie = &http.Cookie{Value: "part1.part2.part3"}

			result, err := verifyToken(request, cookie)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid cookie format")
			assert.Equal(t, request, result)
		},
	)

	t.Run(
		"Невалидный формат - только разделители", func(t *testing.T) {
			setup(t)
			cookie = &http.Cookie{Value: ".."}

			result, err := verifyToken(request, cookie)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid cookie format")
			assert.Equal(t, request, result)
		},
	)

	t.Run(
		"Невалидная подпись HMAC", func(t *testing.T) {
			setup(t)
			testUserID := "test_user_id"
			encodedUserID := base64.URLEncoding.EncodeToString([]byte(testUserID))

			h := hmac.New(sha256.New, secretKey)
			h.Write([]byte("WRONG_USER_ID"))
			incorrectSignature := base64.URLEncoding.EncodeToString(h.Sum(nil))

			testCookie := &http.Cookie{
				Name:  "auth",
				Value: encodedUserID + "." + incorrectSignature,
			}

			result, err := verifyToken(request, testCookie)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid signature")
			assert.Equal(t, request, result)
		},
	)

	t.Run(
		"Невалидный base64 в encodedUserID", func(t *testing.T) {
			setup(t)
			cookie = &http.Cookie{Value: "not_valid_base64!!.invalid_signature"}

			result, err := verifyToken(request, cookie)

			assert.Error(t, err)
			assert.Equal(t, request, result)
		},
	)

	t.Run(
		"Успешная верификация с валидным generated cookie", func(t *testing.T) {
			setup(t)
			validCookie, _ := generateNewCookie()

			result, err := verifyToken(request, validCookie)

			require.NoError(t, err)
			assert.NotNil(t, result)

			userID := result.Context().Value(UserIDKey)
			assert.NotNil(t, userID, "UserID должен быть установлен в контексте")

			userIDBytes, ok := userID.([]byte)
			assert.True(t, ok, "UserID должен быть типом []byte")
			assert.NotEmpty(t, userIDBytes)
		},
	)

	t.Run(
		"Успешная верификация с вручную созданным валидным cookie", func(t *testing.T) {
			setup(t)
			manualCookie, _ := generateNewCookie()

			sections := strings.Split(manualCookie.Value, ".")
			encodedUserID := sections[0]

			userIDBytes, err := base64.URLEncoding.DecodeString(encodedUserID)
			require.NoError(t, err)

			h := hmac.New(sha256.New, secretKey)
			h.Write(userIDBytes)
			correctSignature := base64.URLEncoding.EncodeToString(h.Sum(nil))

			correctCookie := &http.Cookie{
				Name:  "auth",
				Value: encodedUserID + "." + correctSignature,
			}

			result, err := verifyToken(request, correctCookie)

			require.NoError(t, err)
			assert.NotNil(t, result)

			userID := result.Context().Value(UserIDKey)
			assert.NotNil(t, userID, "UserID должен быть установлен в контексте")

			decodedUserIDBytes, ok := userID.([]byte)
			assert.True(t, ok, "UserID должен быть типом []byte")
			assert.NotEmpty(t, decodedUserIDBytes)
		},
	)
}

func TestAuthMiddleware(t *testing.T) {
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
			assert.Equal(t, http.StatusInternalServerError, writer.Code)
			assert.Contains(t, writer.Body.String(), "error while verify cookie")
			assert.Contains(t, writer.Body.String(), "invalid cookie format")
		},
	)

	t.Run(
		"Существующий cookie - неверная подпись", func(t *testing.T) {
			setup(t)
			invalidCookie := &http.Cookie{Name: "auth", Value: "valid_base64.wrong_signature"}
			request.AddCookie(invalidCookie)

			middleware.ServeHTTP(writer, request)

			assert.False(t, nextHandler.called, "NextHandler НЕ должен быть вызван при неверной подписи")
			assert.Equal(t, http.StatusInternalServerError, writer.Code)
			assert.Contains(t, writer.Body.String(), "error while verify cookie")
			assert.Contains(t, writer.Body.String(), "invalid signature")
		},
	)

	t.Run(
		"Checker для извлечения UserID из контекста", func(t *testing.T) {
			setup(t)

			manualCookie, _ := generateNewCookie()
			request.AddCookie(manualCookie)

			handlerWithCheck := http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					userID := r.Context().Value(UserIDKey)
					if userID != nil {
						if userIDBytes, ok := userID.([]byte); ok && len(userIDBytes) > 0 {
							w.WriteHeader(http.StatusOK)
							_, _ = w.Write([]byte("user_id_found"))
						}
					} else {
						w.WriteHeader(http.StatusInternalServerError)
					}
				},
			)
			middleware = Auth(handlerWithCheck)

			middleware.ServeHTTP(writer, request)

			assert.Equal(t, http.StatusOK, writer.Code)
			assert.Contains(t, writer.Body.String(), "user_id_found")
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

	if userID := r.Context().Value(UserIDKey); userID != nil {
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
