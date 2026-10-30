// Package api serves the vault's file list and its files over HTTP, and saves its notes.
package api

import (
	"encoding/json/v2"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// list is the body of GET /notes and POST /notes/sync: the paths of the vault's files, and whether
// notes saves notes, which the page takes as false when it's left out (decision 400).
type list struct {
	Files    []string `json:"files"`
	Editable bool     `json:"editable,omitzero"`
}

// Register scans the vault in folder, and fails if it can't. Then it serves the vault on router:
// GET /notes lists its files, GET /notes/files/<path> returns one of them, and POST /notes/sync
// scans the vault again. With a store, PUT /notes/files/<path> saves a note in it, POST
// /notes/files/<path> makes a new one, and the list says so. Other methods get 405, and logger
// gets the errors that reading the vault, or writing to it, runs into.
func Register(router chi.Router, folder string, store Store, logger *slog.Logger) error {
	idx, err := newIndex(folder)
	if err != nil {
		return err
	}

	editable := store != nil

	router.Get("/notes", func(w http.ResponseWriter, _ *http.Request) {
		writeList(w, list{Files: idx.files(), Editable: editable})
	})
	router.Get(filesPath+"*", serveFile(idx, logger))
	router.Post("/notes/sync", syncVault(idx, editable, logger))

	if editable {
		router.Put(filesPath+"*", saver{idx: idx, store: store, logger: logger}.ServeHTTP)
		router.Post(filesPath+"*", creator{idx: idx, store: store, logger: logger}.ServeHTTP)
	}

	return nil
}

// syncVault returns the handler of POST /notes/sync: it scans the vault again and answers the new
// list, which says whether the vault is editable. When the scan fails, the client gets a 500,
// logger gets the cause, and the list stays.
func syncVault(idx *index, editable bool, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		files, err := idx.rescan()
		if err != nil {
			logger.ErrorContext(r.Context(), "sync the vault", "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)

			return
		}

		writeList(w, list{Files: files, Editable: editable})
	}
}

// writeList answers body as the list's JSON.
func writeList(w http.ResponseWriter, body list) {
	w.Header().Set("Content-Type", "application/json")
	// The vault lists only valid UTF-8 names, so the list always encodes: an error means the
	// client has gone.
	_ = json.MarshalWrite(w, body)
}
