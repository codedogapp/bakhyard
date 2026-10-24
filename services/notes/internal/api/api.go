// Package api serves the vault's file list and its files over HTTP.
package api

import (
	"encoding/json/v2"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
)

// list is the body of GET /notes and POST /notes/sync: the paths of the vault's files.
type list struct {
	Files []string `json:"files"`
}

// Register serves the vault in root on router: GET /notes lists its files, starting with files,
// GET /notes/files/<path> returns one of them, and POST /notes/sync scans the vault again. Other
// methods get 405, and logger gets the errors that reading the vault runs into.
func Register(router chi.Router, root *os.Root, files []string, logger *slog.Logger) {
	idx := newIndex(root, files)

	router.Get("/notes", func(w http.ResponseWriter, _ *http.Request) {
		writeList(w, idx.current().files)
	})
	router.Get(filesPath+"*", serveFile(root, idx, logger))
	router.Post("/notes/sync", syncVault(idx, logger))
}

// syncVault returns the handler of POST /notes/sync: it scans the vault again and answers the new
// list. When the scan fails, the client gets a 500, logger gets the cause, and the list stays.
func syncVault(idx *index, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		latest, err := idx.rescan()
		if err != nil {
			logger.ErrorContext(r.Context(), "sync the vault", "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)

			return
		}

		writeList(w, latest.files)
	}
}

// writeList answers files as the list's JSON body.
func writeList(w http.ResponseWriter, files []string) {
	w.Header().Set("Content-Type", "application/json")
	// The vault lists only valid UTF-8 names, so the list always encodes: an error means the
	// client has gone.
	_ = json.MarshalWrite(w, list{Files: files})
}
