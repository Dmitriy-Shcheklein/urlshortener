package middlewares

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/logger"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGzipWriter(t *testing.T) {
	t.Run("Write делегирует базовому ResponseWriter", func(t *testing.T) {
		underlyingWriter := &mockResponseWriter{
			status: http.StatusOK,
			size:   0,
		}

		gzWriter := gzipWriter{
			ResponseWriter: underlyingWriter,
			Writer:         &bytes.Buffer{},
		}

		testData := []byte("test data")
		n, err := gzWriter.Write(testData)

		require.NoError(t, err)
		assert.Equal(t, len(testData), n)
	})

	t.Run("Write возвращает корректные значения", func(t *testing.T) {
		writerBuffer := bytes.NewBufferString("already written")
		underlyingWriter := &mockResponseWriter{
			status: http.StatusOK,
			size:   0,
		}

		gzWriter := gzipWriter{
			ResponseWriter: underlyingWriter,
			Writer:         writerBuffer,
		}

		testData := []byte("new data")
		n, err := gzWriter.Write(testData)

		require.NoError(t, err)
		assert.Equal(t, len(testData), n)
	})

	t.Run("Write с ошибкой базового Writer", func(t *testing.T) {
		errorWriter := &errorWriter{err: assert.AnError}
		underlyingWriter := &mockResponseWriter{
			status: http.StatusOK,
			size:   0,
		}

		gzWriter := gzipWriter{
			ResponseWriter: underlyingWriter,
			Writer:         errorWriter,
		}

		testData := []byte("test data")
		n, err := gzWriter.Write(testData)

		assert.Error(t, err)
		assert.Equal(t, 0, n)
	})
}

func TestDecompressRequest(t *testing.T) {
	t.Run("Успешная декомпрессия", func(t *testing.T) {
		logger.Logger = new(zerolog.Nop())

		originalData := "original data"
		compressedData, err := compressData(originalData)
		require.NoError(t, err)

		request := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(compressedData))
		request.Header.Set("Content-Encoding", "gzip")
		request.ContentLength = int64(len(compressedData))

		writer := httptest.NewRecorder()

		decompressRequest(writer, request)

		decompressed, err := io.ReadAll(request.Body)
		require.NoError(t, err)
		assert.Equal(t, originalData, string(decompressed))
		assert.Equal(t, int64(len(originalData)), request.ContentLength)
		assert.Empty(t, request.Header.Get("Content-Encoding"))
		assert.NotEqual(t, http.StatusBadRequest, writer.Code)
	})

	t.Run("Ошибка создания gzip.NewReader", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("invalid gzip data"))
		request.Header.Set("Content-Encoding", "gzip")

		writer := httptest.NewRecorder()

		decompressRequest(writer, request)

		assert.Equal(t, http.StatusBadRequest, writer.Code)
		assert.Contains(t, writer.Body.String(), "error while read gzip")
	})

	t.Run("Ошибка чтения из gzip reader", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/", &errorReader{err: assert.AnError})
		request.Header.Set("Content-Encoding", "gzip")

		writer := httptest.NewRecorder()

		decompressRequest(writer, request)

		assert.Equal(t, http.StatusBadRequest, writer.Code)
		assert.Contains(t, writer.Body.String(), "error while read gzip")
	})

	t.Run("Проверка удаления Content-Encoding", func(t *testing.T) {
		logger.Logger = new(zerolog.Nop())

		originalData := "test data"
		compressedData, err := compressData(originalData)
		require.NoError(t, err)

		request := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(compressedData))
		request.Header.Set("Content-Encoding", "gzip")

		writer := httptest.NewRecorder()

		decompressRequest(writer, request)

		contentEncoding := request.Header.Get("Content-Encoding")
		assert.Empty(t, contentEncoding, "Content-Encoding должен быть удален")
	})

	t.Run("Проверка обновления ContentLength", func(t *testing.T) {
		logger.Logger = new(zerolog.Nop())

		originalData := "data for length check"
		compressedData, err := compressData(originalData)
		require.NoError(t, err)

		request := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(compressedData))
		request.Header.Set("Content-Encoding", "gzip")

		writer := httptest.NewRecorder()

		decompressRequest(writer, request)

		assert.Equal(t, int64(len(originalData)), request.ContentLength)
	})
}

