package api_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/services/planner/internal/api"
	"github.com/codedogapp/bakhyard/services/planner/internal/copilot"
)

// standIn is a stand-in for Copilot: it starts the plan "plan-1", whose conversation is talk,
// unless it has a failure, and notes the prompts it gets.
type standIn struct {
	talk    copilot.Conversation
	failure error
	prompts []string
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

// plannerAPI is the planner's API, with the logs it writes.
type plannerAPI struct {
	router http.Handler
	logs   *bytes.Buffer
}

// newPlannerAPI returns the API that Register serves from planner.
func newPlannerAPI(planner api.Planner) plannerAPI {
	logs := &bytes.Buffer{}
	router := chi.NewRouter()
	api.Register(router, planner, slog.New(slog.NewTextHandler(logs, nil)))

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

func TestStartingAPlanAnswersItsID(t *testing.T) {
	fake := &standIn{}

	rec := newPlannerAPI(fake).serve(t, http.MethodPost, "/planner/plans",
		`{"prompt":"Plan a trip"}`)

	sent := rec.Result().Header.Get("Content-Type")
	if rec.Code != http.StatusCreated || rec.Body.String() != `{"id":"plan-1"}` ||
		sent != "application/json" {
		t.Errorf("got %d %s (%s), want 201 {\"id\":\"plan-1\"} as JSON", rec.Code, rec.Body, sent)
	}

	if !slices.Equal(fake.prompts, []string{"Plan a trip"}) {
		t.Errorf("Copilot got %q, want the prompt", fake.prompts)
	}
}

func TestStartingAPlanTakesAJSONObjectWithAPrompt(t *testing.T) {
	refusal := "the body isn't a JSON object with just a prompt\n"

	for body, want := range map[string]string{
		`{"prompt":"Plan a trip","model":"gpt"}`: refusal,
		`{"prompt":["Plan a trip"]}`:             refusal,
		`Plan a trip`:                            refusal,
		`{"prompt":" \n\t"}`:                     "the prompt is blank\n",
		`{}`:                                     "the prompt is blank\n",
	} {
		fake := &standIn{}

		rec := newPlannerAPI(fake).serve(t, http.MethodPost, "/planner/plans", body)
		if rec.Code != http.StatusBadRequest || rec.Body.String() != want ||
			len(fake.prompts) != 0 {
			t.Errorf("%s: got %d %q and Copilot got %q, want 400 %q and nothing for Copilot",
				body, rec.Code, rec.Body, fake.prompts, want)
		}
	}
}

func TestStartingAPlanTakesUpTo64KiB(t *testing.T) {
	prompt := strings.Repeat("a", 64<<10)

	rec := newPlannerAPI(&standIn{}).serve(t, http.MethodPost, "/planner/plans",
		`{"prompt":"`+prompt+`"}`)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("got %d, want 413", rec.Code)
	}
}

func TestStartingAPlanFailsWhenCopilotFails(t *testing.T) {
	planner := newPlannerAPI(&standIn{failure: errors.New("no connection")})

	rec := planner.serve(t, http.MethodPost, "/planner/plans", `{"prompt":"Plan a trip"}`)
	if rec.Code != http.StatusBadGateway || rec.Body.String() != "Bad Gateway\n" {
		t.Errorf("got %d %q, want 502 Bad Gateway", rec.Code, rec.Body)
	}

	logs := planner.logs.String()
	if !strings.Contains(logs, `msg="start a plan" err="no connection"`) {
		t.Errorf("logged %q, want the cause", logs)
	}
}

func TestReadingAPlanAnswersItsConversation(t *testing.T) {
	talk := copilot.Conversation{
		Messages: []copilot.Message{
			{ID: "1", From: copilot.User, Text: "Plan a trip"},
			{ID: "m2", From: copilot.Assistant, Text: "Where to?"},
		},
		Writing: []copilot.Message{{ID: "m3", From: copilot.Assistant, Text: "So"}},
		Plan:    "# Plan\n",
	}

	for _, test := range []struct {
		talk copilot.Conversation
		want string
	}{
		{talk, `{"messages":[{"id":"1","from":"user","text":"Plan a trip"},` +
			`{"id":"m2","from":"assistant","text":"Where to?"}],` +
			`"writing":[{"id":"m3","from":"assistant","text":"So"}],"plan":"# Plan\n"}`},
		{copilot.Conversation{}, `{"messages":[],"writing":[],"plan":""}`},
	} {
		rec := newPlannerAPI(&standIn{talk: test.talk}).serve(t, http.MethodGet,
			"/planner/plans/plan-1", "")

		sent := rec.Result().Header.Get("Content-Type")
		if rec.Code != http.StatusOK || rec.Body.String() != test.want ||
			sent != "application/json" {
			t.Errorf("got %d %s (%s), want 200 %s as JSON", rec.Code, rec.Body, sent, test.want)
		}
	}
}

func TestReadingAnUnknownPlanIsNotFound(t *testing.T) {
	rec := newPlannerAPI(&standIn{}).serve(t, http.MethodGet, "/planner/plans/plan-2", "")
	if rec.Code != http.StatusNotFound || rec.Body.String() != "no such plan\n" {
		t.Errorf("got %d %q, want 404 no such plan", rec.Code, rec.Body)
	}
}

func TestReadingAPlanFailsWhenCopilotFails(t *testing.T) {
	planner := newPlannerAPI(&standIn{failure: errors.New("no connection")})

	rec := planner.serve(t, http.MethodGet, "/planner/plans/plan-1", "")
	if rec.Code != http.StatusBadGateway || rec.Body.String() != "Bad Gateway\n" {
		t.Errorf("got %d %q, want 502 Bad Gateway", rec.Code, rec.Body)
	}

	logs := planner.logs.String()
	if !strings.Contains(logs, `msg="read a plan" err="no connection"`) {
		t.Errorf("logged %q, want the cause", logs)
	}
}

func TestEachRouteIsNotFoundWithoutCopilot(t *testing.T) {
	planner := newPlannerAPI(nil)

	for method, target := range map[string]string{
		http.MethodPost: "/planner/plans",
		http.MethodGet:  "/planner/plans/plan-1",
	} {
		rec := planner.serve(t, method, target, `{"prompt":"Plan a trip"}`)
		if rec.Code != http.StatusNotFound || rec.Body.String() != "Copilot is not set up\n" {
			t.Errorf("%s %s: got %d %q, want 404 Copilot is not set up", method, target,
				rec.Code, rec.Body)
		}
	}
}

func TestEachRouteTakesOnlyItsMethod(t *testing.T) {
	planner := newPlannerAPI(&standIn{})

	for _, route := range []struct{ method, target string }{
		{http.MethodGet, "/planner/plans"},
		{http.MethodPut, "/planner/plans"},
		{http.MethodPost, "/planner/plans/plan-1"},
		{http.MethodDelete, "/planner/plans/plan-1"},
	} {
		rec := planner.serve(t, route.method, route.target, "")
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s %s: got %d, want 405", route.method, route.target, rec.Code)
		}
	}
}
