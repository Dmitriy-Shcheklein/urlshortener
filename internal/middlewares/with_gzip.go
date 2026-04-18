package middlewares

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/logger"
)

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func WithGzip(h http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			isAcceptEncoding := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
			isContentEncoding := strings.Contains(r.Header.Get("Content-Encoding"), "gzip")

			if !isContentEncoding && !isAcceptEncoding {
				h.ServeHTTP(w, r)
				return
			}

			responseWriter := w

			if isAcceptEncoding {
				validTypes := map[string]bool{
					"application/json": true,
					"text/plain":       true,
				}

				contentType := r.Header.Get("Content-Type")

				_, ok := validTypes[contentType]
				if !ok {
					h.ServeHTTP(w, r)
					return
				}

				gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
				if err != nil {
					http.Error(w, "error while create gzip", http.StatusBadRequest)
					return
				}
				defer func() {
					if err = gz.Close(); err != nil {
						logger.Logger.Error().Err(err).Msg("error while close gzip writer")
					}
				}()

				w.Header().Set("Content-Encoding", "gzip")
				gzWriter := gzipWriter{ResponseWriter: w, Writer: gz}
				responseWriter = gzWriter
			}

			if isContentEncoding {
				decompressRequest(w, r)
			}
			h.ServeHTTP(responseWriter, r)
		},
	)
}

func decompressRequest(w http.ResponseWriter, r *http.Request) {
	gzReader, err := gzip.NewReader(r.Body)
	if err != nil {
		http.Error(w, "error while read gzip", http.StatusBadRequest)
		return
	}

	decompressed, err := io.ReadAll(gzReader)
	if err != nil {
		http.Error(w, "Failed to decompress", http.StatusBadRequest)
		return
	}
	if err = r.Body.Close(); err != nil {
		logger.Logger.Error().Err(err).Msg("error while clode rea")
	}
	if err = gzReader.Close(); err != nil {
		logger.Logger.Error().Err(err).Msg("error while close gzip reader")
	}

	r.Body = io.NopCloser(bytes.NewReader(decompressed))
	r.Header.Del("Content-Encoding")
	r.ContentLength = int64(len(decompressed))
}
