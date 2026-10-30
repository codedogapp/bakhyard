package api_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/services/planner/internal/copilot"
)

func TestReadingAPlanAnswersItsConversation(t *testing.T) {
	talk := copilot.Conversation{
		Messages: []copilot.Message{
			{ID: "1", From: copilot.User, Text: "Plan a trip"},
			{ID: "m2", From: copilot.Assistant, Text: "Where to?"},
		},
		Writing:   []copilot.Message{{ID: "m3", From: copilot.Assistant, Text: "So"}},
		Questions: []copilot.Question{{ID: "q1", Text: "Which city?", Choices: []string{"Rome"}}},
		Plan:      "# Plan\n",
		Working:   true,
	}
	stopped := copilot.Conversation{Failure: "Quota exceeded"}

	for _, test := range []struct {
		talk copilot.Conversation
		want string
	}{
		{talk, `{"messages":[{"id":"1","from":"user","text":"Plan a trip"},` +
			`{"id":"m2","from":"assistant","text":"Where to?"}],` +
			`"writing":[{"id":"m3","from":"assistant","text":"So"}],` +
			`"questions":[{"id":"q1","text":"Which city?","choices":["Rome"],"freeform":false}],` +
			`"plan":"# Plan\n","working":true,"failure":""}`},
		{stopped, `{"messages":[],"writing":[],"questions":[],"plan":"",` +
			`"working":false,"failure":"Quota exceeded"}`},
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
