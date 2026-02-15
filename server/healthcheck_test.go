package server_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/b-sea/go-server/server"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

var (
	_ server.HealthChecker = (*HealthCheck)(nil)
)

type JSONError struct {
	Inner any
}

func (e *JSONError) Error() string {
	return fmt.Sprintf("json error: %v", e.Inner)
}

func (e *JSONError) MarshalJSON() ([]byte, error) {
	return json.Marshal(
		struct {
			Details any `json:"details"`
		}{
			Details: e.Inner,
		},
	)
}

type HealthCheck struct {
	Err error
}

func (m *HealthCheck) HealthCheck(context.Context) error {
	return m.Err
}

func TestServerHealth(t *testing.T) {
	type testCase struct {
		url        string
		option     server.Option
		result     string
		statusCode int
	}

	tests := map[string]testCase{
		"healthy no dependencies": {
			url:        "/health",
			option:     nil,
			result:     "",
			statusCode: http.StatusOK,
		},
		"healthy verbose no dependencies": {
			url:        "/health?verbose",
			option:     nil,
			result:     "{\"status\":\"healthy\",\"uptime\":0}\n",
			statusCode: http.StatusOK,
		},
		"healthy with dependencies": {
			url:        "/health",
			option:     server.AddHealthDependency("sub-system", &HealthCheck{}),
			result:     "",
			statusCode: http.StatusOK,
		},
		"healthy verbose with dependencies": {
			url:        "/health?verbose",
			option:     server.AddHealthDependency("sub-system", &HealthCheck{}),
			result:     "{\"status\":\"healthy\",\"uptime\":0,\"dependencies\":{\"sub-system\":\"healthy\"}}\n",
			statusCode: http.StatusOK,
		},
		"unhealthy": {
			url:        "/health",
			option:     server.AddHealthDependency("sub-system", &HealthCheck{Err: errors.New("something bad")}),
			result:     "",
			statusCode: http.StatusInternalServerError,
		},
		"unhealthy verbose with dependencies": {
			url:        "/health?verbose",
			option:     server.AddHealthDependency("sub-system", &HealthCheck{Err: errors.New("something bad")}),
			result:     "{\"status\":\"unhealthy\",\"uptime\":0,\"dependencies\":{\"sub-system\":\"something bad\"}}\n",
			statusCode: http.StatusInternalServerError,
		},
		"unhealthy verbose with dependencies marshal": {
			url:        "/health?verbose",
			option:     server.AddHealthDependency("sub-system", &HealthCheck{Err: &JSONError{Inner: "extra details"}}),
			result:     "{\"status\":\"unhealthy\",\"uptime\":0,\"dependencies\":{\"sub-system\":{\"details\":\"extra details\"}}}\n",
			statusCode: http.StatusInternalServerError,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			options := []server.Option{}
			if test.option != nil {
				options = append(options, test.option)
			}

			testServer := httptest.NewServer(server.New(zerolog.Nop(), &server.NoOpRecorder{}, options...))

			endpoint := testServer.URL + test.url
			request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, endpoint, nil)
			request.Close = true

			response, err := http.DefaultClient.Do(request)
			assert.NoError(t, err)

			body, err := io.ReadAll(response.Body)
			assert.NoError(t, err)

			assert.NoError(t, response.Body.Close())

			assert.Equal(t, test.statusCode, response.StatusCode)
			assert.Equal(t, "application/json", response.Header.Get("Content-Type"))
			assert.Equal(t, test.result, string(body))

			testServer.Close()
		})
	}
}

func TestDependencyHealth(t *testing.T) {
	type testCase struct {
		url         string
		option      server.Option
		result      string
		statusCode  int
		contentType string
	}

	tests := map[string]testCase{
		"healthy": {
			url:         "/health/sub-system",
			option:      server.AddHealthDependency("sub-system", &HealthCheck{}),
			result:      "",
			statusCode:  http.StatusOK,
			contentType: "application/json",
		},
		"healthy verbose": {
			url:         "/health/sub-system?verbose",
			option:      server.AddHealthDependency("sub-system", &HealthCheck{}),
			result:      "\"healthy\"\n",
			statusCode:  http.StatusOK,
			contentType: "application/json",
		},
		"unhealthy": {
			url:         "/health/sub-system",
			option:      server.AddHealthDependency("sub-system", &HealthCheck{Err: errors.New("something bad")}),
			result:      "",
			statusCode:  http.StatusInternalServerError,
			contentType: "application/json",
		},
		"unhealthy verbose": {
			url:         "/health/sub-system?verbose",
			option:      server.AddHealthDependency("sub-system", &HealthCheck{Err: errors.New("something bad")}),
			result:      "\"something bad\"\n",
			statusCode:  http.StatusInternalServerError,
			contentType: "application/json",
		},
		"unhealthy verbose marshal": {
			url:         "/health/sub-system?verbose",
			option:      server.AddHealthDependency("sub-system", &HealthCheck{Err: &JSONError{Inner: "extra details"}}),
			result:      "{\"details\":\"extra details\"}\n",
			statusCode:  http.StatusInternalServerError,
			contentType: "application/json",
		},
		"not found": {
			url:         "/health/different",
			option:      server.AddHealthDependency("sub-system", &HealthCheck{}),
			result:      "404 page not found\n",
			statusCode:  http.StatusNotFound,
			contentType: "text/plain; charset=utf-8",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			testServer := httptest.NewServer(server.New(zerolog.Nop(), &server.NoOpRecorder{}, test.option))

			endpoint := testServer.URL + test.url
			request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, endpoint, nil)
			request.Close = true

			response, err := http.DefaultClient.Do(request)
			assert.NoError(t, err)

			body, err := io.ReadAll(response.Body)
			assert.NoError(t, err)

			assert.NoError(t, response.Body.Close())

			assert.Equal(t, test.statusCode, response.StatusCode)
			assert.Equal(t, test.contentType, response.Header.Get("Content-Type"))
			assert.Equal(t, test.result, string(body))

			testServer.Close()
		})
	}
}
