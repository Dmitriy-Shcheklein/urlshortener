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

func (w gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func WithGzip(h http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
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
					io.WriteString(w, "error while create gzip")
					return
				}
				defer gz.Close()

				w.Header().Set("Content-Encoding", "gzip")
				h.ServeHTTP(gzipWriter{ResponseWriter: w, Writer: gz}, r)

				return
			}

			if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
				reader, err := gzip.NewReader(r.Body)
				if err != nil {
					io.WriteString(w, "error while read gzip")
					return
				}
				r.Body = reader
				h.ServeHTTP(w, r)
				return
			}
			h.ServeHTTP(w, r)
		},
	)
}
