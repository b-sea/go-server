// Package metrics implements standard metrics recorders.
package metrics

import (
	"net/http"
	"time"
)

// NoOp is a simple metrics that does nothing.
type NoOp struct{}

// Handler is the logger response to an HTTP request.
func (r *NoOp) Handler() http.Handler {
	return http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
}

// ObserveHTTPRequestDuration records the duration of an HTTP request.
func (r *NoOp) ObserveHTTPRequestDuration(string, string, int, time.Duration) {}

// ObserveHTTPResponseSize records how large an HTTP response is.
func (r *NoOp) ObserveHTTPResponseSize(string, string, int, int64) {}
