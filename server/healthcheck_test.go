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
		version    string
		result     string
		statusCode int
	}

	tests := map[string]testCase{
		"healthy no details": {
			checker:    &mock.HealthCheck{},
			version:    "",
			result:     "{\"status\":\"healthy\",\"services\":{\"test\":{\"status\":\"healthy\"}}}\n",
			statusCode: http.StatusOK,
		},
		"healthy with details": {
			checker: &mock.HealthCheck{
				Result: "extra info",
			},
			version:    "",
			result:     "{\"status\":\"healthy\",\"services\":{\"test\":{\"status\":\"healthy\",\"details\":\"extra info\"}}}\n",
			statusCode: http.StatusOK,
		},
		"unhealthy no marshal": {
			checker: &mock.HealthCheck{
				Err: errors.New("some random error"),
			},
			version:    "",
			result:     "{\"status\":\"unhealthy\",\"services\":{\"test\":{\"status\":\"unhealthy\",\"details\":\"some random error\"}}}\n",
			statusCode: http.StatusInternalServerError,
		},
		"unhealthy with marshal": {
			checker: &mock.HealthCheck{
				Err: &mock.MarshaledError{Data: "special error"},
			},
			version:    "",
			result:     "{\"status\":\"unhealthy\",\"services\":{\"test\":{\"status\":\"unhealthy\",\"details\":{\"whoops\":\"special error\"}}}}\n",
			statusCode: http.StatusInternalServerError,
		},
		"with version": {
			checker:    &mock.HealthCheck{},
			version:    "v1.2.3.test",
			result:     "{\"status\":\"healthy\",\"version\":\"v1.2.3.test\",\"services\":{\"test\":{\"status\":\"healthy\"}}}\n",
			statusCode: http.StatusOK,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			testServer := httptest.NewServer(
				server.New(
					zerolog.Nop(), mock.NewNoOp(),
					server.SetVersion(test.version),
					server.AddHealthCheck("test", test.checker),
				),
			)

			request, _ := http.NewRequestWithContext(
				context.Background(),
				http.MethodGet,
				fmt.Sprintf("%s/health", testServer.URL),
				nil,
			)
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
