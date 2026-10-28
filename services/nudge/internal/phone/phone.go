// Package phone is nudge's door for the user's page: the questions to answer, the sessions that
// run, and the stream that tells the page when either changes.
package phone

import (
	"context"
	"encoding/json/v2"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/services/nudge/internal/change"
	"github.com/codedogapp/bakhyard/services/nudge/internal/question"
	"github.com/codedogapp/bakhyard/services/nudge/internal/session"
)

// maxAnswer caps an answer's body: a reply of a few pages.
const maxAnswer = 64 << 10

// Door serves the page's routes.
type Door struct {
	Questions *question.Store
	Sessions  *session.Registry
	Changes   *change.Hub
	Logger    *slog.Logger
	// Heartbeat is how often the event stream sends a comment, so that proxies keep it open.
	Heartbeat time.Duration
}

// state is the body of GET /nudge/state.
type state struct {
	Questions []question.Question `json:"questions"`
	Sessions  []session.Session   `json:"sessions"`
}

// Register serves the door on router: GET /nudge/state lists what the page shows, POST
// /nudge/questions/{id}/answer answers a question and GET /nudge/events streams a signal
// whenever the state changes. stopping ends when the service is told to stop, and ends the event
// streams with it.
func (d Door) Register(stopping context.Context, router chi.Router) {
	router.Get("/nudge/state", d.state)
	router.Post("/nudge/questions/{id}/answer", d.answer)
	router.Get("/nudge/events", func(w http.ResponseWriter, r *http.Request) {
		d.events(stopping, w, r)
	})
}

// state answers the pending questions, newest first, and the running sessions.
func (d Door) state(w http.ResponseWriter, r *http.Request) {
	questions, err := d.Questions.Pending(r.Context())
	if err != nil {
		d.Logger.ErrorContext(r.Context(), "list the questions", "err", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.MarshalWrite(w, state{Questions: questions, Sessions: d.Sessions.List()})
}

// answer hands the answer in the body to the agent that asks the question in the path, and
// answers 204. A question that has gone gets a 404, and an answer that doesn't fit it a 400 that
// says why.
func (d Door) answer(w http.ResponseWriter, r *http.Request) {
	var given question.Answer

	err := json.UnmarshalRead(http.MaxBytesReader(w, r.Body, maxAnswer), &given,
		json.RejectUnknownMembers(true))
	if err != nil {
		http.Error(w, "read the answer: "+err.Error(), http.StatusBadRequest)

		return
	}

	switch err := d.Questions.Reply(r.Context(), r.PathValue("id"), given); {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, question.ErrNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, question.ErrEmptyAnswer), errors.Is(err, question.ErrUnknownOption):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		d.Logger.ErrorContext(r.Context(), "answer the question", "err", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}
}
