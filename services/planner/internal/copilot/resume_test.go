package copilot

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	sdk "github.com/github/copilot-sdk/go"
)

// resumed is what a client asks for when it resumes a plan's session.
type resumed struct {
	created

	ContinuePendingWork *bool `json:"continuePendingWork"`
}

// workspace returns a session's workspace, with plan in its plan file.
func workspace(t *testing.T, plan string) string {
	t.Helper()

	folder := t.TempDir()
	if err := os.WriteFile(filepath.Join(folder, "plan.md"), []byte(plan), 0o600); err != nil {
		t.Fatal(err)
	}

	return folder
}

func TestResume(t *testing.T) {
	fake := newFakeCLI("", "")
	fake.workspace = workspace(t, "# Plan\n\n1. Pack.\n")
	client := started(t, fake)

	talk, err := client.Conversation(t.Context(), earlier)
	if err != nil {
		t.Fatal(err)
	}

	if len(talk.Messages) != 4 || talk.Plan != "# Plan\n\n1. Pack.\n" {
		t.Errorf("Conversation = %+v, want the record's messages and the plan file's plan", talk)
	}

	var session resumed
	fake.asks(t, "session.resume", &session)

	want := resumed{
		SessionID: earlier, Model: "claude-opus-5.5",
		AvailableTools: []string{
			"ask_user", "view", "glob", "grep", "create", "edit", "exit_plan_mode",
		},
		WorkingDirectory: "/work", SkipCustomInstructions: true, Streaming: true,
		RequestPermission: true, RequestUserInput: true, RequestExitPlanMode: true,
		ContinuePendingWork: new(false),
	}
	if !reflect.DeepEqual(session, want) {
		t.Errorf("session.resume = %+v, want %+v", session, want)
	}
}

func TestResumedPlan(t *testing.T) {
	fake := newFakeCLI("# Plan\n\n1. Pack light.\n", "")
	fake.workspace = workspace(t, "# Plan\n\n1. Pack.\n")
	client := started(t, fake)

	if err := client.Send(t.Context(), earlier, "Pack light"); err != nil {
		t.Fatal(err)
	}

	var message sent
	fake.asks(t, "session.send", &message)

	if want := (sent{earlier, "Pack light", "plan"}); message != want {
		t.Errorf("session.send = %+v, want %+v", message, want)
	}

	exitOnly := sdk.ExitPlanModeResult{Approved: true, SelectedAction: "exit_only"}
	if got := answer(t, fake); got != exitOnly {
		t.Errorf("answer = %+v, want %+v", got, exitOnly)
	}

	talk, err := client.Conversation(t.Context(), earlier)
	if err != nil || talk.Plan != "# Plan\n\n1. Pack light.\n" {
		t.Errorf("Conversation = %+v, %v, want the plan that exit_plan_mode brought", talk, err)
	}

	if times := fake.times("session.resume"); times != 1 {
		t.Errorf("resumed the session %d times, want once", times)
	}
}

func TestFollowResumedPlan(t *testing.T) {
	fake := newFakeCLI("", "")
	fake.workspace = workspace(t, "# Plan\n")

	talk, updates, err := started(t, fake).Follow(t.Context(), earlier)
	if err != nil || talk.Plan != "# Plan\n" {
		t.Fatalf("Follow = %+v, %v, want the plan file's plan", talk, err)
	}

	fake.emit(t, earlier, said("m3", "To Rome."))

	want := Update{
		Change:  ChangeMessage,
		Message: Message{ID: "m3", From: Assistant, Text: "To Rome."},
	}
	if got := next(t, updates); !reflect.DeepEqual(got, want) {
		t.Errorf("update = %+v, want %+v", got, want)
	}
}

func TestResumeAsAsked(t *testing.T) {
	tests := map[string]struct {
		ask  func(context.Context, *Client) error
		want error
	}{
		"Conversation": {ask: func(ctx context.Context, client *Client) error {
			_, err := client.Conversation(ctx, earlier)

			return err
		}},
		"Follow": {ask: func(ctx context.Context, client *Client) error {
			_, _, err := client.Follow(ctx, earlier)

			return err
		}},
		"Answer": {ask: func(ctx context.Context, client *Client) error {
			return client.Answer(ctx, earlier, Answer{Question: "q1", Text: "Rome"})
		}, want: ErrNoQuestion},
		"Send": {ask: func(ctx context.Context, client *Client) error {
			return client.Send(ctx, earlier, "Pack light")
		}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			fake := newFakeCLI("", "")
			fake.workspace = t.TempDir()

			if err := test.ask(t.Context(), started(t, fake)); !errors.Is(err, test.want) {
				t.Errorf("%s = %v, want %v", name, err, test.want)
			}

			if !fake.asks(t, "session.resume", nil) {
				t.Errorf("%s didn't resume the plan", name)
			}
		})
	}
}

func TestResumeWithoutAPlan(t *testing.T) {
	for name, folder := range map[string]string{"no plan file": t.TempDir(), "no workspace": ""} {
		t.Run(name, func(t *testing.T) {
			fake := newFakeCLI("", "")
			fake.workspace = folder

			talk, err := started(t, fake).Conversation(t.Context(), earlier)
			if err != nil || talk.Plan != "" {
				t.Errorf("Conversation = %+v, %v, want no plan", talk, err)
			}
		})
	}
}

func TestResumeFails(t *testing.T) {
	folder := t.TempDir()
	if err := os.Mkdir(filepath.Join(folder, "plan.md"), 0o700); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name, id, failing, workspace, want string
		left                               bool
	}{
		{name: "unknown", id: "unknown", want: "no such plan"},
		{name: "metadata", id: earlier, failing: "session.getMetadata", want: "look the plan up: "},
		{name: "resume", id: earlier, failing: "session.resume", want: "resume the session: "},
		{
			name: "workspace", id: earlier, workspace: filepath.Join(folder, "gone"),
			want: "open the session's workspace: ", left: true,
		},
		{name: "plan file", id: earlier, workspace: folder, want: "read the plan: ", left: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fake := newFakeCLI("", test.failing)
			fake.workspace = test.workspace
			client := started(t, fake)

			_, err := client.Conversation(t.Context(), test.id)
			if err == nil || !strings.HasPrefix(err.Error(), test.want) {
				t.Errorf("Conversation = %v, want %q", err, test.want)
			}

			if left := fake.asks(t, "session.detach", nil); left != test.left {
				t.Errorf("left the session: %t, want %t", left, test.left)
			}

			_, _ = client.Conversation(t.Context(), test.id)

			if times := fake.times("session.getMetadata"); times != 2 {
				t.Errorf("looked the plan up %d times, want again as it's asked for again", times)
			}
		})
	}
}

func TestResumeOnce(t *testing.T) {
	fake := newFakeCLI("", "")
	fake.workspace = t.TempDir()
	client := started(t, fake)

	var requests sync.WaitGroup
	for range 8 {
		requests.Go(func() {
			if _, err := client.Conversation(t.Context(), earlier); err != nil {
				t.Error(err)
			}
		})
	}

	requests.Wait()

	if times := fake.times("session.resume"); times != 1 {
		t.Errorf("resumed the session %d times, want once", times)
	}
}
