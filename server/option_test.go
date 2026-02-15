package server_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/b-sea/go-server/server"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestSetVersion(t *testing.T) {
	testServer := server.New(zerolog.Nop(), &server.NoOpRecorder{})
	server.SetVersion("")(testServer)
	assert.Equal(t, "", testServer.Version())

	server.SetVersion("special-test-version")(testServer)
	assert.Equal(t, "special-test-version", testServer.Version())
}

func TestSetPort(t *testing.T) {
	testServer := server.New(zerolog.Nop(), &server.NoOpRecorder{})
	server.SetPort(5000)(testServer)
	assert.Equal(t, ":5000", testServer.Addr())

	server.SetPort(4567)(testServer)
	assert.Equal(t, ":4567", testServer.Addr())
}

func TestSetReadTimeout(t *testing.T) {
	testServer := server.New(zerolog.Nop(), &server.NoOpRecorder{})
	server.SetReadTimeout(time.Hour)(testServer)
	assert.Equal(t, time.Hour, testServer.ReadTimeout())

	testServer = server.New(zerolog.Nop(), &server.NoOpRecorder{})
	server.SetReadTimeout(5 * time.Second)(testServer)
	assert.Equal(t, 5*time.Second, testServer.ReadTimeout())
}

func TestSetWriteTimeout(t *testing.T) {
	testServer := server.New(zerolog.Nop(), &server.NoOpRecorder{})
	server.SetWriteTimeout(time.Hour)(testServer)
	assert.Equal(t, time.Hour, testServer.WriteTimeout())

	testServer = server.New(zerolog.Nop(), &server.NoOpRecorder{})
	server.SetWriteTimeout(5 * time.Second)(testServer)
	assert.Equal(t, 5*time.Second, testServer.WriteTimeout())
}

func TestWithCustomCorrelationID(t *testing.T) {
	var buffer bytes.Buffer
	testServer := httptest.NewServer(
		server.New(
			zerolog.New(&buffer),
			&server.NoOpRecorder{},
			server.WithCustomCorrelationID(func() string { return "123-special-id-456" }),
		),
	)

	request, _ := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		fmt.Sprintf("%s/ping", testServer.URL),
		nil,
	)

	request.Close = true

	response, err := http.DefaultClient.Do(request)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, "123-special-id-456", response.Header.Get("Correlation-ID"))

	testServer.Close()
}

func TestReadCorrelationHeader(t *testing.T) {
	var buffer bytes.Buffer
	testServer := httptest.NewServer(server.New(zerolog.New(&buffer), &server.NoOpRecorder{}, server.ReadCorrelationHeader()))

	request, _ := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		fmt.Sprintf("%s/ping", testServer.URL),
		nil,
	)

	request.Header.Set("Correlation-ID", "i-come-from-a-header-123")

	request.Close = true

	response, err := http.DefaultClient.Do(request)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, "i-come-from-a-header-123", response.Header.Get("Correlation-ID"))

	testServer.Close()
}
