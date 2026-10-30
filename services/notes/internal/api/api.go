// Package api serves the vault's file list and its files over HTTP, and saves its notes.
package api

import (
	"encoding/json/v2"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// list is the body of GET /notes and POST /notes/sync: the paths of the vault's files; whether
// notes saves notes, which the page takes as false when it's left out (decision 400); and the web
// address of the vault's history, after which a note's path gives the note's own (decision 425).
type list struct {
	Files    []string `json:"files"`
	Editable bool     `json:"editable,omitzero"`
	History  string   `json:"history,omitempty"`
}

// Vault is the vault that notes serves: the folder of its checkout; the store that saves its
// notes, or nil when notes doesn't save them; and the web address of its history, or an empty
// string when notes doesn't link to it.
type Vault struct {
	Folder  string
	Store   Store
	History string
}

// listOf returns the list of files, with what notes does with them.
func (v Vault) listOf(files []string) list {
	return list{Files: files, Editable: v.Store != nil, History: v.History}
}

// Register scans the vault in its folder, and fails if it can't. Then it serves the vault on
// router: GET /notes lists its files, GET /notes/files/<path> returns one of them, and POST
// /notes/sync scans the vault again. With a store, PUT /notes/files/<path> saves a note in it, POST
// /notes/files/<path> makes a new one, and the list says so. Other methods get 405, and logger
// gets the errors that reading the vault, or writing to it, runs into.
func Register(router chi.Router, vault Vault, logger *slog.Logger) error {
	idx, err := newIndex(vault.Folder)
	if err != nil {
		return err
	}

	router.Get("/notes", func(w http.ResponseWriter, _ *http.Request) {
		writeList(w, vault.listOf(idx.files()))
	})
	router.Get(filesPath+"*", serveFile(idx, logger))
	router.Post("/notes/sync", syncVault(idx, vault, logger))

	if vault.Store != nil {
		router.Put(filesPath+"*", saver{idx: idx, store: vault.Store, logger: logger}.ServeHTTP)
		router.Post(filesPath+"*", creator{idx: idx, store: vault.Store, logger: logger}.ServeHTTP)
	}

	return nil
}

// syncVault returns the handler of POST /notes/sync: it scans the vault again and answers the new
// list, which says what notes does with the vault. When the scan fails, the client gets a 500,
// logger gets the cause, and the list stays.
func syncVault(idx *index, vault Vault, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		files, err := idx.rescan()
		if err != nil {
			logger.ErrorContext(r.Context(), "sync the vault", "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)

			return
		}

		writeList(w, vault.listOf(files))
	}
}

// writeList answers body as the list's JSON.
func writeList(w http.ResponseWriter, body list) {
	w.Header().Set("Content-Type", "application/json")
	// The vault lists only valid UTF-8 names, so the list always encodes: an error means the
	// client has gone.
	_ = json.MarshalWrite(w, body)
}
