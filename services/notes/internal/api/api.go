// Package api serves the vault's file list as JSON over HTTP.
package api

import (
	"encoding/json/v2"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// list is the body of GET /notes: the paths of the vault's files.
type list struct {
	Files []string `json:"files"`
}

// Register serves GET /notes on router with files, the paths of the vault's files; other methods
// get 405.
func Register(router chi.Router, files []string) {
	body := list{Files: files}

	router.Get("/notes", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// The vault lists only valid UTF-8 names, so the list always encodes: an error means the
		// client has gone.
		_ = json.MarshalWrite(w, body)
	})
}
