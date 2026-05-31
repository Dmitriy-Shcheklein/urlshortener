package middlewares

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/logger"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoggingResponseWriter(t *testing.T) {
	logger.Logger = new(zerolog.Nop())
	t.Run("Write успешно делегирует базовому ResponseWriter", func(t *testing.T) {
		underlyingWriter := httptest.NewRecorder()
		responseData := &responseData{status: 0, size: 0}
		loggingWriter := loggingResponseWriter{
			ResponseWriter: underlyingWriter,
			responseData:   responseData,
		}

		testData := []byte("test data")
		n, err := loggingWriter.Write(testData)

		require.NoError(t, err)
		assert.Equal(t, len(testData), n)
		assert.Equal(t, testData, underlyingWriter.Body.Bytes())
	})

	t.Run("Write возвращает корректные значения", func(t *testing.T) {
		underlyingWriter := &loggingMockResponseWriter{
			writeResult: struct {
				n   int
				err error
			}{
				n:   10,
				err: nil,
			},
		}
		responseData := &responseData{status: 0, size: 0}
		loggingWriter := loggingResponseWriter{
			ResponseWriter: underlyingWriter,
			responseData:   responseData,
		}

		testData := []byte("test data")
		n, err := loggingWriter.Write(testData)

		require.NoError(t, err)
		assert.Equal(t, 10, n)
	})

	t.Run("Write накапливает размер", func(t *testing.T) {
		underlyingWriter := httptest.NewRecorder()
		responseData := &responseData{status: 0, size: 0}
		loggingWriter := loggingResponseWriter{
			ResponseWriter: underlyingWriter,
			responseData:   responseData,
		}

		testData1 := []byte("first ")
		testData2 := []byte("second")
		testData3 := []byte("third")

		_, _ = loggingWriter.Write(testData1)
		_, _ = loggingWriter.Write(testData2)
		_, _ = loggingWriter.Write(testData3)

		assert.Equal(t, len(testData1)+len(testData2)+len(testData3), responseData.size)
		assert.Equal(t, len(testData1)+len(testData2)+len(testData3), len(underlyingWriter.Body.Bytes()))
	})

	t.Run("Write с ошибкой", func(t *testing.T) {
		expectedErr := assert.AnError
		underlyingWriter := &loggingMockResponseWriter{
			writeResult: struct {
				n   int
				err error
			}{
				n:   0,
				err: expectedErr,
			},
		}
		responseData := &responseData{status: 0, size: 0}
		loggingWriter := loggingResponseWriter{
			ResponseWriter: underlyingWriter,
			responseData:   responseData,
		}

		testData := []byte("test data")
		n, err := loggingWriter.Write(testData)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, 0, n)
		assert.Equal(t, 0, responseData.size, "size не должен увеличиваться при ошибке")
	})

	t.Run("WriteHeader устанавливает статус", func(t *testing.T) {
		underlyingWriter := httptest.NewRecorder()
		responseData := &responseData{status: 0, size: 0}
		loggingWriter := loggingResponseWriter{
			ResponseWriter: underlyingWriter,
			responseData:   responseData,
		}

		loggingWriter.WriteHeader(http.StatusOK)

		assert.Equal(t, http.StatusOK, responseData.status)
		assert.Equal(t, http.StatusOK, underlyingWriter.Code)
	})

	t.Run("WriteHeader вызывается один раз", func(t *testing.T) {
		mockWriter := &writeHeaderCounter{ResponseWriter: httptest.NewRecorder()}
		responseData := &responseData{status: 0, size: 0}
		loggingWriter := loggingResponseWriter{
			ResponseWriter: mockWriter,
			responseData:   responseData,
		}

		loggingWriter.WriteHeader(http.StatusOK)
		loggingWriter.WriteHeader(http.StatusNotFound)

		assert.Equal(t, 2, mockWriter.writeHeaderCalls, "Каждый вызов должен делегироваться")
		assert.Equal(t, http.StatusNotFound, responseData.status, "Последний статус должен быть сохранен")
	})
}

