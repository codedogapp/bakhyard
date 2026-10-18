package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/codedogapp/bakhyard/services/home/internal/health"
)

const (
	// Loopback only; 8080 is the gateway's.
	listenAddr        = "127.0.0.1:8081"
	readHeaderTimeout = 5 * time.Second
)

func main() {
	mux := http.NewServeMux()
	health.Register(mux)

	server := &http.Server{
		Addr:              listenAddr,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	err := server.ListenAndServe()
	slog.Error("serve", "addr", listenAddr, "err", err)
	os.Exit(1)
}
