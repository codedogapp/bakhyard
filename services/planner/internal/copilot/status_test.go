package copilot

import (
	"reflect"
	"testing"

	sdk "github.com/github/copilot-sdk/go"
)

// turn returns the event of the start of the agent's turn id.
func turn(id string) sdk.SessionEvent {
	return sdk.SessionEvent{Data: &sdk.AssistantTurnStartData{TurnID: id}}
}

// failed returns the event of an error that stops the agent.
func failed(message string) sdk.SessionEvent {
	return sdk.SessionEvent{Data: &sdk.SessionErrorData{ErrorType: "quota", Message: message}}
}

// idle returns the event of the session gone idle.
func idle() sdk.SessionEvent {
	return sdk.SessionEvent{Data: &sdk.SessionIdleData{}}
}

func TestMind(t *testing.T) {
	planned := newPlan(nil)
	updates, _ := planned.follow()

	for _, step := range []struct {
		event sdk.SessionEvent
		want  Status
	}{
		{turn("0"), Status{Working: true}},
		{delta("m1", "Wh"), Status{Working: true}},
		// A turn of the same loop changes nothing.
		{turn("1"), Status{Working: true}},
		{failed("Quota exceeded"), Status{Failure: "Quota exceeded"}},
		// The failure says why the agent stopped, until it works again.
		{idle(), Status{Failure: "Quota exceeded"}},
		{turn("0"), Status{Working: true}},
		{delta("m2", "So"), Status{Working: true}},
		{idle(), Status{}},
	} {
		planned.see(step.event)

		if got := planned.snapshot().Status; got != step.want {
			t.Errorf("after %T: status = %+v, want %+v", step.event.Data, got, step.want)
		}
	}

	// The agent stopped before it finished m1 and m2.
	if got := planned.snapshot().Writing; len(got) != 0 {
		t.Errorf("Writing = %+v, want none once the agent has stopped", got)
	}

	planned.unfollow(updates)

	want := []Update{
		{Change: ChangeStatus, Status: Status{Working: true}},
		{Change: ChangeDelta, Message: Message{ID: "m1", From: Assistant, Text: "Wh"}},
		{Change: ChangeStatus, Status: Status{Failure: "Quota exceeded"}},
		{Change: ChangeStatus, Status: Status{Working: true}},
		{Change: ChangeDelta, Message: Message{ID: "m2", From: Assistant, Text: "So"}},
		{Change: ChangeStatus, Status: Status{}},
	}
	if got := closed(t, updates); !reflect.DeepEqual(got, want) {
		t.Errorf("updates = %+v, want %+v", got, want)
	}
}

func TestFollowTellsTheStatus(t *testing.T) {
	fake := newFakeCLI("", "")
	client := started(t, fake)

	planID, err := client.Plan(t.Context(), "Plan a trip")
	if err != nil {
		t.Fatal(err)
	}

	talk, updates, err := client.Follow(t.Context(), planID)
	if err != nil {
		t.Fatal(err)
	}

	if talk.Status != (Status{}) {
		t.Errorf("status = %+v, want idle before the agent's turn", talk.Status)
	}

	fake.emit(t, planID, turn("0"), delta("m3", "To "), failed("Rate limited"), idle())

	for _, want := range []Update{
		{Change: ChangeStatus, Status: Status{Working: true}},
		{Change: ChangeDelta, Message: Message{ID: "m3", From: Assistant, Text: "To "}},
		{Change: ChangeStatus, Status: Status{Failure: "Rate limited"}},
	} {
		if got := next(t, updates); !reflect.DeepEqual(got, want) {
			t.Errorf("update = %+v, want %+v", got, want)
		}
	}

	// The idle event tells nothing, so the start of the next turn, which does, shows that the
	// client has taken in every event before the second Follow.
	fake.emit(t, planID, turn("1"))
	next(t, updates)

	talk, _, err = client.Follow(t.Context(), planID)
	if err != nil {
		t.Fatal(err)
	}

	if talk.Status != (Status{Working: true}) || len(talk.Writing) != 0 {
		t.Errorf("Follow = %+v, want the agent at work, with nothing half-written", talk)
	}
}
