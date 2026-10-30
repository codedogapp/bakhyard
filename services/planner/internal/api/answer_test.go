package api_test

import (
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/services/planner/internal/copilot"
)

// answerTarget is the route that answers the question q1 of the plan plan-1.
const answerTarget = "/planner/plans/plan-1/questions/q1/answer"

func TestAnsweringAQuestionHandsItTheText(t *testing.T) {
	for body, text := range map[string]string{
		`{"text":"Rome"}`:                    "Rome",
		`{"text":" Rome, then Florence.\n"}`: " Rome, then Florence.\n",
	} {
		fake := &standIn{}

		rec := newPlannerAPI(fake).serve(t, http.MethodPost, answerTarget, body)
		if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
			t.Errorf("%s: got %d %q, want 204 and no body", body, rec.Code, rec.Body)
		}

		want := []copilot.Answer{{Question: "q1", Text: text}}
		if !slices.Equal(fake.answers, want) {
			t.Errorf("%s: Copilot got %+v, want %+v", body, fake.answers, want)
		}
	}
}

func TestAnsweringAQuestionTakesAJSONObjectWithAText(t *testing.T) {
	refusal := "the body isn't a JSON object with just a text\n"

	for body, want := range map[string]string{
		`{"text":"Rome","question":"q2"}`: refusal,
		`{"text":["Rome"]}`:               refusal,
		`Rome`:                            refusal,
		`{"text":" \n\t"}`:                "the answer is blank\n",
		`{}`:                              "the answer is blank\n",
	} {
		fake := &standIn{}

		rec := newPlannerAPI(fake).serve(t, http.MethodPost, answerTarget, body)
		if rec.Code != http.StatusBadRequest || rec.Body.String() != want ||
			len(fake.answers) != 0 {
			t.Errorf("%s: got %d %q and Copilot got %+v, want 400 %q and nothing for Copilot",
				body, rec.Code, rec.Body, fake.answers, want)
		}
	}
}

func TestAnsweringAQuestionTakesUpTo64KiB(t *testing.T) {
	text := strings.Repeat("a", 64<<10)

	rec := newPlannerAPI(&standIn{}).serve(t, http.MethodPost, answerTarget,
		`{"text":"`+text+`"}`)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("got %d, want 413", rec.Code)
	}
}

func TestAnsweringAQuestionSaysWhyItCannot(t *testing.T) {
	for _, test := range []struct {
		target  string
		failure error
		code    int
		want    string
	}{
		{"/planner/plans/plan-2/questions/q1/answer", nil, http.StatusNotFound, "no such plan\n"},
		{answerTarget, copilot.ErrNoQuestion, http.StatusNotFound, "no such question\n"},
		{
			answerTarget, copilot.ErrNotAChoice, http.StatusBadRequest,
			"the answer is not one of the choices\n",
		},
	} {
		planner := newPlannerAPI(&standIn{failure: test.failure})

		rec := planner.serve(t, http.MethodPost, test.target, `{"text":"Rome"}`)
		if rec.Code != test.code || rec.Body.String() != test.want || planner.logs.Len() != 0 {
			t.Errorf("%s with %v: got %d %q and logged %q, want %d %q and no log", test.target,
				test.failure, rec.Code, rec.Body, planner.logs, test.code, test.want)
		}
	}
}

func TestAnsweringAQuestionFailsOnAnyOtherError(t *testing.T) {
	planner := newPlannerAPI(&standIn{failure: errors.New("broken")})

	rec := planner.serve(t, http.MethodPost, answerTarget, `{"text":"Rome"}`)
	if rec.Code != http.StatusInternalServerError ||
		rec.Body.String() != "Internal Server Error\n" {
		t.Errorf("got %d %q, want 500 Internal Server Error", rec.Code, rec.Body)
	}

	logs := planner.logs.String()
	if !strings.Contains(logs, `msg="answer a question" err=broken`) {
		t.Errorf("logged %q, want the cause", logs)
	}
}
