// Package server implements the API web server.
package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/xid"
	"github.com/rs/zerolog"
)

const (
	defaultPort    = 5000
	defaultTimeout = 5 * time.Second
)

// Server is a supply-run API web server.
type Server struct {
	newCorrelationID func() string
	router           *mux.Router
	http             *http.Server
	healthChecks     map[string]HealthChecker
	startedAt        time.Time
	log              zerolog.Logger
	version          string
}

// New creates a new Server.
func New(log zerolog.Logger, recorder Recorder, options ...Option) *Server {
	server := &Server{
		newCorrelationID: func() string { return xid.New().String() },
		router:           mux.NewRouter(),
		http: &http.Server{
			Addr:              fmt.Sprintf(":%d", defaultPort),
			ReadTimeout:       defaultTimeout,
			ReadHeaderTimeout: defaultTimeout,
			WriteTimeout:      defaultTimeout,
		},
		healthChecks: make(map[string]HealthChecker),
		startedAt:    time.Time{},
		log:          log,
		version:      "",
	}

	options = append(
		options,
		AddMiddleware(server.telemetryMiddleware(recorder)),
		AddHandler(
			"/ping",
			func() http.HandlerFunc {
				return func(writer http.ResponseWriter, _ *http.Request) {
					_, _ = writer.Write([]byte(`pong`))
				}
			}(),
			http.MethodGet,
		),
		AddHandler("/health", server.healthCheckHandler(recorder), http.MethodGet),
		AddHandler(
			"/metrics",
			func() http.HandlerFunc {
				return func(writer http.ResponseWriter, request *http.Request) {
					recorder.Handler().ServeHTTP(writer, request)
				}
			}(),
			http.MethodGet,
		),
	)

	for _, option := range options {
		option(server)
	}

	// Re-define the default NotFound handler so it passes through middleware correctly.
	server.router.NotFoundHandler = server.router.NewRoute().HandlerFunc(http.NotFound).GetHandler()
	server.http.Handler = server.router

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

func (s *Server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	s.http.Handler.ServeHTTP(writer, request)
}

// Start the Server.
func (s *Server) Start() error {
	s.log.Info().Str("addr", s.http.Addr).Msg("starting server")

	s.startedAt = time.Now()

	if err := s.http.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err //nolint: wrapcheck
	}

	return nil
}

// Stop the Server.
func (s *Server) Stop() error {
	s.log.Info().Str("addr", s.http.Addr).Msg("stopping server")

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	return s.http.Shutdown(ctx) //nolint: wrapcheck
}
