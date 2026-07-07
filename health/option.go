package health

import "log/slog"

type ServiceOption func(h *Service)

func WithLogger(logger *slog.Logger) ServiceOption {
	return func(h *Service) {
		h.logger = logger
	}
}

func WithRecorder(recorder Recorder) ServiceOption {
	return func(h *Service) {
		h.recorder = recorder
	}
}
