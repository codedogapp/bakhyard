package api_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/services/planner/internal/api"
	"github.com/codedogapp/bakhyard/services/planner/internal/copilot"
)

// standIn is a stand-in for Copilot: it lists plans, starts the plan "plan-1", whose
// conversation is talk, whose updates come from updates and whose questions take any answer,
// unless it has a failure, and notes the prompts, the messages and the answers it gets and the
// context that the plan is followed for.
type standIn struct {
	plans    []copilot.Entry
	talk     copilot.Conversation
	updates  chan copilot.Update
	failure  error
	prompts  []string
	messages []string
	answers  []copilot.Answer
	followed context.Context
}

func (s *standIn) Plans(context.Context) ([]copilot.Entry, error) {
	return s.plans, s.failure
}

func (s *standIn) Plan(_ context.Context, prompt string) (string, error) {
	s.prompts = append(s.prompts, prompt)

	return "plan-1", s.failure
}

func (s *standIn) Conversation(_ context.Context, id string) (copilot.Conversation, error) {
	if s.failure != nil {
		return copilot.Conversation{}, s.failure
	}

	if id != "plan-1" {
		return copilot.Conversation{}, copilot.ErrNoPlan
	}

	return s.talk, nil
}

func (s *standIn) Follow(
	ctx context.Context,
	id string,
) (copilot.Conversation, <-chan copilot.Update, error) {
	s.followed = ctx
	talk, err := s.Conversation(ctx, id)

	return talk, s.updates, err
}

func (s *standIn) Answer(id string, answer copilot.Answer) error {
	if id != "plan-1" {
		return copilot.ErrNoPlan
	}

	s.answers = append(s.answers, answer)

	return s.failure
}

func (s *standIn) Send(_ context.Context, id, text string) error {
	if id != "plan-1" {
		return copilot.ErrNoPlan
	}

	s.messages = append(s.messages, text)

	return s.failure
}

// plannerAPI is the planner's API, with the logs it writes.
type plannerAPI struct {
	router http.Handler
	logs   *bytes.Buffer
}

// newPlannerAPI returns the API that a Door serves from planner, in a service that never stops.
func newPlannerAPI(planner api.Planner) plannerAPI {
	logs := &bytes.Buffer{}
	router := chi.NewRouter()
	door := api.Door{Planner: planner, Logger: slog.New(slog.NewTextHandler(logs, nil))}
	door.Register(context.Background(), router)

	return plannerAPI{router: router, logs: logs}
}

// serve returns the API's answer to a request for method on target with body.
func (a plannerAPI) serve(t *testing.T, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), method, target, strings.NewReader(body))
	rec := httptest.NewRecorder()
	a.router.ServeHTTP(rec, req)

	return rec
}

func TestEachRouteIsNotFoundWithoutCopilot(t *testing.T) {
	planner := newPlannerAPI(nil)

	for _, route := range []struct{ method, target string }{
		{http.MethodGet, "/planner/plans"},
		{http.MethodPost, "/planner/plans"},
		{http.MethodGet, "/planner/plans/plan-1"},
		{http.MethodGet, "/planner/plans/plan-1/events"},
		{http.MethodPost, "/planner/plans/plan-1/messages"},
		{http.MethodPost, "/planner/plans/plan-1/questions/q1/answer"},
	} {
		rec := planner.serve(t, route.method, route.target, `{"prompt":"Plan a trip"}`)
		if rec.Code != http.StatusNotFound || rec.Body.String() != "Copilot is not set up\n" {
			t.Errorf("%s %s: got %d %q, want 404 Copilot is not set up", route.method,
				route.target, rec.Code, rec.Body)
		}
	}
}

func TestEachRouteTakesOnlyItsMethod(t *testing.T) {
	planner := newPlannerAPI(&standIn{})

	for _, route := range []struct{ method, target string }{
		{http.MethodPut, "/planner/plans"},
		{http.MethodDelete, "/planner/plans"},
		{http.MethodPost, "/planner/plans/plan-1"},
		{http.MethodDelete, "/planner/plans/plan-1"},
		{http.MethodPost, "/planner/plans/plan-1/events"},
		{http.MethodGet, "/planner/plans/plan-1/messages"},
		{http.MethodGet, "/planner/plans/plan-1/questions/q1/answer"},
	} {
		rec := planner.serve(t, route.method, route.target, "")
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s %s: got %d, want 405", route.method, route.target, rec.Code)
		}
	}
}
