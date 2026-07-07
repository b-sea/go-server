package handler

import (
	"encoding/json"
	"net/http"

	"github.com/b-sea/go-server/health"
	"github.com/b-sea/go-server/server/oapi"
)

func Health(service *health.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")

		healthCheck := service.CheckAll(r.Context())

		response := oapi.HealthResponse{
			Status:       oapi.HealthStatus(healthCheck.Status),
			Dependencies: make(map[string]oapi.CheckResponse),
		}

		for name, check := range healthCheck.Dependencies {
			response.Dependencies[name] = oapi.CheckResponse{
				Status:     oapi.HealthStatus(check.Status),
				Message:    check.Message,
				DurationMS: int(check.Duration.Milliseconds()),
			}
		}

		if healthCheck.Status != health.StatusHealthy {
			w.WriteHeader(http.StatusInternalServerError)
		}

		_ = json.NewEncoder(w).Encode(&response)
	}
}