func TestWithLoggingMiddleware(t *testing.T) {
	logger.Logger = new(zerolog.Nop())

	t.Run("Успешный ответ (200 OK)", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("success"))
		})

		middleware := WithLogging(nextHandler)
		writer := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/test", nil)

		middleware.ServeHTTP(writer, request)

		assert.Equal(t, http.StatusOK, writer.Code)
		assert.Equal(t, "success", writer.Body.String())
	})

	t.Run("Ответ с ошибкой (404 Not Found)", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("not found"))
		})

		middleware := WithLogging(nextHandler)
		writer := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/notfound", nil)

		middleware.ServeHTTP(writer, request)

		assert.Equal(t, http.StatusNotFound, writer.Code)
		assert.Equal(t, "not found", writer.Body.String())
	})

	t.Run("Ответ с ошибкой (500 Internal Server Error)", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("server error"))
		})

		middleware := WithLogging(nextHandler)
		writer := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/error", nil)

		middleware.ServeHTTP(writer, request)

		assert.Equal(t, http.StatusInternalServerError, writer.Code)
		assert.Equal(t, "server error", writer.Body.String())
	})

	t.Run("Пустой ответ (204 No Content)", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})

		middleware := WithLogging(nextHandler)
		writer := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodDelete, "/resource", nil)

		middleware.ServeHTTP(writer, request)

		assert.Equal(t, http.StatusNoContent, writer.Code)
		assert.Equal(t, "", writer.Body.String())
	})

	t.Run("Ответ разными методами (GET)", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			w.WriteHeader(http.StatusOK)
		})

		middleware := WithLogging(nextHandler)
		writer := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api", nil)

		middleware.ServeHTTP(writer, request)

		assert.Equal(t, http.StatusOK, writer.Code)
	})

	t.Run("Ответ разными методами (POST)", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			w.WriteHeader(http.StatusCreated)
		})

		middleware := WithLogging(nextHandler)
		writer := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/create", nil)

		middleware.ServeHTTP(writer, request)

		assert.Equal(t, http.StatusCreated, writer.Code)
	})

	t.Run("Ответ разными методами (PUT)", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPut, r.Method)
			w.WriteHeader(http.StatusOK)
		})

		middleware := WithLogging(nextHandler)
		writer := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPut, "/update", nil)

		middleware.ServeHTTP(writer, request)

		assert.Equal(t, http.StatusOK, writer.Code)
	})

	t.Run("Ответ разными методами (DELETE)", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodDelete, r.Method)
			w.WriteHeader(http.StatusNoContent)
		})

		middleware := WithLogging(nextHandler)
		writer := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodDelete, "/delete", nil)

		middleware.ServeHTTP(writer, request)

		assert.Equal(t, http.StatusNoContent, writer.Code)
	})

	t.Run("Ответ на разные URI", func(t *testing.T) {
		testCases := []struct {
			uri          string
			expectedBody string
			expectedCode int
		}{
			{"/api/v1/shorten", "shorten response", http.StatusOK},
			{"/", "home page", http.StatusOK},
			{"/health", "health ok", http.StatusOK},
		}

		for _, tc := range testCases {
			t.Run("URI: "+tc.uri, func(t *testing.T) {
				nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, tc.uri, r.RequestURI)
					w.WriteHeader(tc.expectedCode)
					_, _ = w.Write([]byte(tc.expectedBody))
				})

				middleware := WithLogging(nextHandler)
				writer := httptest.NewRecorder()
				request := httptest.NewRequest(http.MethodGet, tc.uri, nil)

				middleware.ServeHTTP(writer, request)

				assert.Equal(t, tc.expectedCode, writer.Code)
				assert.Equal(t, tc.expectedBody, writer.Body.String())
			})
		}
	})

	t.Run("Ответ с большим body", func(t *testing.T) {
		largeData := strings.Repeat("data ", 1000)

		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(largeData))
		})

		middleware := WithLogging(nextHandler)
		writer := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/large", nil)

		middleware.ServeHTTP(writer, request)

		assert.Equal(t, http.StatusOK, writer.Code)
		assert.Equal(t, largeData, writer.Body.String())
		assert.Equal(t, len(largeData), writer.Body.Len())
	})

	t.Run("Ответ с пустым body (только header)", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := WithLogging(nextHandler)
		writer := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/no-body", nil)

		middleware.ServeHTTP(writer, request)

		assert.Equal(t, http.StatusOK, writer.Code)
		assert.Equal(t, "", writer.Body.String())
		assert.Equal(t, 0, writer.Body.Len())
	})

	t.Run("Handler вызывается с правильным ResponseWriter", func(t *testing.T) {
		var receivedWriter http.ResponseWriter
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedWriter = w
			_, isLoggingWriter := w.(*loggingResponseWriter)
			assert.True(t, isLoggingWriter, "Должен быть передан loggingResponseWriter")
			w.WriteHeader(http.StatusOK)
		})

		middleware := WithLogging(nextHandler)
		writer := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/test", nil)

		middleware.ServeHTTP(writer, request)

		assert.NotNil(t, receivedWriter)
	})

	t.Run("Проверка что loggingResponseWriter делегирует базовому ResponseWriter", func(t *testing.T) {
		var headersSet map[string]string
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("data"))
		})

		middleware := WithLogging(nextHandler)
		writer := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/delegate", nil)

		middleware.ServeHTTP(writer, request)

		headersSet = map[string]string{}
		for key, values := range writer.Header() {
			if len(values) > 0 {
				headersSet[key] = values[0]
			}
		}

		assert.Equal(t, "application/json", headersSet["Content-Type"])
		assert.Equal(t, http.StatusOK, writer.Code)
		assert.Equal(t, "data", writer.Body.String())
	})

	t.Run("Проверка захвата ResponseWriter", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if lw, ok := w.(*loggingResponseWriter); ok {
				assert.NotNil(t, lw.responseData, "responseData должен быть инициализирован")
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("test response"))
		})

		middleware := WithLogging(nextHandler)
		writer := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/create", nil)

		middleware.ServeHTTP(writer, request)

		assert.Equal(t, http.StatusCreated, writer.Code)
		assert.Equal(t, "test response", writer.Body.String())
	})
}