func TestWithGzipMiddleware(t *testing.T) {
	logger.Logger = new(zerolog.Nop())

	t.Run("Нет gzip заголовков", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("response"))
		})

		middleware := WithGzip(nextHandler)
		writer := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/", nil)

		middleware.ServeHTTP(writer, request)

		assert.Equal(t, http.StatusOK, writer.Code)
		assert.Equal(t, "response", writer.Body.String())
		assert.Empty(t, writer.Header().Get("Content-Encoding"))
	})

	t.Run("Только Accept-Encoding - невалидный Content-Type", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("response"))
		})

		middleware := WithGzip(nextHandler)
		writer := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set("Accept-Encoding", "gzip")
		request.Header.Set("Content-Type", "application/xml")

		middleware.ServeHTTP(writer, request)

		assert.Equal(t, http.StatusOK, writer.Code)
		assert.Equal(t, "response", writer.Body.String())
		assert.Empty(t, writer.Header().Get("Content-Encoding"))
	})

	t.Run("Только Accept-Encoding - валидный Content-Type (application/json)", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"message":"success"}`))
		})

		middleware := WithGzip(nextHandler)
		writer := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set("Accept-Encoding", "gzip")
		request.Header.Set("Content-Type", "application/json")

		middleware.ServeHTTP(writer, request)

		assert.Equal(t, http.StatusOK, writer.Code)
		assert.Equal(t, "gzip", writer.Header().Get("Content-Encoding"))
		assert.NotEmpty(t, writer.Body.Bytes())
	})

	t.Run("Только Accept-Encoding - валидный Content-Type (text/plain)", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("plain text response"))
		})

		middleware := WithGzip(nextHandler)
		writer := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set("Accept-Encoding", "gzip")
		request.Header.Set("Content-Type", "text/plain")

		middleware.ServeHTTP(writer, request)

		assert.Equal(t, http.StatusOK, writer.Code)
		assert.Equal(t, "gzip", writer.Header().Get("Content-Encoding"))
		assert.NotEmpty(t, writer.Body.Bytes())
	})

	t.Run("Только Accept-Encoding - валидный Content-Type - проверка сжатия", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("test data for gzip compression"))
		})

		middleware := WithGzip(nextHandler)
		writer := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set("Accept-Encoding", "gzip")
		request.Header.Set("Content-Type", "application/json")

		middleware.ServeHTTP(writer, request)

		assert.Equal(t, http.StatusOK, writer.Code)
		assert.Equal(t, "gzip", writer.Header().Get("Content-Encoding"))

		compressedData := writer.Body.Bytes()

		reader, err := gzip.NewReader(bytes.NewReader(compressedData))
		require.NoError(t, err)
		defer func() { _ = reader.Close() }()

		decompressed, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, "test data for gzip compression", string(decompressed))
	})

	t.Run("Только Content-Encoding - успешная декомпрессия", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			decompressed, err := io.ReadAll(r.Body)
			require.NoError(t, err)

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("received: " + string(decompressed)))
		})

		middleware := WithGzip(nextHandler)
		writer := httptest.NewRecorder()

		originalData := "compressed request data"
		compressedData, err := compressData(originalData)
		require.NoError(t, err)

		request := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(compressedData))
		request.Header.Set("Content-Encoding", "gzip")
		request.Header.Set("Content-Type", "text/plain")

		middleware.ServeHTTP(writer, request)

		assert.Equal(t, http.StatusOK, writer.Code)
		assert.Contains(t, writer.Body.String(), originalData)
		assert.Empty(t, writer.Header().Get("Content-Encoding"))
	})

	t.Run("Только Content-Encoding - ошибка декомпрессии", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := WithGzip(nextHandler)
		writer := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("invalid gzip"))
		request.Header.Set("Content-Encoding", "gzip")

		middleware.ServeHTTP(writer, request)

		assert.Equal(t, http.StatusBadRequest, writer.Code)
		assert.Contains(t, writer.Body.String(), "error while read gzip")
	})

	t.Run("Одновременный Accept-Encoding и Content-Encoding", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			decompressed, err := io.ReadAll(r.Body)
			require.NoError(t, err)

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("received: " + string(decompressed)))
		})

		middleware := WithGzip(nextHandler)
		writer := httptest.NewRecorder()

		originalData := "compressed request"
		compressedRequest, err := compressData(originalData)
		require.NoError(t, err)

		request := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(compressedRequest))
		request.Header.Set("Content-Encoding", "gzip")
		request.Header.Set("Accept-Encoding", "gzip")
		request.Header.Set("Content-Type", "application/json")

		middleware.ServeHTTP(writer, request)

		assert.Equal(t, http.StatusOK, writer.Code)
		assert.Contains(t, writer.Body.String(), originalData)
		assert.Equal(t, "gzip", writer.Header().Get("Content-Encoding"))
	})

	t.Run("Проверка gzipWriter.Write интерфейса", func(t *testing.T) {
		testData := []byte("test data for interface check")

		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(testData)
		})

		middleware := WithGzip(nextHandler)
		writer := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set("Accept-Encoding", "gzip")
		request.Header.Set("Content-Type", "text/plain")

		middleware.ServeHTTP(writer, request)

		assert.Equal(t, http.StatusOK, writer.Code)
		assert.Equal(t, "gzip", writer.Header().Get("Content-Encoding"))

		reader, err := gzip.NewReader(bytes.NewReader(writer.Body.Bytes()))
		require.NoError(t, err)
		defer func() { _ = reader.Close() }()

		receivedData, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, string(testData), string(receivedData))
	})

	t.Run("Проверка Content-Encoding установлен только при Accept-Encoding", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("response"))
		})

		middleware := WithGzip(nextHandler)
		writer := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set("Content-Encoding", "gzip")
		request.Header.Set("Content-Type", "application/json")

		originalData := "compressed request"
		compressedData, err := compressData(originalData)
		require.NoError(t, err)

		request.Body = io.NopCloser(bytes.NewReader(compressedData))

		middleware.ServeHTTP(writer, request)

		assert.Equal(t, http.StatusOK, writer.Code)
		assert.Empty(t, writer.Header().Get("Content-Encoding"), "Content-Encoding должен быть установлен только для response")
	})
}

func TestGzipMiddlewareIntegration(t *testing.T) {
	logger.Logger = new(zerolog.Nop())

	t.Run("Интеграция с реальным HTTP", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			decompressedContent, _ := io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"received":"` + string(decompressedContent) + `"}`))
		})

		middleware := WithGzip(nextHandler)
		writer := httptest.NewRecorder()

		originalData := "compressed test data"
		compressedData, err := compressData(originalData)
		require.NoError(t, err)

		request := httptest.NewRequest(http.MethodPost, "/api", bytes.NewReader(compressedData))
		request.Header.Set("Content-Encoding", "gzip")
		request.Header.Set("Accept-Encoding", "gzip")
		request.Header.Set("Content-Type", "application/json")

		middleware.ServeHTTP(writer, request)

		assert.Equal(t, http.StatusOK, writer.Code)
		assert.Equal(t, "gzip", writer.Header().Get("Content-Encoding"))

		reader, err := gzip.NewReader(bytes.NewReader(writer.Body.Bytes()))
		require.NoError(t, err)
		defer func() { _ = reader.Close() }()

		receivedData, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Contains(t, string(receivedData), originalData)
	})

	t.Run("Разные Content-Type работают корректно", func(t *testing.T) {
		testCases := []struct {
			contentType    string
			shouldCompress bool
			expectedHeader string
		}{
			{"application/json", true, "gzip"},
			{"text/plain", true, "gzip"},
			{"application/xml", false, ""},
			{"text/html", false, ""},
			{"application/octet-stream", false, ""},
		}

		for _, tc := range testCases {
			t.Run(tc.contentType, func(t *testing.T) {
				nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("response"))
				})

				middleware := WithGzip(nextHandler)
				writer := httptest.NewRecorder()
				request := httptest.NewRequest(http.MethodGet, "/", nil)
				request.Header.Set("Accept-Encoding", "gzip")
				request.Header.Set("Content-Type", tc.contentType)

				middleware.ServeHTTP(writer, request)

				assert.Equal(t, tc.expectedHeader, writer.Header().Get("Content-Encoding"))
			})
		}
	})

	t.Run("Многократное compression через middleware", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("multiple response part 1 "))
			_, _ = w.Write([]byte("part 2 "))
			_, _ = w.Write([]byte("part 3"))
		})

		middleware := WithGzip(nextHandler)
		writer := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set("Accept-Encoding", "gzip")
		request.Header.Set("Content-Type", "text/plain")

		middleware.ServeHTTP(writer, request)

		assert.Equal(t, http.StatusOK, writer.Code)
		assert.Equal(t, "gzip", writer.Header().Get("Content-Encoding"))

		reader, err := gzip.NewReader(bytes.NewReader(writer.Body.Bytes()))
		require.NoError(t, err)
		defer func() { _ = reader.Close() }()

		receivedData, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, "multiple response part 1 part 2 part 3", string(receivedData))
	})
}

func compressData(data string) ([]byte, error) {
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	if _, err := gzWriter.Write([]byte(data)); err != nil {
		return nil, err
	}
	if err := gzWriter.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type errorWriter struct {
	err error
}

func (w *errorWriter) Write(p []byte) (int, error) {
	return 0, w.err
}

type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (int, error) {
	return 0, r.err
}

type mockResponseWriter struct {
	writeResult struct {
		n   int
		err error
	}
	status int
	size   int
}

func (m *mockResponseWriter) Write(b []byte) (int, error) {
	m.size += len(b)
	return m.writeResult.n, m.writeResult.err
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {
	m.status = statusCode
}

func (m *mockResponseWriter) Header() http.Header {
	return http.Header{}
}
