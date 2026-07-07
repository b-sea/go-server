package server

import (
	"context"
	"log/slog"
)

type ContextHandler struct {
	slog.Handler
}

func NewContextHandler(handler slog.Handler) *ContextHandler {
	return &ContextHandler{handler}
}

func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if correlationID, ok := ctx.Value(correlationKey).(string); ok {
		r.Add(slog.String("correlation_id", string(correlationID)))
	}

	return h.Handler.Handle(ctx, r) //nolint: wrapcheck
}

func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{h.Handler.WithAttrs(attrs)}
}

func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{h.Handler.WithGroup(name)}
}
