// Package agent is nudge's door for agents: where each question waits for its answer, and each
// session reports its progress.
package agent

import (
	"context"
	"encoding/json/v2"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/internal/notifier"
	"github.com/codedogapp/bakhyard/services/nudge/internal/question"
	"github.com/codedogapp/bakhyard/services/nudge/internal/session"
)

// maxBody caps a request's body: a few lines of text, and details of a few pages.
const maxBody = 64 << 10

// Pusher sends a notification to the user's devices.
type Pusher interface {
	Send(ctx context.Context, message notifier.Message) error
}

// Door serves the agents' routes.
type Door struct {
	Questions *question.Store
	Sessions  *session.Registry
	Pusher    Pusher
	Logger    *slog.Logger
}

// Register serves the door on router: POST /agent/questions asks a question and holds the request
// until it has an answer, and DELETE /agent/questions/{id} withdraws one. POST /agent/sessions
// starts a session, which lasts as long as its request, and PUT /agent/sessions/{id}/note sets
// its note. stopping ends when the service is told to stop: the requests that wait end too, with
// 503 for a question, so that the agents ask again once the service is back.
func (d Door) Register(stopping context.Context, router chi.Router) {
	router.Post("/agent/questions", func(w http.ResponseWriter, r *http.Request) {
		d.ask(stopping, w, r)
	})
	router.Delete("/agent/questions/{id}", d.withdraw)
	router.Post("/agent/sessions", func(w http.ResponseWriter, r *http.Request) {
		d.startSession(stopping, w, r)
	})
	router.Put("/agent/sessions/{id}/note", d.report)
}

// readBody decodes r's body, a JSON object without members that v lacks, into v. When it can't,
// the client gets a 400 that says why, or a 413 for a body over maxBody, and readBody returns
// false.
func readBody(w http.ResponseWriter, r *http.Request, v any, what string) bool {
	err := json.UnmarshalRead(http.MaxBytesReader(w, r.Body, maxBody), v,
		json.RejectUnknownMembers(true))
	if _, tooLarge := errors.AsType[*http.MaxBytesError](err); tooLarge {
		http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge),
			http.StatusRequestEntityTooLarge)

		return false
	}

	if err != nil {
		http.Error(w, "read the "+what+": "+err.Error(), http.StatusBadRequest)

		return false
	}

	return true
}

// serverError answers a plain 500, and logs err as what.
func (d Door) serverError(w http.ResponseWriter, r *http.Request, what string, err error) {
	d.Logger.ErrorContext(r.Context(), what, "err", err)
	http.Error(w, http.StatusText(http.StatusInternalServerError),
		http.StatusInternalServerError)
}
