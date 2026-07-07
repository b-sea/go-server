package health

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type Service struct {
	timeout  time.Duration
	checkers []Checker
	recorder Recorder
	logger   *slog.Logger
}

func NewService(timeout time.Duration, options ...ServiceOption) *Service {
	service := &Service{
		timeout:  timeout,
		checkers: make([]Checker, 0),
		recorder: nil,
		logger:   nil,
	}

	for _, option := range options {
		option(service)
	}

	return service
}

func (s *Service) Register(checker Checker) {
	if s.recorder != nil {
		checker = &metricsChecker{
			Checker:  checker,
			recorder: s.recorder,
		}
	}

	if s.logger != nil {
		checker = &loggingChecker{
			Checker:    checker,
			lastStatus: Status("unset"),
			logger:     s.logger,
		}
	}

	s.checkers = append(s.checkers, checker)
}

type Result struct {
	Status       Status
	Dependencies map[string]CheckResult
}

type CheckResult struct {
	Status   Status
	Message  string
	Duration time.Duration
}

func (s *Service) CheckAll(ctx context.Context) Result {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	results := make(
		chan struct {
			CheckResult

			Name string
		},
		len(s.checkers),
	)

	var wg sync.WaitGroup

	for _, checker := range s.checkers {
		wg.Add(1)

		go func(c Checker) {
			defer wg.Done()

			results <- struct {
				CheckResult

				Name string
			}{
				CheckResult: c.HealthCheck(ctx),
				Name:        c.HealthLabel(),
			}
		}(checker)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	health := Result{
		Status:       StatusHealthy,
		Dependencies: make(map[string]CheckResult),
	}

	for result := range results {
		health.Dependencies[result.Name] = result.CheckResult

		if result.Status != StatusUnhealthy {
			health.Status = StatusUnhealthy
		}
	}

	return health
}
