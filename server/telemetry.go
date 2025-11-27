package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// Recorder defines functions for tracking HTTP-based metrics.
type Recorder interface {
	Handler() http.Handler
	ObserveHealth(name string, isHealthy bool)
	ObserveRequestDuration(method string, path string, code int, duration time.Duration)
	ObserveResponseSize(method string, path string, code int, bytes int64)
}

type telemetryWriter struct {
	http.ResponseWriter

	StatusCode int
	Size       int
}

func (w *telemetryWriter) WriteHeader(statusCode int) {
	w.StatusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *telemetryWriter) Write(p []byte) (int, error) {
	w.Size += len(p)

	return w.ResponseWriter.Write(p) //nolint: wrapcheck
}

func (s *Server) telemetryMiddleware(recorder Recorder) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			start := time.Now()

			path, err := mux.CurrentRoute(request).GetPathTemplate()
			if err != nil {
				path = request.URL.Path
			}

			hijack := &telemetryWriter{
				ResponseWriter: writer,
				StatusCode:     http.StatusOK,
				Size:           0,
			}

			defer func() {
				panicked := recover()
				if panicked != nil {
					err, ok := panicked.(error)
					if !ok {
						err = fmt.Errorf("%v", panicked) //nolint: err113
					}

					hijack.WriteHeader(http.StatusInternalServerError)
					s.log.Error().Stack().Err(errors.Wrap(err, "http")).Send()
				}

				duration := time.Since(start)

				s.log.Info().
					Str("method", request.Method).
					Str("url", request.URL.RequestURI()).
					Str("user_agent", request.UserAgent()).
					Int("status_code", hijack.StatusCode).
					Dur("duration_ms", duration).
					Int("response_bytes", hijack.Size).
					Msg("request complete")

				recorder.ObserveRequestDuration(request.Method, path, hijack.StatusCode, duration)
				recorder.ObserveResponseSize(request.Method, path, hijack.StatusCode, int64(hijack.Size))
			}()

			// Add a correlation ID
			correlationID := s.newCorrelationID()
			hijack.Header().Add("Correlation-ID", correlationID)
			s.log.UpdateContext(func(c zerolog.Context) zerolog.Context {
				return c.Str("correlation_id", correlationID)
			})

			next.ServeHTTP(hijack, request.WithContext(s.log.WithContext(request.Context())))
		})
	}
}
