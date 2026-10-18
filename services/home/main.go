package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/codedogapp/bakhyard/services/home/internal/api"
	"github.com/codedogapp/bakhyard/services/home/internal/health"
)

const (
	// Loopback only; 8080 is the gateway's.
	listenAddr        = "127.0.0.1:8081"
	readHeaderTimeout = 5 * time.Second
)

func main() {
	router := chi.NewRouter()
	// Answer HEAD wherever GET is served.
	router.Use(middleware.GetHead)
	health.Register(router)
	api.Register(router, time.Now)

	server := &http.Server{
		Addr:              listenAddr,
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	err := server.ListenAndServe()
	slog.Error("serve", "addr", listenAddr, "err", err)
	os.Exit(1)
}
