package request

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type ctxKey string

const (
	correlationKey    = ctxKey("correlation_id")
	correlationHeader = "Correlation-Id"
)

func NewCorrelationID() string {
	return uuid.NewString()
}

func CorrelationIDMiddleware(newCorrelationID func() string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			correlationID := r.Header.Get(correlationHeader)

			if correlationID == "" {
				correlationID = newCorrelationID()
			}

			ctx := context.WithValue(r.Context(), correlationKey, correlationID)

			w.Header().Set(correlationHeader, correlationID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type sizeCodeWriter struct {
	http.ResponseWriter

	StatusCode int
	Size       int
}

func (w *sizeCodeWriter) WriteHeader(statusCode int) {
	w.StatusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *sizeCodeWriter) Write(p []byte) (int, error) {
	w.Size += len(p)

	return w.ResponseWriter.Write(p) //nolint: wrapcheck
}

func LoggerMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hijack := &sizeCodeWriter{
				ResponseWriter: w,
				Size:           0,
				StatusCode:     http.StatusOK,
			}

			defer func(ctx context.Context, start time.Time) {
				logger.With(
					slog.String("method", r.Method),
					slog.String("path", r.RequestURI),
					slog.String("user_agent", r.UserAgent()),
					slog.Int("status", hijack.StatusCode),
					slog.Int("duration_ms", int(time.Since(start).Milliseconds())),
					slog.Int("response_bytes", hijack.Size),
				).InfoContext(ctx, "request complete")
			}(r.Context(), time.Now())

			next.ServeHTTP(hijack, r)
		})
	}
}

type Recorder interface {
	ObserveHTTPRequestDuration(method string, path string, code int, seconds float64)
	ObserveHTTPResponseSize(method string, path string, code int, bytes int64)
}

func MetricsMiddleware(recorder Recorder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hijack := &sizeCodeWriter{
				ResponseWriter: w,
				Size:           0,
				StatusCode:     http.StatusOK,
			}

			defer func(start time.Time) {
				recorder.ObserveHTTPRequestDuration(
					r.Method,
					r.Pattern,
					hijack.StatusCode,
					time.Since(start).Seconds(),
				)
				recorder.ObserveHTTPResponseSize(r.Method, r.Pattern, hijack.StatusCode, int64(hijack.Size))
			}(time.Now())

			next.ServeHTTP(hijack, r)
		})
	}
}
