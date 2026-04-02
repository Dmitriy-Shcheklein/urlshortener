package middlewares

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w *gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func WithGzip(h http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				h.ServeHTTP(w, r)
				return
			}

			validTypes := map[string]bool{
				"application/json": true,
				"text/html":        true,
			}

			contentType := r.Header.Get("Content-Type")

			_, ok := validTypes[contentType]
			if !ok {
				h.ServeHTTP(w, r)
				return
			}

			cw := &gzipWriter{
				ResponseWriter: w,
			}

			gz, err := gzip.NewWriterLevel(cw, gzip.BestSpeed)
			if err != nil {
				io.WriteString(cw, err.Error())
				return
			}
			defer gz.Close()

			w.Header().Set("Content-Encoding", "gzip")
			h.ServeHTTP(cw, r)
		},
	)
}
