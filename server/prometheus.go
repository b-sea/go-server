package server

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const subsystem = "http"

// PrometheusOption is a creation option for PrometheusRecorder.
type PrometheusOption func(p *PrometheusRecorder)

// WithGroupedCodes records status codes as their hundreds value: 2xx/4xx/5xx.
func WithGroupedCodes() PrometheusOption {
	return func(p *PrometheusRecorder) {
		p.groupCodes = true
	}
}

// WithRegisterer sets a custom PrometheusRecorder registerer.
func WithRegisterer(registerer prometheus.Registerer) PrometheusOption {
	return func(p *PrometheusRecorder) {
		p.registerer = registerer
	}
}

var _ Recorder = (*PrometheusRecorder)(nil)

// PrometheusRecorder records metrics with PrometheusRecorder.
type PrometheusRecorder struct {
	groupCodes          bool
	registerer          prometheus.Registerer
	httpRequestDuration *prometheus.HistogramVec
	httpResponseSize    *prometheus.HistogramVec
}

// NewPrometheus creates a new PrometheusRecorder.
func NewPrometheus(namespace string, options ...PrometheusOption) *PrometheusRecorder {
	recorder := &PrometheusRecorder{
		groupCodes: false,
		registerer: prometheus.DefaultRegisterer,
		httpRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "request_duration_seconds",
				Help:      "HTTP Request Duration in Seconds",
			},
			[]string{"method", "path", "code"},
		),
		httpResponseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "response_size_bytes",
				Help:      "HTTP Response Size in Bytes",
			},
			[]string{"method", "path", "code"},
		),
	}

	for _, option := range options {
		option(recorder)
	}

	_ = recorder.registerer.Register(recorder.httpRequestDuration)
	_ = recorder.registerer.Register(recorder.httpResponseSize)

	return recorder
}

// Handler returns an http handler for a PrometheusRecorder.
func (p *PrometheusRecorder) Handler() http.Handler {
	return promhttp.Handler()
}

// ObserveHTTPRequestDuration updates the HTTP request duration metric.
func (p *PrometheusRecorder) ObserveHTTPRequestDuration(method string, path string, code int, duration time.Duration) {
	p.httpRequestDuration.WithLabelValues(method, path, p.formatStatusCode(code)).Observe(duration.Seconds())
}

// ObserveHTTPResponseSize updates the HTTP response size metric.
func (p *PrometheusRecorder) ObserveHTTPResponseSize(method string, path string, code int, bytes int64) {
	p.httpResponseSize.WithLabelValues(method, path, p.formatStatusCode(code)).Observe(float64(bytes))
}

func (p *PrometheusRecorder) formatStatusCode(code int) string {
	if !p.groupCodes {
		return strconv.Itoa(code)
	}

	return fmt.Sprintf("%dxx", code/100) //nolint: mnd
}
