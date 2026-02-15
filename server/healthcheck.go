package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

const (
	healthyStatus   = "healthy"
	unhealthyStatus = "unhealthy"

	verboseParam = "verbose"
)

// HealthChecker defines functions required to run health checks.
type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}

type serviceHealth struct {
	name string
	err  error
}

func (s *Server) checkService(ctx context.Context, name string, checker HealthChecker, out chan<- serviceHealth) {
	out <- serviceHealth{
		name: name,
		err:  checker.HealthCheck(ctx),
	}
}

func (s *Server) healthCheckHandler() http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Add("Content-Type", "application/json")

		result := struct {
			Status       string         `json:"status"`
			Uptime       time.Duration  `json:"uptime"`
			Dependencies map[string]any `json:"dependencies,omitempty"`
		}{
			Status:       healthyStatus,
			Uptime:       s.Uptime(),
			Dependencies: make(map[string]any, 0),
		}

		serviceChan := make(chan serviceHealth)

		for name, checker := range s.healthDependencies {
			go s.checkService(request.Context(), name, checker, serviceChan)
		}

		for range s.healthDependencies {
			health := <-serviceChan

			result.Dependencies[health.name] = healthyStatus

			if health.err == nil {
				continue
			}

			result.Dependencies[health.name] = health.err
			if data, err := json.Marshal(health.err); err != nil || string(data) == "{}" {
				result.Dependencies[health.name] = health.err.Error()
			}

			// This extra check stops a "superfluous call to response.WriteHeader"
			if result.Status == healthyStatus {
				result.Status = unhealthyStatus

				writer.WriteHeader(http.StatusInternalServerError)
			}
		}

		zerolog.Ctx(request.Context()).Info().Interface("health", result).Msg("health check")

		if !request.URL.Query().Has(verboseParam) {
			return
		}

		_ = json.NewEncoder(writer).Encode(&result)
	})
}

func (s *Server) dependencyHealthCheckHandler(name string) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		checker, ok := s.healthDependencies[name]
		if !ok {
			http.Error(writer, "404 page not found", http.StatusNotFound)

			return
		}

		writer.Header().Add("Content-Type", "application/json")

		result := map[string]any{name: healthyStatus}

		if err := checker.HealthCheck(request.Context()); err != nil {
			writer.WriteHeader(http.StatusInternalServerError)

			result[name] = err
			if data, jsonErr := json.Marshal(err); jsonErr != nil || string(data) == "{}" {
				result[name] = err.Error()
			}
		}

		zerolog.Ctx(request.Context()).Info().Interface("health", result).Msg("health check")

		if !request.URL.Query().Has(verboseParam) {
			return
		}

		_ = json.NewEncoder(writer).Encode(result[name])
	})
}
