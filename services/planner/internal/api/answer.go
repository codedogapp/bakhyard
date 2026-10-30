package api

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/codedogapp/bakhyard/internal/jsonbody"
	"github.com/codedogapp/bakhyard/services/planner/internal/copilot"
)

// reply is the user's text: the body of POST /planner/plans/{id}/questions/{question}/answer, and
// of POST /planner/plans/{id}/messages.
type reply struct {
	Text string `json:"text"`
}

// answer returns the handler of POST /planner/plans/{id}/questions/{question}/answer, which hands
// the body's text to the plan's question that waits, and answers 204. A body that isn't a JSON
// object with just a text, a blank text, or a text that isn't one of the choices of a question
// that takes no other, gets a 400 with the reason, and a body over 64 KiB a 413. A plan that the
// planner hasn't started, or a question that doesn't wait, gets a 404, and another error a 500,
// whose cause logger gets.
func answer(planner Planner, logger *slog.Logger) http.HandlerFunc {
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
			http.Error(w, "the answer is blank", http.StatusBadRequest)

			return
		}

		given := copilot.Answer{Question: r.PathValue("question"), Text: body.Text}

		switch err := planner.Answer(r.PathValue("id"), given); {
		case err == nil:
			w.WriteHeader(http.StatusNoContent)
		case errors.Is(err, copilot.ErrNoPlan), errors.Is(err, copilot.ErrNoQuestion):
			http.Error(w, err.Error(), http.StatusNotFound)
		case errors.Is(err, copilot.ErrNotAChoice):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			logger.ErrorContext(r.Context(), "answer a question", "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)
		}
	}
}
