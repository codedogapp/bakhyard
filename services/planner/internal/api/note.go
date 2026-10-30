package api

import (
	"encoding/json/v2"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/codedogapp/bakhyard/services/planner/internal/copilot"
)

// saved is the answer to a plan saved in the vault.
type saved struct {
	// Path is the note's path in the vault.
	Path string `json:"path"`
}

// note returns the handler of POST /planner/plans/{id}/note, which saves the plan's plan in the
// vault as a new note and answers 201 with the note's path. Without a vault it answers 404, as it
// does for a plan that the planner doesn't keep, and a plan that Copilot hasn't written yet gets a
// 409. An error from Copilot or the vault gets a 502, whose cause logger gets.
func note(planner Planner, vault Saver, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if planner == nil {
			http.Error(w, notSetUp, http.StatusNotFound)

			return
		}

		if vault == nil {
			http.Error(w, "Saving to the vault is not set up", http.StatusNotFound)

			return
		}

		talk, err := planner.Conversation(r.Context(), r.PathValue("id"))
		if errors.Is(err, copilot.ErrNoPlan) {
			http.Error(w, err.Error(), http.StatusNotFound)

			return
		}

		if err != nil {
			logger.ErrorContext(r.Context(), "read a plan", "err", err)
			http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)

			return
		}

		if strings.TrimSpace(talk.Plan) == "" {
			http.Error(w, "Copilot hasn't written the plan yet", http.StatusConflict)

			return
		}

		path, err := vault.Save(r.Context(), talk.Plan)
		if err != nil {
			logger.ErrorContext(r.Context(), "save a plan", "err", err)
			http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		// A path that notes answered in JSON is valid UTF-8, so it always encodes: an error means
		// the client has gone.
		_ = json.MarshalWrite(w, saved{Path: path})
	}
}
