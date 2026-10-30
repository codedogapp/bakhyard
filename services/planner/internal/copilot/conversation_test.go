package copilot

import (
	"errors"
	"reflect"
	"testing"

	sdk "github.com/github/copilot-sdk/go"
)

func TestConversation(t *testing.T) {
	fake := newFakeCLI("", "")
	client := started(t, fake)

	planID, err := client.Plan(t.Context(), "Plan a trip")
	if err != nil {
		t.Fatal(err)
	}

	talk, err := client.Conversation(t.Context(), planID)
	if err != nil {
		t.Fatal(err)
	}

	var asked session
	if !fake.asks(t, "session.getMessages", &asked) || asked.SessionID != planID {
		t.Errorf("session.getMessages for %+v, want for session %s", asked, planID)
	}

	want := Conversation{Messages: []Message{
		{ID: "1", From: User, Text: "Plan a trip"},
		{ID: "m2", From: Assistant, Text: "Where to?"},
	}}
	if !reflect.DeepEqual(talk, want) {
		t.Errorf("Conversation = %+v, want %+v", talk, want)
	}
}

func TestConversationFails(t *testing.T) {
	client := started(t, newFakeCLI("# Plan\n", "session.getMessages"))

	if _, err := client.Conversation(t.Context(), "unknown"); !errors.Is(err, ErrNoPlan) {
		t.Errorf("Conversation of an unknown plan = %v, want ErrNoPlan", err)
	}

	planID, err := client.Plan(t.Context(), "Plan a trip")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := client.Conversation(t.Context(), planID); err == nil || errors.Is(err, ErrNoPlan) {
		t.Errorf("Conversation = %v, want the CLI's error", err)
	}
}

func TestMessages(t *testing.T) {
	events := []sdk.SessionEvent{
		{ID: "1", Data: &sdk.UserMessageData{Content: "Plan a trip"}},
		{ID: "2", Data: &sdk.UserMessageData{Content: "And a hotel", Source: new("user")}},
		{ID: "3", Data: &sdk.UserMessageData{Content: "Mind the plan", Source: new("system")}},
		{ID: "4", Data: &sdk.UserMessageData{Content: "Read the PDF", Source: new("skill-pdf")}},
		{ID: "5", Data: &sdk.UserMessageData{Content: "Look it up", Source: new("agent-1")}},
		{ID: "6", Data: &sdk.AssistantMessageData{MessageID: "m1"}},
		{ID: "7", Data: &sdk.AssistantMessageData{MessageID: "m2", Content: "Where to?"}},
		{ID: "8", Data: &sdk.AssistantMessageDeltaData{MessageID: "m3", DeltaContent: "So"}},
		{ID: "9", Data: &sdk.SessionIdleData{}},
	}

	want := []Message{
		{ID: "1", From: User, Text: "Plan a trip"},
		{ID: "2", From: User, Text: "And a hotel"},
		{ID: "m2", From: Assistant, Text: "Where to?"},
	}
	if got := messages(events); !reflect.DeepEqual(got, want) {
		t.Errorf("messages = %+v, want %+v", got, want)
	}
}

func TestRepeats(t *testing.T) {
	talk := Conversation{
		Messages: []Message{{ID: "m1", From: Assistant, Text: "Where to?"}},
		Writing:  []Message{{ID: "m2", From: Assistant, Text: "So"}},
	}

	for _, test := range []struct {
		update  Update
		repeats bool
	}{
		{Update{Change: ChangeDelta, Message: Message{ID: "m1", Text: " to?"}}, true},
		{Update{Change: ChangeMessage, Message: Message{ID: "m1", Text: "Where to?"}}, true},
		{Update{Change: ChangeDelta, Message: Message{ID: "m2", Text: " Rome"}}, false},
		{Update{Change: ChangeMessage, Message: Message{ID: "m2", Text: "So Rome"}}, false},
		{Update{Change: ChangePlan, Plan: "# Plan\n"}, false},
		{Update{Change: ChangeQuestion, Question: Question{ID: "q1", Text: "Which city?"}}, false},
		{Update{Change: ChangeAnswered, Question: Question{ID: "q1", Text: "Which city?"}}, false},
	} {
		if got := talk.Repeats(test.update); got != test.repeats {
			t.Errorf("Repeats(%+v) = %t, want %t", test.update, got, test.repeats)
		}
	}
}
