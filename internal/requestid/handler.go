package requestid

import (
	"context"
	"log/slog"
)

// Handler is a slog handler that adds request_id, the ID of the request that a record's context
// belongs to, to each record that it passes on. Inside a group, request_id is in the group too.
type Handler struct {
	inner slog.Handler
}

// NewHandler returns a Handler that passes the records on to inner.
func NewHandler(inner slog.Handler) Handler {
	return Handler{inner: inner}
}

// Enabled reports whether the inner handler handles records at level.
func (h Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle adds request_id to record when ctx belongs to a request and passes it on.
func (h Handler) Handle(ctx context.Context, record slog.Record) error {
	id, ok := FromContext(ctx)
	if ok {
		record.AddAttrs(slog.String("request_id", id))
	}

	return h.inner.Handle(ctx, record)
}

// WithAttrs returns a Handler whose inner handler has attrs.
func (h Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return Handler{inner: h.inner.WithAttrs(attrs)}
}

// WithGroup returns a Handler whose inner handler has the group name.
func (h Handler) WithGroup(name string) slog.Handler {
	return Handler{inner: h.inner.WithGroup(name)}
}
