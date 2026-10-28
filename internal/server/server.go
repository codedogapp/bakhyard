// Package server runs a service's HTTP server until its context ends, then shuts it down
// gracefully.
package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

const (
	readHeaderTimeout = 5 * time.Second
	// shutdownTimeout fits in process-compose's 10 s between SIGTERM and SIGKILL.
	shutdownTimeout = 8 * time.Second
)

// Run serves handler on the TCP address addr until ctx ends. Then it stops accepting
// connections and waits up to 8 s for the requests in flight to finish. It returns nil if they
// all did.
func Run(ctx context.Context, addr string, handler http.Handler) error {
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", addr)
	if err != nil {
		// net's error already says what failed: "listen tcp <addr>: …".
		return err
	}

	return serve(ctx, listener, handler, shutdownTimeout)
}

func serve(
	ctx context.Context,
	listener net.Listener,
	handler http.Handler,
	timeout time.Duration,
) error {
	// HTTP/1 serves the browsers and the gateway; HTTP/2 without TLS (h2c) serves the gRPC calls
	// between services.
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)

	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		Protocols:         protocols,
	}
	shutdown := make(chan error, 1)

	stop := context.AfterFunc(ctx, func() {
		deadline, cancel := context.WithTimeout(context.WithoutCancel(ctx), timeout)
		defer cancel()

		shutdown <- server.Shutdown(deadline)
	})
	defer stop()

	err := server.Serve(listener)
	if !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}

	err = <-shutdown
	if err != nil {
		return fmt.Errorf("shut down: %w", err)
	}

	return nil
}
