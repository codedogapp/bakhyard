package api_test

import (
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"
)

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
