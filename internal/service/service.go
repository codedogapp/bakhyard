// Package service runs a service on the address in its environment, with its logs in the format
// that its environment names and an ID for each request, until the process gets SIGINT or
// SIGTERM.
package service

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/codedogapp/bakhyard/internal/env"
	"github.com/codedogapp/bakhyard/internal/requestid"
	"github.com/codedogapp/bakhyard/internal/server"
)

// Routes builds a service's HTTP handler. getenv reads the service's environment, such as its
// peers' addresses. ctx ends when the service is told to stop or Run returns, so work that Routes
// starts in the background, such as polling, lasts as long as the service.
type Routes func(ctx context.Context, getenv func(string) string) (http.Handler, error)

// Run sets slog's default logger to the format in BAKHYARD_LOG_FORMAT first, so that even the
// error that stops the service is in it. Then it builds the service's handler with routes, which
// gives each request an ID (package requestid), and serves it on the address in BAKHYARD_LISTEN
// until the process gets SIGINT or SIGTERM. Then it shuts the server down gracefully, as
// server.Run does, and returns nil if every request in flight finished.
func Run(getenv func(string) string, routes Routes) error {
	err := setLogs(getenv, os.Stderr)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	addr, err := env.ListenAddress(getenv)
	if err != nil {
		return err
	}

	handler, err := buildHandler(ctx, getenv, routes)
	if err != nil {
		return err
	}

	return server.Run(ctx, addr, handler)
}

// buildHandler builds the service's handler with routes, behind requestid's middleware, so that
// each request has an ID.
func buildHandler(
	ctx context.Context,
	getenv func(string) string,
	routes Routes,
) (http.Handler, error) {
	handler, err := routes(ctx, getenv)
	if err != nil {
		return nil, err
	}

	return requestid.Middleware(handler), nil
}
