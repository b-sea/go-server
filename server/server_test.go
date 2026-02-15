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
	testServer := server.New(zerolog.Nop(), &server.NoOpRecorder{}, server.SetPort(findOpenPort(t)))

	timer := time.NewTimer(500 * time.Millisecond)

	go func() {
		assert.NoError(t, testServer.Start())
	}()

	<-timer.C

	assert.NoError(t, testServer.Stop())
}

func TestServerMetrics(t *testing.T) {
	testServer := httptest.NewServer(server.New(zerolog.Nop(), &server.NoOpRecorder{}))

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
	testServer := httptest.NewServer(server.New(zerolog.Nop(), &server.NoOpRecorder{}))

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
	testServer := httptest.NewServer(server.New(zerolog.Nop(), &server.NoOpRecorder{}, server.SetVersion("test-123")))

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
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	var buffer bytes.Buffer

	svr := server.New(
		zerolog.New(&buffer).Level(zerolog.ErrorLevel),
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
	assert.Equal(
		t,
		"{\"level\":\"error\",\"correlation_id\":\"123-special-id-456\",\"stack\":["+
			"{\"func\":\"(*Server).addDefaultHandlers.(*Server).telemetryMiddleware.func5.1.1\",\"line\":\"65\",\"source\":\"telemetry.go\"},"+
			"{\"func\":\"gopanic\",\"line\":\"783\",\"source\":\"panic.go\"},"+
			"{\"func\":\"TestPanickedHandler.TestPanickedHandler.func2.func4\",\"line\":\"143\",\"source\":\"server_test.go\"},"+
			"{\"func\":\"HandlerFunc.ServeHTTP\",\"line\":\"2322\",\"source\":\"server.go\"},"+
			"{\"func\":\"(*Server).addDefaultHandlers.(*Server).telemetryMiddleware.func5.1\",\"line\":\"99\",\"source\":\"telemetry.go\"},"+
			"{\"func\":\"HandlerFunc.ServeHTTP\",\"line\":\"2322\",\"source\":\"server.go\"},"+
			"{\"func\":\"(*Router).ServeHTTP\",\"line\":\"212\",\"source\":\"mux.go\"},"+
			"{\"func\":\"(*Server).ServeHTTP\",\"line\":\"103\",\"source\":\"server.go\"},"+
			"{\"func\":\"serverHandler.ServeHTTP\",\"line\":\"3340\",\"source\":\"server.go\"},"+
			"{\"func\":\"(*conn).serve\",\"line\":\"2109\",\"source\":\"server.go\"},"+
			"{\"func\":\"goexit\",\"line\":\"1693\",\"source\":\"asm_amd64.s\"}"+
			"],\"error\":\"panic: uh oh!\"}\n",
		buffer.String(),
	)

	testServer.Close()
}
