package api_test

import (
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"
)

// messageTarget is the route that sends a message to the plan plan-1.
const messageTarget = "/planner/plans/plan-1/messages"

func TestSendingAMessageHandsItTheText(t *testing.T) {
	for body, text := range map[string]string{
		`{"text":"Make it a week"}`:     "Make it a week",
		`{"text":" Make it a week.\n"}`: " Make it a week.\n",
	} {
		fake := &standIn{}

		rec := newPlannerAPI(fake).serve(t, http.MethodPost, messageTarget, body)
		if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
			t.Errorf("%s: got %d %q, want 204 and no body", body, rec.Code, rec.Body)
		}

		if want := []string{text}; !slices.Equal(fake.messages, want) {
			t.Errorf("%s: Copilot got %q, want %q", body, fake.messages, want)
		}
	}
}

func TestSendingAMessageTakesAJSONObjectWithAText(t *testing.T) {
	refusal := "the body isn't a JSON object with just a text\n"

	for body, want := range map[string]string{
		`{"text":"Make it a week","prompt":"Plan a trip"}`: refusal,
		`{"text":["Make it a week"]}`:                      refusal,
		`Make it a week`:                                   refusal,
		`{"text":" \n\t"}`:                                 "the message is blank\n",
		`{}`:                                               "the message is blank\n",
	} {
		fake := &standIn{}

		rec := newPlannerAPI(fake).serve(t, http.MethodPost, messageTarget, body)
		if rec.Code != http.StatusBadRequest || rec.Body.String() != want ||
			len(fake.messages) != 0 {
			t.Errorf("%s: got %d %q and Copilot got %q, want 400 %q and nothing for Copilot",
				body, rec.Code, rec.Body, fake.messages, want)
		}
	}
}

func TestSendingAMessageTakesUpTo64KiB(t *testing.T) {
	text := strings.Repeat("a", 64<<10)

	rec := newPlannerAPI(&standIn{}).serve(t, http.MethodPost, messageTarget,
		`{"text":"`+text+`"}`)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("got %d, want 413", rec.Code)
	}
}

func TestSendingAMessageToAPlanNotStartedIsNotFound(t *testing.T) {
	planner := newPlannerAPI(&standIn{})

	rec := planner.serve(t, http.MethodPost, "/planner/plans/plan-2/messages",
		`{"text":"Make it a week"}`)
	if rec.Code != http.StatusNotFound || rec.Body.String() != "no such plan\n" ||
		planner.logs.Len() != 0 {
		t.Errorf("got %d %q and logged %q, want 404 no such plan and no log", rec.Code,
			rec.Body, planner.logs)
	}
}

func TestSendingAMessageFailsWhenCopilotFails(t *testing.T) {
	planner := newPlannerAPI(&standIn{failure: errors.New("no connection")})

	rec := planner.serve(t, http.MethodPost, messageTarget, `{"text":"Make it a week"}`)
	if rec.Code != http.StatusBadGateway || rec.Body.String() != "Bad Gateway\n" {
		t.Errorf("got %d %q, want 502 Bad Gateway", rec.Code, rec.Body)
	}

	logs := planner.logs.String()
	if !strings.Contains(logs, `msg="send a message" err="no connection"`) {
		t.Errorf("logged %q, want the cause", logs)
	}
}
