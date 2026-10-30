package copilot

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	sdk "github.com/github/copilot-sdk/go"
)

// wait bounds how long a test waits for an update.
const wait = 10 * time.Second

// delta returns the event of a piece of the text of the assistant's message id.
func delta(id, text string) sdk.SessionEvent {
	return sdk.SessionEvent{Data: &sdk.AssistantMessageDeltaData{MessageID: id, DeltaContent: text}}
}

// said returns the event of the assistant's whole message id.
func said(id, text string) sdk.SessionEvent {
	return sdk.SessionEvent{Data: &sdk.AssistantMessageData{MessageID: id, Content: text}}
}

// next returns the next of updates, or fails when none comes.
func next(t *testing.T, updates <-chan Update) Update {
	t.Helper()

	select {
	case update, open := <-updates:
		if !open {
			t.Fatal("the updates closed")
		}

		return update
	case <-time.After(wait):
		t.Fatal("no update came")
	}

	return Update{}
}

// closed returns what is left of updates once they close, or fails when they don't.
func closed(t *testing.T, updates <-chan Update) []Update {
	t.Helper()

	var rest []Update

	timeout := time.After(wait)

	for {
		select {
		case update, open := <-updates:
			if !open {
				return rest
			}

			rest = append(rest, update)
		case <-timeout:
			t.Fatal("the updates didn't close")
		}
	}
}

func TestFollow(t *testing.T) {
	fake := newFakeCLI("", "")
	client := started(t, fake)

	planID, err := client.Plan(t.Context(), "Plan a trip")
	if err != nil {
		t.Fatal(err)
	}

	ctx, leave := context.WithCancel(t.Context())
	defer leave()

	talk, updates, err := client.Follow(ctx, planID)
	if err != nil {
		t.Fatal(err)
	}

	want := Conversation{Messages: []Message{
		{ID: "1", From: User, Text: "Plan a trip"},
		{ID: "m2", From: Assistant, Text: "Where to?"},
		{ID: "5", From: Assistant, Text: "Which city?"},
		{ID: "6", From: User, Text: "Rome"},
	}}
	if !reflect.DeepEqual(talk, want) {
		t.Errorf("Follow = %+v, want %+v", talk, want)
	}

	fake.emit(t, planID, delta("m3", "To "), delta("m3", "Rome."), said("m3", "To Rome."))

	for _, want := range []Update{
		{Change: ChangeDelta, Message: Message{ID: "m3", From: Assistant, Text: "To "}},
		{Change: ChangeDelta, Message: Message{ID: "m3", From: Assistant, Text: "Rome."}},
		{Change: ChangeMessage, Message: Message{ID: "m3", From: Assistant, Text: "To Rome."}},
	} {
		if got := next(t, updates); !reflect.DeepEqual(got, want) {
			t.Errorf("update = %+v, want %+v", got, want)
		}
	}

	if err := fake.send(bring(planID, "# Plan\n")); err != nil {
		t.Fatal(err)
	}

	brought := Update{Change: ChangePlan, Plan: "# Plan\n"}
	if got := next(t, updates); !reflect.DeepEqual(got, brought) {
		t.Errorf("update = %+v, want %+v", got, brought)
	}

	leave()

	if rest := closed(t, updates); len(rest) != 0 {
		t.Errorf("updates after leaving: %+v", rest)
	}
}

func TestFollowJoinsMidMessage(t *testing.T) {
	fake := newFakeCLI("", "")
	client := started(t, fake)

	planID, err := client.Plan(t.Context(), "Plan a trip")
	if err != nil {
		t.Fatal(err)
	}

	_, first, err := client.Follow(t.Context(), planID)
	if err != nil {
		t.Fatal(err)
	}

	// The CLI's record has m2 whole, as when it keeps a message while Follow reads the record.
	fake.emit(t, planID, delta("m2", "Where"), delta("m3", "Let"))
	next(t, first)
	next(t, first)

	talk, _, err := client.Follow(t.Context(), planID)
	if err != nil {
		t.Fatal(err)
	}

	want := []Message{{ID: "m3", From: Assistant, Text: "Let"}}
	if !reflect.DeepEqual(talk.Writing, want) {
		t.Errorf("Writing = %+v, want %+v", talk.Writing, want)
	}
}

func TestFollowFails(t *testing.T) {
	client := started(t, newFakeCLI("", "session.getMessages"))

	if _, _, err := client.Follow(t.Context(), "unknown"); !errors.Is(err, ErrNoPlan) {
		t.Errorf("Follow of an unknown plan = %v, want ErrNoPlan", err)
	}

	planID, err := client.Plan(t.Context(), "Plan a trip")
	if err != nil {
		t.Fatal(err)
	}

	_, updates, err := client.Follow(t.Context(), planID)
	if err == nil || errors.Is(err, ErrNoPlan) || updates != nil {
		t.Errorf("Follow = %v, %v, want the CLI's error", updates, err)
	}

	planned, err := client.find(planID)
	if err != nil {
		t.Fatal(err)
	}

	planned.mu.Lock()
	defer planned.mu.Unlock()

	if len(planned.followers) != 0 {
		t.Errorf("%d followers, want none after a failed Follow", len(planned.followers))
	}
}

func TestSee(t *testing.T) {
	planned := newPlan(nil)
	updates, _ := planned.follow()

	for _, event := range []sdk.SessionEvent{
		delta("m1", "Wh"),
		delta("m1", ""),
		delta("m2", "So"),
		delta("m1", "ere"),
		{ID: "e1", Data: &sdk.UserMessageData{Content: "Mind the plan", Source: new("system")}},
		{ID: "e2", Data: &sdk.SessionIdleData{}},
	} {
		planned.see(event)
	}

	writing := []Message{
		{ID: "m1", From: Assistant, Text: "Where"},
		{ID: "m2", From: Assistant, Text: "So"},
	}
	if got := planned.snapshot().Writing; !reflect.DeepEqual(got, writing) {
		t.Errorf("Writing = %+v, want %+v", got, writing)
	}

	planned.see(said("m1", "Where"))
	// A message that only calls tools has no text.
	planned.see(said("m2", ""))
	planned.see(sdk.SessionEvent{ID: "e3", Data: &sdk.UserMessageData{Content: "Rome"}})

	if got := planned.snapshot().Writing; len(got) != 0 {
		t.Errorf("Writing = %+v, want none once the messages are whole", got)
	}

	planned.unfollow(updates)

	want := []Update{
		{Change: ChangeDelta, Message: Message{ID: "m1", From: Assistant, Text: "Wh"}},
		{Change: ChangeDelta, Message: Message{ID: "m2", From: Assistant, Text: "So"}},
		{Change: ChangeDelta, Message: Message{ID: "m1", From: Assistant, Text: "ere"}},
		{Change: ChangeMessage, Message: Message{ID: "m1", From: Assistant, Text: "Where"}},
		{Change: ChangeMessage, Message: Message{ID: "e3", From: User, Text: "Rome"}},
	}
	if got := closed(t, updates); !reflect.DeepEqual(got, want) {
		t.Errorf("updates = %+v, want %+v", got, want)
	}
}

func TestFollowerFallsBehind(t *testing.T) {
	planned := newPlan(nil)
	slow, _ := planned.follow()

	for range backlog + 1 {
		planned.see(delta("m1", "."))
	}

	if got := closed(t, slow); len(got) != backlog {
		t.Errorf("the follower got %d updates, want %d before it's dropped", len(got), backlog)
	}

	// Neither leaving nor another update touches the dropped follower's closed channel.
	planned.unfollow(slow)
	planned.see(delta("m1", "."))
}
