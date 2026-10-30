package api

import (
	"encoding/json/v2"
	"log/slog"
	"net/http"
	"strings"

	"github.com/codedogapp/bakhyard/internal/jsonbody"
)

// request is the body of POST /planner/plans.
type request struct {
	Prompt string `json:"prompt"`
}

// started is the answer of POST /planner/plans.
type started struct {
	ID string `json:"id"`
}

// start returns the handler of POST /planner/plans, which starts a plan for the body's prompt
// and answers 201 with the plan's ID. A body that isn't a JSON object with just a prompt, or a
// blank prompt, gets a 400 with the reason; a body over 64 KiB gets a 413, and an error from
// Copilot a 502, whose cause logger gets.
func start(planner Planner, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if planner == nil {
			http.Error(w, notSetUp, http.StatusNotFound)

			return
		}

		var body request
		if !jsonbody.Read(w, r, &body, "the body isn't a JSON object with just a prompt") {
			return
		}

		if strings.TrimSpace(body.Prompt) == "" {
			http.Error(w, "the prompt is blank", http.StatusBadRequest)

			return
		}

		planID, err := planner.Plan(r.Context(), body.Prompt)
		if err != nil {
			logger.ErrorContext(r.Context(), "start a plan", "err", err)
			http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		// The ID is a UUID, so it always encodes: an error means the client has gone.
		_ = json.MarshalWrite(w, started{ID: planID})
	}
}
