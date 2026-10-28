package agent

import (
	"context"
	"encoding/json/v2"
	"errors"
	"net/http"
	"time"

	"github.com/codedogapp/bakhyard/services/nudge/internal/notifier"
	"github.com/codedogapp/bakhyard/services/nudge/internal/question"
)

const (
	// pushTimeout bounds the push that a new question causes.
	pushTimeout = 30 * time.Second
	// notificationPath is the page that a tap on a question's notification opens.
	notificationPath = "/nudge"
)

// errStopping is why a held request ends when the service stops.
var errStopping = errors.New("nudge is stopping")

// asked is the body of POST /agent/questions: the question, with the ID that its agent chose.
type asked struct {
	ID      string   `json:"id"`
	Project string   `json:"project"`
	Text    string   `json:"text"`
	Details string   `json:"details"`
	Options []string `json:"options"`
}

// ask holds the request until the question has an answer, the agent leaves or the service stops.
// A new question reaches the user's devices as a push. Asking again with the ID of a question
// that's there attaches to it, so an agent whose connection broke gets its answer without a
// second push. A question that fails its rules gets a 400, one that was cancelled or expired
// a 410, and a stopping service a 503.
func (d Door) ask(w http.ResponseWriter, r *http.Request) {
	var body asked
	if !readBody(w, r, &body, "question") {
		return
	}

	ctx, cancel := context.WithCancelCause(r.Context())
	defer cancel(nil)

	stop := context.AfterFunc(d.Stopping, func() { cancel(errStopping) })
	defer stop()

	created, err := d.Questions.Ask(ctx, question.Question{
		ID: body.ID, Project: body.Project, Text: body.Text, Details: body.Details,
		Options: body.Options,
	})
	if d.refuseAsk(w, r, err) {
		return
	}

	if created {
		d.push(r.Context(), body)
	}

	answer, err := d.Questions.Wait(ctx, body.ID)

	switch {
	case err == nil:
		w.Header().Set("Content-Type", "application/json")
		_ = json.MarshalWrite(w, answer)
	case errors.Is(err, question.ErrNotFound):
		http.Error(w, err.Error(), http.StatusGone)
	case errors.Is(context.Cause(ctx), errStopping):
		http.Error(w, errStopping.Error(), http.StatusServiceUnavailable)
	case r.Context().Err() == nil:
		d.serverError(w, r, "wait for the answer", err)
	}
}

// refuseAsk answers the failed Ask, and reports whether it did. An agent that has gone is
// answered by nobody.
func (d Door) refuseAsk(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}

	switch {
	case errors.Is(err, question.ErrBadID), errors.Is(err, question.ErrNoText),
		errors.Is(err, question.ErrBadOptions):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case r.Context().Err() == nil:
		d.serverError(w, r, "ask the question", err)
	}

	return true
}

// push tells the user about a new question, in the background: the agent waits for its answer, not
// for the push, and a push that fails changes nothing, as the page lists the question anyway.
func (d Door) push(ctx context.Context, q asked) {
	message := notifier.Message{Title: q.Project, Body: q.Text, URL: notificationPath}

	go func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), pushTimeout)
		defer cancel()

		if err := d.Pusher.Send(ctx, message); err != nil {
			d.Logger.WarnContext(ctx, "push the question", "err", err)
		}
	}()
}

// withdraw forgets the question in the path, which ends the request that waits for it with 410.
// It answers 204 whether the question was there or not.
func (d Door) withdraw(w http.ResponseWriter, r *http.Request) {
	if err := d.Questions.Cancel(r.Context(), r.PathValue("id")); err != nil {
		d.serverError(w, r, "withdraw the question", err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
