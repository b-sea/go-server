package health

import (
	"context"
	"log/slog"
)

type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
)

type Checker interface {
	HealthLabel() string
	HealthCheck(ctx context.Context) CheckResult
}

type loggingChecker struct {
	Checker

	lastStatus Status
	logger     *slog.Logger
}

func (c *loggingChecker) HealthCheck(ctx context.Context) CheckResult {
	result := c.Checker.HealthCheck(ctx)

	if c.lastStatus != result.Status {
		c.logger.With(
			slog.String("previous", string(c.lastStatus)),
			slog.String("current", string(result.Status)),
			slog.Int("duration_ms", int(result.Duration.Milliseconds())),
		).InfoContext(ctx, "health status changed")

		c.lastStatus = result.Status
	}

	return result
}

type Recorder interface {
	SetHealthStatus(name string, status float64)
	ObserveHealthDuration(name string, seconds float64)
}

type metricsChecker struct {
	Checker

	recorder Recorder
}

func (c *metricsChecker) HealthCheck(ctx context.Context) CheckResult {
	result := c.Checker.HealthCheck(ctx)

	c.recorder.ObserveHealthDuration(c.HealthLabel(), result.Duration.Seconds())

	status := 0.0
	if result.Status == StatusHealthy {
		status = 1.0
	}

	c.recorder.SetHealthStatus(c.HealthLabel(), status)

	return result
}
