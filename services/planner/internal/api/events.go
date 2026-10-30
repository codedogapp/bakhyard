package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/codedogapp/bakhyard/internal/eventstream"
	"github.com/codedogapp/bakhyard/services/planner/internal/copilot"
)

// planned is the data of the event plan.
type planned struct {
	Plan string `json:"plan"`
}

// events streams the conversation of the plan in the path: first the conversation so far, as
// GET /planner/plans/{id} answers it, then each piece of the assistant's text (delta), each whole
// message (message), each question of the agent's that waits for the user's answer (question)
// and each that has had it (answered), the plan (plan), and each change of whether the agent is
// working, or of the error that stopped it (status), as they come. The stream lasts until the
// page leaves or the service stops, or until the plan's updates end, after which the page's
// EventSource connects again. HEAD gets the stream's headers alone: a stream would hold its
// connection, and the next request on it would wait.
func (d Door) events(stopping context.Context, w http.ResponseWriter, r *http.Request) {
	talk, updates, ok := d.follow(w, r)
	if !ok {
		return
	}

	stream, err := eventstream.Open(w)
	if err != nil || r.Method == http.MethodHead {
		return
	}

	err = stream.Send("conversation", talk)
	heartbeat := time.Tick(d.Heartbeat)

	for err == nil {
		select {
		case update, open := <-updates:
			if !open {
				return
			}

			err = send(stream, talk, update)
		case <-heartbeat:
			err = stream.Beat()
		case <-r.Context().Done():
			return
		case <-stopping.Done():
			return
		}
	}
}

// follow follows the plan in the path until the page leaves, and reports whether it can. Without
// Copilot, or for a plan that the planner doesn't keep, it answers 404; for an error from
// Copilot it answers 502, and the logger gets the cause.
func (d Door) follow(
	w http.ResponseWriter,
	r *http.Request,
) (copilot.Conversation, <-chan copilot.Update, bool) {
	if d.Planner == nil {
		http.Error(w, notSetUp, http.StatusNotFound)

		return copilot.Conversation{}, nil, false
	}

	talk, updates, err := d.Planner.Follow(r.Context(), r.PathValue("id"))
	if errors.Is(err, copilot.ErrNoPlan) {
		http.Error(w, err.Error(), http.StatusNotFound)

		return copilot.Conversation{}, nil, false
	}

	if err != nil {
		d.Logger.ErrorContext(r.Context(), "follow a plan", "err", err)
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)

		return copilot.Conversation{}, nil, false
	}

	return talk, updates, true
}

// send sends the page update as the event that its change names, unless the conversation that
// the page got first already has it.
func send(stream eventstream.Stream, talk copilot.Conversation, update copilot.Update) error {
	switch {
	case talk.Repeats(update):
		return nil
	case update.Change == copilot.ChangePlan:
		return stream.Send(string(update.Change), planned{Plan: update.Plan})
	case update.Change == copilot.ChangeQuestion, update.Change == copilot.ChangeAnswered:
		return stream.Send(string(update.Change), update.Question)
	case update.Change == copilot.ChangeStatus:
		return stream.Send(string(update.Change), update.Status)
	default:
		return stream.Send(string(update.Change), update.Message)
	}
}
