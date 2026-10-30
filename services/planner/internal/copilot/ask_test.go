package copilot

import (
	"errors"
	"reflect"
	"testing"
	"time"

	sdk "github.com/github/copilot-sdk/go"
)

// response is what a plan's handler of the agent's questions returns.
type response struct {
	answer sdk.UserInputResponse
	err    error
}

// pose calls planned's handler with request, in a goroutine as the SDK does, and returns the
// question that the follower of updates hears of, with the handler's response to come.
func pose(
	t *testing.T,
	planned *plan,
	updates <-chan Update,
	request sdk.UserInputRequest,
) (Question, <-chan response) {
	t.Helper()

	responses := make(chan response, 1)

	go func() {
		answer, err := planned.ask(request, sdk.UserInputInvocation{SessionID: "session-1"})
		responses <- response{answer, err}
	}()

	update := next(t, updates)
	if update.Change != ChangeQuestion {
		t.Fatalf("update = %+v, want a question", update)
	}

	return update.Question, responses
}

// returned returns the handler's response, or fails when it doesn't return.
func returned(t *testing.T, responses <-chan response) response {
	t.Helper()

	select {
	case got := <-responses:
		return got
	case <-time.After(wait):
		t.Fatal("the handler didn't return")
	}

	return response{}
}

func TestAsk(t *testing.T) {
	cities := sdk.UserInputRequest{Question: "Which city?", Choices: []string{"Rome", "Paris"}}

	for _, test := range []struct {
		name    string
		request sdk.UserInputRequest
		answer  string
		want    sdk.UserInputResponse
	}{
		{"a choice", cities, "Paris", sdk.UserInputResponse{Answer: "Paris"}},
		{"another answer", cities, "Oslo", sdk.UserInputResponse{Answer: "Oslo", WasFreeform: true}},
		{
			"no choices",
			sdk.UserInputRequest{Question: "When?"},
			"May",
			sdk.UserInputResponse{Answer: "May", WasFreeform: true},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			planned := newPlan(nil)
			updates, _ := planned.follow()
			asked, responses := pose(t, planned, updates, test.request)

			if err := planned.answer(Answer{Question: asked.ID, Text: test.answer}); err != nil {
				t.Fatal(err)
			}

			if got := returned(t, responses); got.answer != test.want || got.err != nil {
				t.Errorf("ask = %+v, %v, want %+v", got.answer, got.err, test.want)
			}

			answered := Update{Change: ChangeAnswered, Question: asked}
			if got := next(t, updates); !reflect.DeepEqual(got, answered) {
				t.Errorf("update = %+v, want %+v", got, answered)
			}

			if waiting := planned.snapshot().Questions; len(waiting) != 0 {
				t.Errorf("Questions = %+v, want none once answered", waiting)
			}
		})
	}
}

func TestAnswerFails(t *testing.T) {
	planned := newPlan(nil)
	updates, _ := planned.follow()
	request := sdk.UserInputRequest{
		Question: "Which city?", Choices: []string{"Rome"}, AllowFreeform: new(false),
	}
	asked, responses := pose(t, planned, updates, request)

	for _, test := range []struct {
		answer Answer
		want   error
	}{
		{Answer{Question: "unknown", Text: "Rome"}, ErrNoQuestion},
		{Answer{Question: asked.ID, Text: "Paris"}, ErrNotAChoice},
		{Answer{Question: asked.ID, Text: "rome"}, ErrNotAChoice},
		{Answer{Question: asked.ID, Text: "Rome"}, nil},
		{Answer{Question: asked.ID, Text: "Rome"}, ErrNoQuestion},
	} {
		if err := planned.answer(test.answer); !errors.Is(err, test.want) {
			t.Errorf("answer(%+v) = %v, want %v", test.answer, err, test.want)
		}
	}

	if got := returned(t, responses); got.answer.Answer != "Rome" || got.err != nil {
		t.Errorf("ask = %+v, %v, want Rome", got.answer, got.err)
	}

	planned.unfollow(updates)

	answered := []Update{{Change: ChangeAnswered, Question: asked}}
	if got := closed(t, updates); !reflect.DeepEqual(got, answered) {
		t.Errorf("updates = %+v, want %+v", got, answered)
	}
}

func TestAskEndsWhenTheClientStops(t *testing.T) {
	stopped := make(chan struct{})
	planned := newPlan(stopped)
	updates, _ := planned.follow()
	_, responses := pose(t, planned, updates, sdk.UserInputRequest{Question: "Which city?"})

	close(stopped)

	if got := returned(t, responses); !errors.Is(got.err, errStopped) {
		t.Errorf("ask = %+v, %v, want errStopped", got.answer, got.err)
	}
}

func TestQuestionsWaitInOrder(t *testing.T) {
	planned := newPlan(nil)
	want := []Question{
		{ID: "q1", Text: "Which city?", Choices: []string{"Rome"}},
		{ID: "q2", Text: "When?", Freeform: true},
	}
	first := planned.wait(want[0])
	planned.wait(want[1])

	if got := planned.snapshot().Questions; !reflect.DeepEqual(got, want) {
		t.Errorf("Questions = %+v, want %+v", got, want)
	}

	if err := planned.answer(Answer{Question: "q1", Text: "Rome"}); err != nil {
		t.Fatal(err)
	}

	if got := planned.snapshot().Questions; !reflect.DeepEqual(got, want[1:]) {
		t.Errorf("Questions = %+v, want %+v once q1 has its answer", got, want[1:])
	}

	if got := <-first.answers; got != "Rome" {
		t.Errorf("q1's answer = %q, want Rome", got)
	}
}
