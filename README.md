# go-server

A standardized web server setup.

## Install

```bash
go get github.com/b-sea/go-server
```

## Basic Usage

```go
package main

import (
    "net/http"
    "os"
    "os/signal"
    "syscall"

    "github.com/b-sea/go-server/server"
    "github.com/rs/zerolog"
)

func main() {
    // Define additional endpoint(s) for the server.
    helloHandler := func() http.HandlerFunc {
        return func(writer http.ResponseWriter, _ *http.Request) {
            _, _ = writer.Write([]byte(`hello!`))
        }
    }

    svr := server.New(zerolog.Nop(), &server.NoOpRecorder{}, server.AddHandler("/hello", helloHandler(), http.MethodGet))

    channel := make(chan os.Signal, 1)
    signal.Notify(channel, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        if err := svr.Start(); err != nil {
            os.Exit(1)
        }
    }()

    <-channel

    if err := svr.Stop(); err != nil {
        os.Exit(1)
    }
}

```

## Utility Endpoints

The server comes with 4 standard utility endpoints to provide a life check, a health check, 
get the server version, and metrics.

### GET /ping

The `/ping` endpoint will return a `200` code and a body of `pong` if the service is alive.

### GET /version

The `/version` endpoint will return a `200` code and the server version if set. If not set, the version will 
be `unversioned`.

### GET /health

The `/health` endpoint reports the overall health of the server. By default, this endpoint will simply return a 
`200 OK` if healthy and `500 Internal Server Error` if unhealthy.

To see detailed information, `/health?verbose` can be used. The `version` field will only appear if a version has 
been provided to the server.

```json
{
    "status": "healthy",
    "uptime": 348698434,
}
```

#### Dependencies

The health endpoint can be expanded to include dependencies with the `AddHealthDependency` option.

The health of all dependencies will be automatically checked when the `/health` endpoint is called and will be used 
to determine the health of the server. If any dependencies are unhealthy, the server will consider itself 
unhealthy overall.

If `/health?verbose` is used, the dependency's health results will be displayed alongside the rest of the health data.

```json
{
    "status": "unhealthy",
    "uptime": 348698434,
    "dependencies":{
        "my-dependency": "error details",
        "another-dependency": "healthy"
    }
}
```

Individual dependencies can be checked with `GET /health/dependency-name`. These act similar to the main healthcheck. 
For detailed information, `/health/dependency-name?verbose` can be used.

### GET /metrics

The `/metrics` endpoint exposes system metrics for scraping.

### Recorded Metrics

There are two metrics recorded by the server:

* **ObserveRequestDuration** - tracks every request method, path, status code, and duration
* **ObserveResponseSize** - tracks every response method, path, status code, and byte size

### Default Recorders

The server package provides two basic metrics recorders for convenience:

* **No-Op** - the `NoOpRecorder` is a disabled recorder in which all records are ignored.
* **Prometheus** - the `PrometheusRecorder` implements metrics for Prometheus. It can be expanded as needed.
    ```go
    type Recorder struct {
        server.PrometheusRecorder

        MyCustomMetric *prometheus.GaugeVec
    }

    server.New(zerolog.Nop(), &Recorder{})
    ```

## Logging

The server handles logging with [zerolog](https://github.com/rs/zerolog).

Every request has a logger put in the request context along with a 
[correlation id](https://last9.io/blog/correlation-id-vs-trace-id/). This logger can be retrieved and used with

```go
func (s *Server) myHandler() http.Handler {
    return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
        log := zerolog.Ctx(request.Context())
        log.Info().Msg("used logger from context")
    })
}
```

Additionally, every request is logged with the following log fields:

* correlation_id
* user_agent
* method
* url
* status_code
* duration_ms
* response_byes

### Panics

If the web server encounters a panic, the stack trace will be logged out (as long as the logger is configured 
to display stacks) and handler will return a `500 Internal Server Error`
