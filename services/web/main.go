package main

import (
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/codedogapp/bakhyard/services/web/internal/spa"
)

const (
	// Loopback only; 8080 is the gateway's and 8081 home's.
	listenAddr        = "127.0.0.1:8082"
	readHeaderTimeout = 5 * time.Second
	// Vite builds the app into dist/app; dist/.gitkeep keeps the embed valid before that.
	appDir = "dist/app"
)

//go:embed all:dist
var dist embed.FS

func main() {
	app, err := fs.Sub(dist, appDir)
	if err != nil {
		slog.Error("open the embedded app", "dir", appDir, "err", err)
		os.Exit(1)
	}

	router := chi.NewRouter()
	// Answer HEAD wherever GET is served.
	router.Use(middleware.GetHead)
	spa.Register(router, app)

	server := &http.Server{
		Addr:              listenAddr,
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	err = server.ListenAndServe()
	slog.Error("serve", "addr", listenAddr, "err", err)
	os.Exit(1)
}
