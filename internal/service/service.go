// Package service runs a service on the address the cockpit gives it, until the process gets
// SIGINT or SIGTERM.
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
// peers' addresses.
type Routes func(getenv func(string) string) (http.Handler, error)

// Run builds the service's handler with routes and serves it on the address in BAKHYARD_LISTEN
// until the process gets SIGINT or SIGTERM. Then it shuts the server down gracefully, as
// server.Run does, and returns nil if every request in flight finished.
func Run(getenv func(string) string, routes Routes) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	addr, err := env.ListenAddress(getenv)
	if err != nil {
		return err
	}

	handler, err := routes(getenv)
	if err != nil {
		return err
	}

	return server.Run(ctx, addr, handler)
}
