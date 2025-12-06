package server

import (
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/gorilla/mux"
)

// Option is a creation option for a Server.
type Option func(server *Server)

// SetVersion sets the server version.
func SetVersion(version string) Option {
	return func(server *Server) {
		if version == "" {
			return
		}

		server.log.Debug().Str("version", version).Msg("set version")
		server.version = version
	}
}

// SetPort overrides the port used by the Server.
func SetPort(port int) Option {
	return func(server *Server) {
		if port == defaultPort {
			return
		}

		server.log.Debug().Int("port", port).Msgf("override server port")
		server.http.Addr = fmt.Sprintf(":%d", port)
	}
}

// SetReadTimeout overrides the HTTP read and read header timeouts for the Server.
func SetReadTimeout(duration time.Duration) Option {
	return func(server *Server) {
		if duration <= 0 || duration == defaultTimeout {
			return
		}

		server.log.Debug().Dur("timeout_ms", duration).Msg("override server read timeout")
		server.http.ReadTimeout = duration
		server.http.ReadHeaderTimeout = duration
	}
}

// SetWriteTimeout overrides the HTTP write timeout for the Server.
func SetWriteTimeout(duration time.Duration) Option {
	return func(server *Server) {
		if duration <= 0 || duration == defaultTimeout {
			return
		}

		server.log.Debug().Dur("timeout_ms", duration).Msg("override server write timeout")
		server.http.WriteTimeout = duration
	}
}

// ReadCorrelationHeader will allow the service to read a correlation ID from a request header.
func ReadCorrelationHeader() Option {
	return func(server *Server) {
		server.readCorrelationHeader = true
	}
}

// WithCustomCorrelationID defines a custom Correlation ID generator.
func WithCustomCorrelationID(fn func() string) Option {
	return func(server *Server) {
		server.newCorrelationID = fn
	}
}

// AddHealthDependency adds a sub system to include during server healthchecks.
func AddHealthDependency(name string, checker HealthChecker) Option {
	return func(server *Server) {
		AddHandler(
			"/health/"+name,
			server.dependencyHealthCheckHandler(name),
			http.MethodGet,
		)(server)

		server.healthDependencies[name] = checker
	}
}

// AddHandler adds an HTTP hander to the Server.
func AddHandler(path string, handler http.Handler, methods ...string) Option {
	return func(server *Server) {
		handle := server.router.Handle(path, handler)

		if len(methods) == 0 {
			server.log.Debug().Str("path", path).Msg("route registered")

			return
		}

		slices.Sort(methods)

		for i := range methods {
			server.log.Debug().Str("method", methods[i]).Str("path", path).Msg("route registered")
		}

		handle.Methods(methods...)
	}
}

// AddMiddleware adds middleware to the Server.
func AddMiddleware(middleware mux.MiddlewareFunc) Option {
	return func(server *Server) {
		server.router.Use(middleware)
	}
}
