// Package api serves the planner's plans over HTTP.
package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/services/planner/internal/copilot"
)

// notSetUp is the answer of every route without Copilot.
const notSetUp = "Copilot is not set up"

// Planner works out plans with Copilot, as a copilot.Client does.
type Planner interface {
	// Plan starts a plan for prompt and returns its ID.
	Plan(ctx context.Context, prompt string) (string, error)
	// Conversation returns the conversation of the plan id so far, or copilot.ErrNoPlan when
	// the planner hasn't started that plan.
	Conversation(ctx context.Context, id string) (copilot.Conversation, error)
	// Follow returns the conversation of the plan id so far and its updates from then on, which
	// end with ctx or when the follower falls behind, or copilot.ErrNoPlan when the planner
	// hasn't started that plan.
	Follow(ctx context.Context, id string) (copilot.Conversation, <-chan copilot.Update, error)
	// Answer hands answer to its question of the plan id, which then no longer waits. It returns
	// copilot.ErrNoPlan when the planner hasn't started that plan, copilot.ErrNoQuestion when
	// the question doesn't wait, and copilot.ErrNotAChoice when the answer isn't one of the
	// choices of a question that takes no other.
	Answer(id string, answer copilot.Answer) error
}

// Door serves the planner's routes. Without Copilot, Planner is nil, and every route answers 404.
type Door struct {
	Planner Planner
	Logger  *slog.Logger
	// Heartbeat is how often a plan's event stream sends a comment, so that proxies keep it open.
	Heartbeat time.Duration
}

// Register serves the plans on router: POST /planner/plans starts a plan (start),
// GET /planner/plans/{id} answers its conversation so far (conversation),
// GET /planner/plans/{id}/events streams it (events), and
// POST /planner/plans/{id}/questions/{question}/answer answers one of its questions that wait
// (answer). stopping ends when the service is told to stop, and ends the event streams with it.
// Other methods get 405.
func (d Door) Register(stopping context.Context, router chi.Router) {
	router.Post("/planner/plans", start(d.Planner, d.Logger))
	router.Get("/planner/plans/{id}", conversation(d.Planner, d.Logger))
	router.Get("/planner/plans/{id}/events", func(w http.ResponseWriter, r *http.Request) {
		d.events(stopping, w, r)
	})
	router.Post("/planner/plans/{id}/questions/{question}/answer", answer(d.Planner, d.Logger))
}
