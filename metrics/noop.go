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

// ObserveHealth records the health of a named service.
func (r *NoOp) ObserveHealth(string, bool) {}

// ObserveRequestDuration records the duration of an HTTP request.
func (r *NoOp) ObserveRequestDuration(string, string, int, time.Duration) {}

// ObserveResponseSize records how large an HTTP response is.
func (r *NoOp) ObserveResponseSize(string, string, int, int64) {}
