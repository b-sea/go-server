package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// Option is a creation option for a Server.
type Option func(ctx context.Context, server *Server)

// SetVersion sets the server version.
func SetVersion(version string) Option {
	return func(_ context.Context, server *Server) {
		if version == "" {
			return
		}

		server.version = version
	}
}

// SetPort overrides the port used by the Server.
func SetPort(port int) Option {
	return func(_ context.Context, server *Server) {
		if port == defaultPort {
			return
		}

		server.http.Addr = fmt.Sprintf(":%d", port)
	}
}

// SetReadTimeout overrides the HTTP read and read header timeouts for the Server.
func SetReadTimeout(duration time.Duration) Option {
	return func(_ context.Context, server *Server) {
		if duration == defaultTimeout {
			return
		}

		server.http.ReadTimeout = duration
		server.http.ReadHeaderTimeout = duration
	}
}

// SetWriteTimeout overrides the HTTP write timeout for the Server.
func SetWriteTimeout(duration time.Duration) Option {
	return func(_ context.Context, server *Server) {
		if duration == defaultTimeout {
			return
		}

		server.http.WriteTimeout = duration
	}
}

// ReadCorrelationHeader will allow the service to read a correlation ID from a request header.
func ReadCorrelationHeader() Option {
	return func(_ context.Context, server *Server) {
		server.readCorrelationHeader = true
	}
}

// WithCustomCorrelationID defines a custom Correlation ID generator.
func WithCustomCorrelationID(fn func() string) Option {
	return func(_ context.Context, server *Server) {
		server.newCorrelationID = fn
	}
}

// AddHealthDependency adds a sub system to include during server healthchecks.
func AddHealthDependency(name string, checker HealthChecker) Option {
	return func(ctx context.Context, server *Server) {
		zerolog.Ctx(ctx).Debug().Str("name", name).Msg("register health dependency")

		server.router.Handle(
			healthEndpoint+"/"+name,
			server.dependencyHealthCheckHandler(name),
		).Methods(http.MethodGet)

		server.healthDependencies[name] = checker
	}
}
