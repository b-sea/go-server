package server

import (
	"net/http"
	"time"
)

var _ Recorder = (*NoOpRecorder)(nil)

// NoOpRecorder is a simple metrics recorder that does nothing.
type NoOpRecorder struct{}

// Handler is the logger response to an HTTP request.
func (r *NoOpRecorder) Handler() http.Handler {
	return http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
}

// ObserveHTTPRequestDuration records the duration of an HTTP request.
func (r *NoOpRecorder) ObserveHTTPRequestDuration(string, string, int, time.Duration) {}

// ObserveHTTPResponseSize records how large an HTTP response is.
func (r *NoOpRecorder) ObserveHTTPResponseSize(string, string, int, int64) {}
