package api

import (
	"encoding/json/v2"
	"errors"
	"log/slog"
	"net/http"

	"github.com/codedogapp/bakhyard/services/planner/internal/copilot"
)

// conversation returns the handler of GET /planner/plans/{id}, which answers the plan's
// conversation so far, and its plan once the agent has brought it. A plan that the planner
// doesn't keep gets a 404, and an error from Copilot a 502, whose cause logger gets.
func conversation(planner Planner, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if planner == nil {
			http.Error(w, notSetUp, http.StatusNotFound)

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

		w.Header().Set("Content-Type", "application/json")
		// The SDK decodes the CLI's JSON into valid UTF-8, so the conversation always encodes:
		// an error means the client has gone.
		_ = json.MarshalWrite(w, talk)
	}
}
