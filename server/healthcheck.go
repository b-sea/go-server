package server

import (
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
	HealthCheck() error
}

type serviceHealth struct {
	name string
	err  error
}

func (s *Server) checkService(name string, in <-chan HealthChecker, out chan<- serviceHealth) {
	checker := <-in

	health := serviceHealth{
		name: name,
		err:  checker.HealthCheck(),
	}

	out <- health
}

func (s *Server) healthCheckHandler() http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Add("Content-Type", "application/json")

		result := struct {
			Status       string            `json:"status"`
			Version      string            `json:"version,omitempty"`
			Uptime       time.Duration     `json:"uptime"`
			Dependencies map[string]string `json:"dependencies,omitempty"`
		}{
			Status:       healthyStatus,
			Version:      s.version,
			Uptime:       s.Uptime(),
			Dependencies: make(map[string]string, 0),
		}

		checkChan := make(chan HealthChecker)
		serviceChan := make(chan serviceHealth)

		for name, checker := range s.healthDependencies {
			go s.checkService(name, checkChan, serviceChan)

			checkChan <- checker
		}

		for range s.healthDependencies {
			health := <-serviceChan

			result.Dependencies[health.name] = healthyStatus

			if health.err != nil {
				result.Dependencies[health.name] = health.err.Error()

				// This extra check stops a "superfluous call to response.WriteHeader"
				if result.Status == healthyStatus {
					result.Status = unhealthyStatus

					writer.WriteHeader(http.StatusInternalServerError)
				}
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

		result := map[string]string{name: healthyStatus}

		if err := checker.HealthCheck(); err != nil {
			writer.WriteHeader(http.StatusInternalServerError)

			result[name] = err.Error()
		}

		zerolog.Ctx(request.Context()).Info().Interface("health", result).Msg("health check")

		if !request.URL.Query().Has(verboseParam) {
			return
		}

		_ = json.NewEncoder(writer).Encode(result[name])
	})
}
