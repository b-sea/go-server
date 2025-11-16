package mock

import (
	"net/http"
	"time"

	"github.com/b-sea/go-server/server"
)

var (
	_ server.Recorder = (*NoOp)(nil)
)

type NoOp struct{}

func NewNoOp() *NoOp {
	return &NoOp{}
}

func (r *NoOp) Handler() http.Handler {
	return http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
}

func (r *NoOp) ObserveHealth(string, bool) {}

func (r *NoOp) ObserveRequestDuration(string, string, int, time.Duration) {}

func (r *NoOp) ObserveResponseSize(string, string, int, int64) {}
