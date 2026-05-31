package middlewares

import (
	"net/http"
	"strconv"
	"time"

	"github.com/rs/zerolog"
)

type (
	responseData struct {
		status int
		size   int
	}

	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

func (a *AppMiddleware) WithLogging(h http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		uri := r.RequestURI
		method := r.Method
		responseData := &responseData{
			status: 0,
			size:   0,
		}
		lw := loggingResponseWriter{
			ResponseWriter: w,
			responseData:   responseData,
		}
		h.ServeHTTP(&lw, r)
		duration := time.Since(start)
		a.logger.Info().Dict(
			"data", zerolog.Dict().Str("Method", method).Str("URI", uri).Str("Duration", duration.String()),
		).Msg("Request data")
		a.logger.Info().Dict(
			"data", zerolog.Dict().Str("Status", strconv.Itoa(responseData.status)).Str(
				"Body size", strconv.Itoa(responseData.size),
			),
		).Msg("Response data")
	}
	return http.HandlerFunc(logFn)
}
