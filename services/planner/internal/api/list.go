package api

import (
	"encoding/json/v2"
	"log/slog"
	"net/http"

	"github.com/codedogapp/bakhyard/services/planner/internal/copilot"
)

// plans is the answer of GET /planner/plans.
type plans struct {
	Plans []copilot.Entry `json:"plans"`
}

// list returns the handler of GET /planner/plans, which answers the plans, those of the service's
// earlier runs among them, the last updated first. An error from Copilot gets a 502, whose cause
// logger gets.
func list(planner Planner, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if planner == nil {
			http.Error(w, notSetUp, http.StatusNotFound)

			return
		}

		entries, err := planner.Plans(r.Context())
		if err != nil {
			logger.ErrorContext(r.Context(), "list the plans", "err", err)
			http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		// The list comes from the CLI's JSON, which the SDK has decoded, so it always encodes: an
		// error means the client has gone.
		_ = json.MarshalWrite(w, plans{Plans: entries})
	}
}
