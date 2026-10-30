package copilot

import (
	"errors"
	"reflect"
	"testing"

	sdk "github.com/github/copilot-sdk/go"
)

func TestConversation(t *testing.T) {
	fake := newFakeCLI("# Plan\n", "")
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

	want := []Message{{From: User, Text: "Plan a trip"}, {From: Assistant, Text: "Where to?"}}
	if !reflect.DeepEqual(talk.Messages, want) {
		t.Errorf("Messages = %+v, want %+v", talk.Messages, want)
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
		{Data: &sdk.UserMessageData{Content: "Plan a trip"}},
		{Data: &sdk.UserMessageData{Content: "And a hotel", Source: new("user")}},
		{Data: &sdk.UserMessageData{Content: "Mind the plan", Source: new("system")}},
		{Data: &sdk.UserMessageData{Content: "Read the PDF", Source: new("skill-pdf")}},
		{Data: &sdk.UserMessageData{Content: "Look it up", Source: new("agent-1")}},
		{Data: &sdk.AssistantMessageData{MessageID: "m1"}},
		{Data: &sdk.AssistantMessageData{MessageID: "m2", Content: "Where to?"}},
		{Data: &sdk.SessionIdleData{}},
	}

	want := []Message{
		{From: User, Text: "Plan a trip"},
		{From: User, Text: "And a hotel"},
		{From: Assistant, Text: "Where to?"},
	}
	if got := messages(events); !reflect.DeepEqual(got, want) {
		t.Errorf("messages = %+v, want %+v", got, want)
	}
}