func TestLoggingResponseWriterIntegration(t *testing.T) {
	t.Run("Интеграция с middlware - реальные HTTP сценарий", func(t *testing.T) {
		logger.Logger = new(zerolog.Nop())
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"message":"success"}`))
		})

		middleware := WithLogging(nextHandler)
		writer := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/test", nil)

		middleware.ServeHTTP(writer, request)

		assert.Equal(t, http.StatusOK, writer.Code)
		assert.Equal(t, "application/json", writer.Header().Get("Content-Type"))
		assert.Equal(t, `{"message":"success"}`, writer.Body.String())
	})

	t.Run("Многократный вызов Write", func(t *testing.T) {
		logger.Logger = new(zerolog.Nop())
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("part1 "))
			_, _ = w.Write([]byte("part2 "))
			_, _ = w.Write([]byte("part3"))
		})

		middleware := WithLogging(nextHandler)
		writer := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/multiple", nil)

		middleware.ServeHTTP(writer, request)

		assert.Equal(t, http.StatusOK, writer.Code)
		assert.Equal(t, "part1 part2 part3", writer.Body.String())
	})

	t.Run("Different HTTP methods work correctly", func(t *testing.T) {
		logger.Logger = new(zerolog.Nop())
		methods := []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodPatch,
			http.MethodOptions,
			http.MethodHead,
		}

		for _, method := range methods {
			t.Run(method, func(t *testing.T) {
				nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, method, r.Method)
					w.WriteHeader(http.StatusOK)
				})

				middleware := WithLogging(nextHandler)
				writer := httptest.NewRecorder()
				request := httptest.NewRequest(method, "/method", nil)

				middleware.ServeHTTP(writer, request)

				assert.Equal(t, http.StatusOK, writer.Code)
			})
		}
	})
}

type loggingMockResponseWriter struct {
	writeResult struct {
		n   int
		err error
	}
	status int
	size   int
}

func (m *loggingMockResponseWriter) Write(b []byte) (int, error) {
	m.size += len(b)
	return m.writeResult.n, m.writeResult.err
}

func (m *loggingMockResponseWriter) WriteHeader(statusCode int) {
	m.status = statusCode
}

func (m *loggingMockResponseWriter) Header() http.Header {
	return http.Header{}
}

type writeHeaderCounter struct {
	http.ResponseWriter
	writeHeaderCalls int
}

func (w *writeHeaderCounter) WriteHeader(statusCode int) {
	w.writeHeaderCalls++
	w.ResponseWriter.WriteHeader(statusCode)
}
