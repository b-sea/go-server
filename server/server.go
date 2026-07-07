package server

import (
	"log/slog"
	"net/http"

	"github.com/b-sea/go-server/handler"
	"github.com/b-sea/go-server/health"
	"github.com/b-sea/go-server/request"
	"github.com/b-sea/go-server/server/oapi"
)

type Recorder interface {
	health.Recorder
	request.Recorder

	Handler() http.Handler
}

var _ oapi.ServerInterface = (*Server)(nil)

type Server struct {
	version  string
	health   *health.Service
	recorder Recorder
	logger   *slog.Logger
}

func New(version string, health *health.Service, recorder Recorder, logger *slog.Logger) *Server {
	return &Server{
		version:  version,
		health:   health,
		recorder: recorder,
		logger:   logger,
	}
}

func (s *Server) Ping(writer http.ResponseWriter, request *http.Request) {
	handler.Ping().ServeHTTP(writer, request)
}

func (s *Server) Health(writer http.ResponseWriter, request *http.Request) {
	handler.Health(s.health).ServeHTTP(writer, request)
}

func (s *Server) Version(writer http.ResponseWriter, request *http.Request) {
	handler.Version(s.version).ServeHTTP(writer, request)
}

func (s *Server) Metrics(writer http.ResponseWriter, request *http.Request) {
	s.recorder.Handler().ServeHTTP(writer, request)
}
