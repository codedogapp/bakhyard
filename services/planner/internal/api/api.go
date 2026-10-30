// Package api serves the planner's plans over HTTP.
package api

import (
	"context"
	"log/slog"

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
}

// Register serves the plans on router: POST /planner/plans starts a plan (start), and
// GET /planner/plans/{id} answers its conversation so far (conversation). Without Copilot,
// planner is nil and the client gets a 404. Other methods get 405.
func Register(router chi.Router, planner Planner, logger *slog.Logger) {
	router.Post("/planner/plans", start(planner, logger))
	router.Get("/planner/plans/{id}", conversation(planner, logger))
}
