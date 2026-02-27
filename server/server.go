// Package server implements the API web server.
package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
)

const (
	defaultPort    = 5000
	defaultTimeout = 5 * time.Second

	healthEndpoint  = "/health"
	metricsEndpoint = "/metrics"
	pingEndpoint    = "/ping"
	versionEndpoint = "/version"
)

// Server is a supply-run API web server.
type Server struct {
	mu                    sync.Mutex
	readCorrelationHeader bool
	newCorrelationID      func() string
	router                *mux.Router
	http                  *http.Server
	healthDependencies    map[string]HealthChecker
	startedAt             time.Time
	version               string
}

// New creates a new Server.
func New(ctx context.Context, recorder Recorder, options ...Option) *Server {
	server := &Server{
		readCorrelationHeader: false,
		newCorrelationID:      uuid.NewString,
		router:                mux.NewRouter(),
		http: &http.Server{
			Addr:              fmt.Sprintf(":%d", defaultPort),
			ReadTimeout:       defaultTimeout,
			ReadHeaderTimeout: defaultTimeout,
			WriteTimeout:      defaultTimeout,
		},
		healthDependencies: make(map[string]HealthChecker),
		startedAt:          time.Time{},
		version:            "",
	}

	server.addDefaultHandlers(ctx, recorder)

	for _, option := range options {
		option(ctx, server)
	}

	return server
}

// Version returns the server version.
func (s *Server) Version() string {
	return s.version
}

// Uptime is the amount of time the server has beeen running.
func (s *Server) Uptime() time.Duration {
	if s.startedAt.IsZero() {
		return 0
	}

	return time.Since(s.startedAt)
}

// Addr returns the server address.
func (s *Server) Addr() string {
	return s.http.Addr
}

// ReadTimeout returns the server read and read header timeout.
func (s *Server) ReadTimeout() time.Duration {
	return s.http.ReadTimeout
}

// WriteTimeout returns the server write timeout.
func (s *Server) WriteTimeout() time.Duration {
	return s.http.WriteTimeout
}

// Router returns the server router.
func (s *Server) Router() *mux.Router {
	return s.router
}

func (s *Server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	s.prepareHTTPServe()
	s.http.Handler.ServeHTTP(writer, request)
}

// Start the Server.
func (s *Server) Start(ctx context.Context) error {
	zerolog.Ctx(ctx).Info().Str("addr", s.http.Addr).Msg("starting server")
	s.prepareHTTPServe()

	s.mu.Lock()
	s.startedAt = time.Now()
	s.mu.Unlock()

	if err := s.http.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err //nolint: wrapcheck
	}

	return nil
}

// Stop the Server.
func (s *Server) Stop(ctx context.Context) error {
	zerolog.Ctx(ctx).Info().Str("addr", s.http.Addr).Msg("stopping server")

	s.mu.Lock()
	s.startedAt = time.Time{}
	s.mu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	return s.http.Shutdown(ctx) //nolint: wrapcheck
}

func (s *Server) prepareHTTPServe() {
	if s.router.NotFoundHandler == nil {
		// Re-define the default NotFound handler so it passes through middleware correctly.
		s.router.NotFoundHandler = s.router.NewRoute().HandlerFunc(http.NotFound).GetHandler()
	}

	s.http.Handler = s.router
}

func (s *Server) addDefaultHandlers(ctx context.Context, recorder Recorder) {
	zerolog.Ctx(ctx).Debug().Str("middleware", "telemetry").Msg("register")
	s.router.Use(s.telemetryMiddleware(recorder))

	zerolog.Ctx(ctx).Debug().Str("method", http.MethodGet).Str("path", pingEndpoint).Msg("register")
	s.router.Handle(
		pingEndpoint,
		func() http.HandlerFunc {
			return func(writer http.ResponseWriter, _ *http.Request) {
				_, _ = writer.Write([]byte(`pong`))
			}
		}(),
	).Methods(http.MethodGet)

	zerolog.Ctx(ctx).Debug().Str("method", http.MethodGet).Str("path", versionEndpoint).Msg("register")
	s.router.Handle(
		versionEndpoint,
		func() http.HandlerFunc {
			return func(writer http.ResponseWriter, _ *http.Request) {
				version := "unversioned"
				if s.version != "" {
					version = s.version
				}

				_, _ = writer.Write([]byte(version))
			}
		}(),
	).Methods(http.MethodGet)

	zerolog.Ctx(ctx).Debug().Str("method", http.MethodGet).Str("path", metricsEndpoint).Msg("register")
	s.router.Handle(
		pingEndpoint,
		func() http.HandlerFunc {
			return func(writer http.ResponseWriter, _ *http.Request) {
				_, _ = writer.Write([]byte(`pong`))
			}
		}(),
	).Methods(http.MethodGet)

	s.router.Handle(
		metricsEndpoint,
		func() http.HandlerFunc {
			return func(writer http.ResponseWriter, request *http.Request) {
				recorder.Handler().ServeHTTP(writer, request)
			}
		}(),
	).Methods(http.MethodGet)

	zerolog.Ctx(ctx).Debug().Str("method", http.MethodGet).Str("path", healthEndpoint).Msg("register")
	s.router.Handle(healthEndpoint, s.healthCheckHandler()).Methods(http.MethodGet)
}
