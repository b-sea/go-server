package server_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/b-sea/go-server/metrics"
	"github.com/b-sea/go-server/server"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

var (
	_ server.HealthChecker = (*HealthCheck)(nil)
)

type HealthCheck struct {
	Err error
}

func (m *HealthCheck) HealthCheck() error {
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
		"unhealthy verbose": {
			url:        "/health?verbose",
			option:     server.AddHealthDependency("sub-system", &HealthCheck{Err: errors.New("something bad")}),
			result:     "{\"status\":\"unhealthy\",\"uptime\":0,\"dependencies\":{\"sub-system\":\"something bad\"}}\n",
			statusCode: http.StatusInternalServerError,
		},
		"with version": {
			url:        "/health?verbose",
			option:     server.SetVersion("v1.2.3.test"),
			result:     "{\"status\":\"healthy\",\"version\":\"v1.2.3.test\",\"uptime\":0}\n",
			statusCode: http.StatusOK,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			options := []server.Option{}
			if test.option != nil {
				options = append(options, test.option)
			}

			testServer := httptest.NewServer(server.New(zerolog.Nop(), &metrics.NoOp{}, options...))

			endpoint := testServer.URL + "/" + test.url
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
			testServer := httptest.NewServer(server.New(zerolog.Nop(), &metrics.NoOp{}, test.option))

			endpoint := testServer.URL + "/" + test.url
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
