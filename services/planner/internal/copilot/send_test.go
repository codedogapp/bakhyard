package copilot

import (
	"errors"
	"strings"
	"testing"
)

func TestSend(t *testing.T) {
	fake := newFakeCLI("", "")
	client := started(t, fake)

	planID, err := client.Plan(t.Context(), "Plan a trip")
	if err != nil {
		t.Fatal(err)
	}

	if err := client.Send(t.Context(), planID, "Make it a week"); err != nil {
		t.Fatal(err)
	}

	var message sent
	fake.asks(t, "session.send", &message)

	if want := (sent{planID, "Make it a week", "plan"}); message != want {
		t.Errorf("session.send = %+v, want %+v", message, want)
	}
}

func TestSendFails(t *testing.T) {
	fake := newFakeCLI("", "")
	client := started(t, fake)

	err := client.Send(t.Context(), "unknown", "Make it a week")
	if !errors.Is(err, ErrNoPlan) || fake.asks(t, "session.send", nil) {
		t.Errorf("Send to an unknown plan = %v, want ErrNoPlan and nothing sent", err)
	}

	planID, err := client.Plan(t.Context(), "Plan a trip")
	if err != nil {
		t.Fatal(err)
	}

	fake.fail("session.send")

	err = client.Send(t.Context(), planID, "Make it a week")
	if err == nil || !strings.HasPrefix(err.Error(), "send the message: ") {
		t.Errorf("Send = %v, want the CLI's error, wrapped", err)
	}
}
