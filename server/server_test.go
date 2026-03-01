package server_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/b-sea/go-server/server"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
	"github.com/stretchr/testify/assert"
)

func findOpenPort(t *testing.T) int {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	assert.NoError(t, err)

	listener, err := net.ListenTCP("tcp", addr)
	assert.NoError(t, err)

	port := listener.Addr().(*net.TCPAddr).Port
	assert.NoError(t, listener.Close())

	return port
}

func TestServerStartStop(t *testing.T) {
	testServer := server.New(context.Background(), &server.NoOpRecorder{}, server.WithPort(findOpenPort(t)))

	timer := time.NewTimer(500 * time.Millisecond)

	go func() {
		assert.NoError(t, testServer.Start(context.Background()))
	}()

	<-timer.C

	assert.NoError(t, testServer.Stop(context.Background()))
}

func TestServerMetrics(t *testing.T) {
	testServer := httptest.NewServer(server.New(context.Background(), &server.NoOpRecorder{}))

	request, _ := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		fmt.Sprintf("%s/metrics", testServer.URL),
		nil,
	)

	request.Close = true

	response, err := http.DefaultClient.Do(request)
	assert.NoError(t, err)

	body, err := io.ReadAll(response.Body)
	assert.NoError(t, err)

	assert.NoError(t, response.Body.Close())

	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, "", response.Header.Get("Content-Type"))
	assert.Equal(t, ``, string(body))

	testServer.Close()
}

func TestServerPing(t *testing.T) {
	testServer := httptest.NewServer(server.New(context.Background(), &server.NoOpRecorder{}))

	request, _ := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		fmt.Sprintf("%s/ping", testServer.URL),
		nil,
	)

	request.Close = true

	response, err := http.DefaultClient.Do(request)
	assert.NoError(t, err)

	body, err := io.ReadAll(response.Body)
	assert.NoError(t, err)

	assert.NoError(t, response.Body.Close())

	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, `text/plain; charset=utf-8`, response.Header.Get("Content-Type"))
	assert.Equal(t, `pong`, string(body))

	testServer.Close()
}

func TestServerVersion(t *testing.T) {
	testServer := httptest.NewServer(server.New(context.Background(), &server.NoOpRecorder{}, server.WithVersion("test-123")))

	request, _ := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		fmt.Sprintf("%s/version", testServer.URL),
		nil,
	)

	request.Close = true

	response, err := http.DefaultClient.Do(request)
	assert.NoError(t, err)

	body, err := io.ReadAll(response.Body)
	assert.NoError(t, err)

	assert.NoError(t, response.Body.Close())

	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, `text/plain; charset=utf-8`, response.Header.Get("Content-Type"))
	assert.Equal(t, `test-123`, string(body))

	testServer.Close()
}

func TestPanickedHandler(t *testing.T) {
	var buffer bytes.Buffer

	log := zerolog.New(&buffer).Level(zerolog.ErrorLevel)
	zerolog.DefaultContextLogger = &log
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	svr := server.New(
		context.Background(),
		&server.NoOpRecorder{},
		server.WithCustomCorrelationID(func() string { return "123-special-id-456" }),
	)

	svr.Router().Handle(
		"/test",
		func() http.HandlerFunc {
			return func(writer http.ResponseWriter, _ *http.Request) {
				panic("uh oh!")
			}
		}(),
	).Methods(http.MethodGet)

	testServer := httptest.NewServer(svr)

	request, _ := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		fmt.Sprintf("%s/test", testServer.URL),
		nil,
	)

	request.Close = true

	response, err := http.DefaultClient.Do(request)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
	assert.Contains(t, buffer.String(), "{\"level\":\"error\",\"correlation_id\":\"123-special-id-456\",\"stack\":[")

	testServer.Close()
}
