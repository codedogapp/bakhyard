package copilot

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"reflect"
	"testing"
	"uuid"

	sdk "github.com/github/copilot-sdk/go"
)

// userInput is the request in which the CLI brings the agent's question.
type userInput struct {
	SessionID     string   `json:"sessionId"`
	Question      string   `json:"question"`
	Choices       []string `json:"choices,omitzero"`
	AllowFreeform *bool    `json:"allowFreeform,omitzero"`
}

// reply is the client's answer to the agent's question, as the CLI gets it.
type reply struct {
	Answer      string `json:"answer"`
	WasFreeform bool   `json:"wasFreeform"`
}

// question returns the request that brings the client of session id the agent's question, which
// takes only one of choices.
func question(id, text string, choices ...string) outgoing {
	request := userInput{SessionID: id, Question: text, Choices: choices, AllowFreeform: new(false)}

	return outgoing{ID: jsontext.Value("2"), Method: "userInput.request", Params: request}
}

// planning starts a plan on client, follows it, and returns its ID and its updates.
func planning(t *testing.T, client *Client) (string, <-chan Update) {
	t.Helper()

	planID, err := client.Plan(t.Context(), "Plan a trip")
	if err != nil {
		t.Fatal(err)
	}

	_, updates, err := client.Follow(t.Context(), planID)
	if err != nil {
		t.Fatal(err)
	}

	return planID, updates
}

func TestQuestion(t *testing.T) {
	fake := newFakeCLI("", "")
	client := started(t, fake)
	planID, updates := planning(t, client)

	if err := fake.send(question(planID, "Which city?", "Rome", "Paris")); err != nil {
		t.Fatal(err)
	}

	update := next(t, updates)

	asked := Question{
		ID:      update.Question.ID,
		Text:    "Which city?",
		Choices: []string{"Rome", "Paris"},
	}
	if update.Change != ChangeQuestion || !reflect.DeepEqual(update.Question, asked) {
		t.Errorf("update = %+v, want the question %+v", update, asked)
	}

	talk, err := client.Conversation(t.Context(), planID)
	if err != nil || !reflect.DeepEqual(talk.Questions, []Question{asked}) {
		t.Errorf("Conversation = %+v, %v, want the question waiting", talk, err)
	}

	if err := client.Answer(planID, Answer{Question: asked.ID, Text: "Rome"}); err != nil {
		t.Fatal(err)
	}

	var got reply
	if err := json.Unmarshal(fake.replied(t), &got); err != nil || got != (reply{Answer: "Rome"}) {
		t.Errorf("the CLI got %+v, %v, want Rome, one of the choices", got, err)
	}
}

func TestAnswerForAnUnknownPlan(t *testing.T) {
	client := started(t, newFakeCLI("", ""))

	err := client.Answer("unknown", Answer{Question: "q1", Text: "Rome"})
	if !errors.Is(err, ErrNoPlan) {
		t.Errorf("Answer = %v, want ErrNoPlan", err)
	}
}

func TestStopEndsTheQuestions(t *testing.T) {
	fake := newFakeCLI("", "")
	client := started(t, fake)
	planID, updates := planning(t, client)

	if err := fake.send(question(planID, "Which city?")); err != nil {
		t.Fatal(err)
	}

	next(t, updates)

	if err := client.Stop(); err != nil {
		t.Fatal(err)
	}

	if got := fake.replied(t); got != nil {
		t.Errorf("the CLI got %s, want an error", got)
	}
}

func TestQuestionOf(t *testing.T) {
	cities := []string{"Rome", "Paris"}

	for _, test := range []struct {
		request  sdk.UserInputRequest
		freeform bool
	}{
		{sdk.UserInputRequest{Question: "Which city?", Choices: cities}, true},
		{sdk.UserInputRequest{Question: "Which?", Choices: cities, AllowFreeform: new(true)}, true},
		{sdk.UserInputRequest{Question: "Which?", Choices: cities, AllowFreeform: new(false)}, false},
		{sdk.UserInputRequest{Question: "When?", AllowFreeform: new(false)}, true},
	} {
		got := questionOf(test.request)
		want := Question{
			ID: got.ID, Text: test.request.Question, Choices: test.request.Choices,
			Freeform: test.freeform,
		}

		if _, err := uuid.Parse(got.ID); err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("questionOf(%+v) = %+v, want %+v with a UUID", test.request, got, want)
		}
	}

	request := sdk.UserInputRequest{Question: "When?"}
	if first, second := questionOf(request), questionOf(request); first.ID == second.ID {
		t.Error("two questions have the same ID")
	}
}
