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

	"github.com/b-sea/go-server/metrics"
	"github.com/b-sea/go-server/server"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestServerStartStop(t *testing.T) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	assert.NoError(t, err)

	listener, err := net.ListenTCP("tcp", addr)
	assert.NoError(t, err)

	port := listener.Addr().(*net.TCPAddr).Port
	assert.NoError(t, listener.Close())

	testServer := server.New(zerolog.Nop(), &metrics.NoOp{}, server.SetPort(port))

	timer := time.NewTimer(500 * time.Millisecond)

	go func() {
		assert.NoError(t, testServer.Start())
	}()

	<-timer.C

	assert.NoError(t, testServer.Stop())
}

func TestServerMetrics(t *testing.T) {
	testServer := httptest.NewServer(server.New(zerolog.Nop(), &metrics.NoOp{}))

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
	testServer := httptest.NewServer(server.New(zerolog.Nop(), &metrics.NoOp{}))

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

func TestPanickedHandler(t *testing.T) {
	var buffer bytes.Buffer
	testServer := httptest.NewServer(
		server.New(
			zerolog.New(&buffer).Level(zerolog.ErrorLevel),
			&metrics.NoOp{},
			server.WithCustomCorrelationID(func() string { return "123-special-id-456" }),
			server.AddHandler("/test",
				func() http.HandlerFunc {
					return func(writer http.ResponseWriter, _ *http.Request) {
						panic("uh oh!")
					}
				}(),
			),
		),
	)

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
	assert.Equal(
		t,
		"{\"level\":\"error\",\"correlation_id\":\"123-special-id-456\",\"error\":\"http: uh oh!\"}\n",
		buffer.String(),
	)

	testServer.Close()
}
