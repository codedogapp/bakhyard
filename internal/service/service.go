// Package service runs a service on the address in its environment, with its logs in the format
// that its environment names, until the process gets SIGINT or SIGTERM.
package service

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/codedogapp/bakhyard/internal/env"
	"github.com/codedogapp/bakhyard/internal/server"
)

// Routes builds a service's HTTP handler. getenv reads the service's environment, such as its
// peers' addresses. ctx ends when the service is told to stop or Run returns, so work that Routes
// starts in the background, such as polling, lasts as long as the service.
type Routes func(ctx context.Context, getenv func(string) string) (http.Handler, error)

// Run sets slog's default logger to the format in BAKHYARD_LOG_FORMAT first, so that even the
// error that stops the service is in it. Then it builds the service's handler with routes and
// serves it on the address in BAKHYARD_LISTEN until the process gets SIGINT or SIGTERM. Then it
// shuts the server down gracefully, as server.Run does, and returns nil if every request in
// flight finished.
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

	handler, err := routes(ctx, getenv)
	if err != nil {
		return err
	}

	return server.Run(ctx, addr, handler)
}
