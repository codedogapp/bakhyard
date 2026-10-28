package agent_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/services/nudge/internal/notifier"
	"github.com/codedogapp/bakhyard/services/nudge/internal/question"
)

func TestAnAnswerGoesBackToTheAgentAndTheUserGetsOnePush(t *testing.T) {
	gate := newDoor(t)
	done := gate.send(t.Context(), http.MethodPost, "/agent/questions", merge)

	want := notifier.Message{Title: "atlas", Body: "Merge?", URL: "/nudge"}
	if got := <-gate.pushes; got != want {
		t.Errorf("pushed %+v, want %+v", got, want)
	}

	// The same agent asks again, as after a broken connection: it attaches, with no second push.
	again := gate.send(t.Context(), http.MethodPost, "/agent/questions", merge)

	pending, err := gate.questions.Pending(t.Context())
	if err != nil || len(pending) != 1 || pending[0].ID != id1 || pending[0].Project != "atlas" {
		t.Fatalf("pending: %+v, %v", pending, err)
	}

	err = gate.questions.Reply(t.Context(), id1, question.Answer{Option: "Yes", Text: "squash"})
	if err != nil {
		t.Fatal(err)
	}

	for _, got := range []reply{<-done, <-again} {
		if got.status != http.StatusOK || got.body != `{"option":"Yes","text":"squash"}` {
			t.Errorf("reply %d %s", got.status, got.body)
		}
	}

	gate.noMorePushes(t)
}

func TestAFailedPushChangesNothingForTheAgent(t *testing.T) {
	gate := newDoor(t)
	gate.pusherFails(errors.New("notify is down"))

	done := gate.send(t.Context(), http.MethodPost, "/agent/questions", merge)
	<-gate.pushes

	if err := gate.questions.Reply(t.Context(), id1, question.Answer{Text: "ok"}); err != nil {
		t.Fatal(err)
	}

	if got := <-done; got.status != http.StatusOK {
		t.Errorf("reply %d %s", got.status, got.body)
	}
}

func TestABadQuestionGetsA400(t *testing.T) {
	tests := map[string]string{
		"not JSON":       `nope`,
		"unknown member": `{"id":"` + id1 + `","text":"x","extra":1}`,
		"no id":          `{"text":"x"}`,
		"no text":        `{"id":"` + id1 + `","text":" "}`,
		"bad options":    `{"id":"` + id1 + `","text":"x","options":["a","a"]}`,
	}

	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			gate := newDoor(t)

			if got := gate.call(
				t,
				http.MethodPost,
				"/agent/questions",
				body,
			); got.status != http.StatusBadRequest {
				t.Errorf("reply %d %s, want 400", got.status, got.body)
			}
		})
	}
}

func TestATooLargeBodyGetsA413(t *testing.T) {
	gate := newDoor(t)
	body := `{"id":"` + id1 + `","text":"` + strings.Repeat("a", 70000) + `"}`

	if got := gate.call(
		t,
		http.MethodPost,
		"/agent/questions",
		body,
	); got.status != http.StatusRequestEntityTooLarge {
		t.Errorf("reply %d, want 413", got.status)
	}
}

func TestWithdrawingAQuestionEndsItsRequestWith410(t *testing.T) {
	gate := newDoor(t)
	done := gate.send(t.Context(), http.MethodPost, "/agent/questions", merge)
	<-gate.pushes

	for range 2 { // the second finds nothing, and that is fine too
		if got := gate.call(
			t,
			http.MethodDelete,
			"/agent/questions/"+id1,
			"",
		); got.status != http.StatusNoContent {
			t.Errorf("withdraw: reply %d %s", got.status, got.body)
		}
	}

	if got := <-done; got.status != http.StatusGone {
		t.Errorf("reply %d %s, want 410", got.status, got.body)
	}
}

func TestStoppingTheServiceEndsHeldRequestsWith503(t *testing.T) {
	gate := newDoor(t)
	done := gate.send(t.Context(), http.MethodPost, "/agent/questions", merge)
	<-gate.pushes
	gate.stop()

	if got := <-done; got.status != http.StatusServiceUnavailable {
		t.Errorf("reply %d %s, want 503", got.status, got.body)
	}

	// The question stays, so the agent finds it again when the service is back.
	if pending, _ := gate.questions.Pending(t.Context()); len(pending) != 1 {
		t.Errorf("pending: %+v", pending)
	}
}
