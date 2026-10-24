// Package api serves the vault's file list and its files over HTTP.
package api

import (
	"encoding/json/v2"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
)

// list is the body of GET /notes: the paths of the vault's files.
type list struct {
	Files []string `json:"files"`
}

// Register serves the vault on router: GET /notes lists files, the paths of the vault's files,
// and GET /notes/files/<path> returns one of them from root. Other methods get 405, and logger
// gets the errors that reading a file runs into.
func Register(router chi.Router, root *os.Root, files []string, logger *slog.Logger) {
	body := list{Files: files}

	router.Get("/notes", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// The vault lists only valid UTF-8 names, so the list always encodes: an error means the
		// client has gone.
		_ = json.MarshalWrite(w, body)
	})
	router.Get(filesPath+"*", serveFile(root, files, logger))
}
