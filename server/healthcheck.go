package server

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog"
)

const (
	healthyStatus   = "healthy"
	unhealthyStatus = "unhealthy"
)

// HealthChecker defines functions required to run health checks.
type HealthChecker interface {
	HealthCheck() (any, error)
}

type serviceHealth struct {
	name    string
	Status  string `json:"status"`
	Details any    `json:"details,omitempty"`
}

func (s *Server) runHealthCheck(name string, in <-chan HealthChecker, out chan<- serviceHealth) {
	checker := <-in

	details, err := checker.HealthCheck()
	health := serviceHealth{
		name:    name,
		Status:  healthyStatus,
		Details: details,
	}

	if err != nil {
		if data, jsonErr := json.Marshal(err); jsonErr != nil || string(data) == "{}" {
			health.Details = err.Error()
		} else {
			health.Details = err
		}

		health.Status = unhealthyStatus
	}

	out <- health
}

func (s *Server) healthCheckHandler(recorder Recorder) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Add("Content-Type", "application/json")

		result := struct {
			Status   string                   `json:"status"`
			Version  string                   `json:"version,omitempty"`
			Services map[string]serviceHealth `json:"services,omitempty"`
		}{
			Status:   healthyStatus,
			Version:  s.version,
			Services: make(map[string]serviceHealth, 0),
		}

		checkChan := make(chan HealthChecker)
		healthChan := make(chan serviceHealth)

		for name, checker := range s.healthChecks {
			go s.runHealthCheck(name, checkChan, healthChan)

			checkChan <- checker
		}

		for range s.healthChecks {
			health := <-healthChan
			result.Services[health.name] = health

			if result.Status == healthyStatus && health.Status == unhealthyStatus {
				result.Status = unhealthyStatus

				writer.WriteHeader(http.StatusInternalServerError)
			}

			recorder.ObserveHealth(health.name, health.Status == healthyStatus)
		}

		zerolog.Ctx(request.Context()).Info().Interface("health", result).Msg("healthcheck")
		recorder.ObserveHealth("server", result.Status == healthyStatus)

		_ = json.NewEncoder(writer).Encode(&result)
	})
}
