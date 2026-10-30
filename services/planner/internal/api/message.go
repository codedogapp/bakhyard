package api

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/codedogapp/bakhyard/internal/jsonbody"
	"github.com/codedogapp/bakhyard/services/planner/internal/copilot"
)

// message returns the handler of POST /planner/plans/{id}/messages, which sends the body's text to
// the plan as the user's next message, and answers 204. A body that isn't a JSON object with just
// a text, or a blank text, gets a 400 with the reason, and a body over 64 KiB a 413. A plan that
// the planner doesn't keep gets a 404, and an error from Copilot a 502, whose cause logger gets.
func message(planner Planner, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if planner == nil {
			http.Error(w, notSetUp, http.StatusNotFound)

			return
		}

		var body reply
		if !jsonbody.Read(w, r, &body, "the body isn't a JSON object with just a text") {
			return
		}

		if strings.TrimSpace(body.Text) == "" {
			http.Error(w, "the message is blank", http.StatusBadRequest)

			return
		}

		switch err := planner.Send(r.Context(), r.PathValue("id"), body.Text); {
		case err == nil:
			w.WriteHeader(http.StatusNoContent)
		case errors.Is(err, copilot.ErrNoPlan):
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			logger.ErrorContext(r.Context(), "send a message", "err", err)
			http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		}
	}
}
