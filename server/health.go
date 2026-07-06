package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type HealthChecker interface {
	HealthLabel() string
	HealthCheck(ctx context.Context) HealthCheckResult
}

type loggingHealthChecker struct {
	HealthChecker

	lastStatus HealthStatus
	logger     *slog.Logger
}

func (c *loggingHealthChecker) HealthCheck(ctx context.Context) HealthCheckResult {
	result := c.HealthChecker.HealthCheck(ctx)

	if c.lastStatus != result.Status {
		c.logger.With(
			slog.String("before", string(c.lastStatus)),
			slog.String("current", string(result.Status)),
			slog.Int("duration_ms", int(result.Duration.Milliseconds())),
		).InfoContext(ctx, "health status changed")

		c.lastStatus = result.Status
	}

	return result
}

type HealthRecorder interface {
	SetHealthStatus(name string, status float64)
	ObserveHealthDuration(name string, seconds float64)
}

type metricsHealthChecker struct {
	HealthChecker

	recorder HealthRecorder
}

func (c *metricsHealthChecker) HealthCheck(ctx context.Context) HealthCheckResult {
	result := c.HealthChecker.HealthCheck(ctx)

	c.recorder.ObserveHealthDuration(c.HealthLabel(), result.Duration.Seconds())

	status := 0.0
	if result.Status == Healthy {
		status = 1.0
	}

	c.recorder.SetHealthStatus(c.HealthLabel(), status)

	return result
}

type HealthServiceOption func(h *HealthService)

func WithHealthLogger(logger *slog.Logger) HealthServiceOption {
	return func(h *HealthService) {
		h.logger = logger
	}
}

func WithHealthRecorder(recorder HealthRecorder) HealthServiceOption {
	return func(h *HealthService) {
		h.recorder = recorder
	}
}

type HealthService struct {
	timeout  time.Duration
	checkers []HealthChecker
	recorder HealthRecorder
	logger   *slog.Logger
}

func NewHealthService(timeout time.Duration, options ...HealthServiceOption) *HealthService {
	service := &HealthService{
		timeout:  timeout,
		checkers: make([]HealthChecker, 0),
		recorder: nil,
		logger:   nil,
	}

	for _, option := range options {
		option(service)
	}

	return service
}

func (s *HealthService) Register(checker HealthChecker) {
	if s.recorder != nil {
		checker = &metricsHealthChecker{
			HealthChecker: checker,
			recorder:      s.recorder,
		}
	}

	if s.logger != nil {
		checker = &loggingHealthChecker{
			HealthChecker: checker,
			lastStatus:    HealthStatus("unset"),
			logger:        s.logger,
		}
	}

	s.checkers = append(s.checkers, checker)
}

type HealthResult struct {
	Status       HealthStatus
	Dependencies map[string]HealthCheckResult
}

type HealthCheckResult struct {
	Status   HealthStatus
	Message  string
	Duration time.Duration
}

func (s *HealthService) CheckHealth(ctx context.Context) HealthResult {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	results := make(
		chan struct {
			HealthCheckResult

			Name string
		},
		len(s.checkers),
	)

	var wg sync.WaitGroup

	for _, checker := range s.checkers {
		wg.Add(1)

		go func(c HealthChecker) {
			defer wg.Done()

			results <- struct {
				HealthCheckResult

				Name string
			}{
				HealthCheckResult: c.HealthCheck(ctx),
				Name:              c.HealthLabel(),
			}
		}(checker)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	health := HealthResult{
		Status:       Healthy,
		Dependencies: make(map[string]HealthCheckResult),
	}

	for result := range results {
		health.Dependencies[result.Name] = result.HealthCheckResult

		if result.Status != Healthy {
			health.Status = Unhealthy
		}
	}

	return health
}

func HealthHandler(service *HealthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")

		healthCheck := service.CheckHealth(r.Context())

		response := HealthResponse{
			Status:       healthCheck.Status,
			Dependencies: make(map[string]CheckResponse),
		}

		for name, check := range healthCheck.Dependencies {
			response.Dependencies[name] = CheckResponse{
				Status:  check.Status,
				Message: check.Message,
			}
		}

		if healthCheck.Status != Healthy {
			w.WriteHeader(http.StatusInternalServerError)
		}

		_ = json.NewEncoder(w).Encode(&response)
	}
}
