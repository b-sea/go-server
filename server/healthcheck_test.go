package server_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/b-sea/go-server/mock"
	"github.com/b-sea/go-server/server"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestServerHealth(t *testing.T) {
	type testCase struct {
		checker    server.HealthChecker
		verbose    bool
		version    string
		result     string
		statusCode int
	}

	tests := map[string]testCase{
		"healthy no dependencies": {
			checker:    nil,
			verbose:    false,
			version:    "",
			result:     "",
			statusCode: http.StatusOK,
		},
		"verbose healthy no dependencies": {
			checker:    nil,
			verbose:    true,
			version:    "",
			result:     "{\"status\":\"healthy\",\"uptime\":\"0s\"}\n",
			statusCode: http.StatusOK,
		},
		"healthy with dependencies": {
			checker:    &mock.HealthCheck{},
			verbose:    false,
			version:    "",
			result:     "",
			statusCode: http.StatusOK,
		},
		"verbose healthy with dependencies": {
			checker:    &mock.HealthCheck{},
			verbose:    true,
			version:    "",
			result:     "{\"status\":\"healthy\",\"uptime\":\"0s\",\"dependencies\":{\"test\":\"healthy\"}}\n",
			statusCode: http.StatusOK,
		},
		"unhealthy": {
			checker:    &mock.HealthCheck{Err: errors.New("something bad")},
			verbose:    false,
			version:    "",
			result:     "",
			statusCode: http.StatusInternalServerError,
		},
		"verbose unhealthy": {
			checker:    &mock.HealthCheck{Err: errors.New("something bad")},
			verbose:    true,
			version:    "",
			result:     "{\"status\":\"unhealthy\",\"uptime\":\"0s\",\"dependencies\":{\"test\":\"something bad\"}}\n",
			statusCode: http.StatusInternalServerError,
		},
		"with version": {
			checker:    nil,
			verbose:    true,
			version:    "v1.2.3.test",
			result:     "{\"status\":\"healthy\",\"version\":\"v1.2.3.test\",\"uptime\":\"0s\"}\n",
			statusCode: http.StatusOK,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			options := []server.Option{
				server.SetVersion(test.version),
			}
			if test.checker != nil {
				options = append(options, server.AddHealthCheck("test", test.checker))
			}

			testServer := httptest.NewServer(
				server.New(
					zerolog.Nop(), mock.NewNoOp(),
					options...,
				),
			)

			endpoint := fmt.Sprintf("%s/health", testServer.URL)
			if test.verbose {
				endpoint += "?verbose"
			}

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
