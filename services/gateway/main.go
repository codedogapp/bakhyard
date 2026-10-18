package main

import (
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/codedogapp/bakhyard/services/gateway/internal/proxy"
)

const (
	// Loopback only; tailscale serve will publish it.
	listenAddr = "127.0.0.1:8080"
	// home's listen address, until the services get their peers from the environment.
	homeAddr          = "127.0.0.1:8081"
	readHeaderTimeout = 5 * time.Second
)

func main() {
	logger := slog.Default()

	mux := http.NewServeMux()
	proxy.Register(mux, "home", &url.URL{Scheme: "http", Host: homeAddr}, logger)

	server := &http.Server{
		Addr:              listenAddr,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	err := server.ListenAndServe()
	logger.Error("serve", "addr", listenAddr, "err", err)
	os.Exit(1)
}
