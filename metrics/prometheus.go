package metrics

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/b-sea/go-server/server"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const subsystem = "http"

// PrometheusOption is a creation option for Prometheus.
type PrometheusOption func(p *Prometheus)

// WithGroupedCodes records status codes as their hundreds value: 2xx/4xx/5xx.
func WithGroupedCodes() PrometheusOption {
	return func(p *Prometheus) {
		p.groupCodes = true
	}
}

// WithRegisterer sets a custom Prometheus registerer.
func WithRegisterer(registerer prometheus.Registerer) PrometheusOption {
	return func(p *Prometheus) {
		p.registerer = registerer
	}
}

var _ server.Recorder = (*Prometheus)(nil)

// Prometheus records metrics with Prometheus.
type Prometheus struct {
	groupCodes          bool
	registerer          prometheus.Registerer
	httpRequestDuration *prometheus.HistogramVec
	httpResponseSize    *prometheus.HistogramVec
}

// NewPrometheus creates a new Prometheus recorder.
func NewPrometheus(namespace string, options ...PrometheusOption) *Prometheus {
	recorder := &Prometheus{
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

// Handler returns an http handler for Prometheus.
func (p *Prometheus) Handler() http.Handler {
	return promhttp.Handler()
}

// ObserveHTTPRequestDuration updates the HTTP request duration metric.
func (p *Prometheus) ObserveHTTPRequestDuration(method string, path string, code int, duration time.Duration) {
	p.httpRequestDuration.WithLabelValues(method, path, p.formatStatusCode(code)).Observe(duration.Seconds())
}

// ObserveHTTPResponseSize updates the HTTP response size metric.
func (p *Prometheus) ObserveHTTPResponseSize(method string, path string, code int, bytes int64) {
	p.httpResponseSize.WithLabelValues(method, path, p.formatStatusCode(code)).Observe(float64(bytes))
}

func (p *Prometheus) formatStatusCode(code int) string {
	if !p.groupCodes {
		return strconv.Itoa(code)
	}

	return fmt.Sprintf("%dxx", code/100) //nolint: mnd
}
