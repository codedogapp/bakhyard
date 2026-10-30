package copilot

import (
	"encoding/json/v2"
	"errors"
	"reflect"
	"testing"
	"time"

	sdk "github.com/github/copilot-sdk/go"
	"github.com/github/copilot-sdk/go/rpc"
)

// created is what a client asks for when it creates a plan's session.
type created struct {
	SessionID              string   `json:"sessionId"`
	Model                  string   `json:"model"`
	AvailableTools         []string `json:"availableTools"`
	WorkingDirectory       string   `json:"workingDirectory"`
	SkipCustomInstructions bool     `json:"skipCustomInstructions"`
	RequestPermission      bool     `json:"requestPermission"`
	RequestExitPlanMode    bool     `json:"requestExitPlanMode"`
}

// sent is what a client sends with a plan's prompt.
type sent struct {
	SessionID string `json:"sessionId"`
	Prompt    string `json:"prompt"`
	AgentMode string `json:"agentMode"`
}

// started returns a started client of fake, which works in /work.
func started(t *testing.T, fake *fakeCLI) *Client {
	t.Helper()

	client := newClient(
		&sdk.ClientOptions{Connection: sdk.URIConnection{URL: fake.serve(t)}},
		"/work",
	)
	if err := client.Start(t.Context()); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := client.Stop(); err != nil {
			t.Error(err)
		}
	})

	return client
}

// answer returns the client's answer to the request that brings the plan.
func answer(t *testing.T, fake *fakeCLI) sdk.ExitPlanModeResult {
	t.Helper()

	var result sdk.ExitPlanModeResult

	select {
	case got := <-fake.answers:
		if err := json.Unmarshal(got, &result); err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("no answer to exitPlanMode.request")
	}

	return result
}

func TestPlan(t *testing.T) {
	fake := newFakeCLI("# Plan\n\n1. Pack.\n", "")
	client := started(t, fake)

	planID, err := client.Plan(t.Context(), "Plan a trip")
	if err != nil {
		t.Fatal(err)
	}

	var session created
	fake.asks(t, "session.create", &session)

	want := created{
		SessionID: planID, Model: "claude-opus-5.5",
		AvailableTools: []string{
			"ask_user", "view", "glob", "grep", "create", "edit", "exit_plan_mode",
		},
		WorkingDirectory: "/work", SkipCustomInstructions: true,
		RequestPermission: true, RequestExitPlanMode: true,
	}
	if planID == "" || !reflect.DeepEqual(session, want) {
		t.Errorf("session.create = %+v, want %+v", session, want)
	}

	var prompt sent
	fake.asks(t, "session.send", &prompt)

	if want := (sent{planID, "Plan a trip", "plan"}); prompt != want {
		t.Errorf("session.send = %+v, want %+v", prompt, want)
	}

	exitOnly := sdk.ExitPlanModeResult{Approved: true, SelectedAction: "exit_only"}
	if got := answer(t, fake); got != exitOnly {
		t.Errorf("answer = %+v, want %+v", got, exitOnly)
	}

	talk, err := client.Conversation(t.Context(), planID)
	if err != nil || talk.Plan != "# Plan\n\n1. Pack.\n" {
		t.Errorf("Conversation = %+v, %v, want the plan that exit_plan_mode brought", talk, err)
	}
}

func TestPlanFails(t *testing.T) {
	tests := []struct {
		failing string
		left    bool
	}{
		{failing: "session.create"},
		{failing: "session.send", left: true},
	}

	for _, test := range tests {
		t.Run(test.failing, func(t *testing.T) {
			fake := newFakeCLI("", test.failing)
			client := started(t, fake)

			if planID, err := client.Plan(t.Context(), "Plan a trip"); err == nil || planID != "" {
				t.Fatalf("Plan = %q, %v, want an error", planID, err)
			}

			if left := fake.asks(t, "session.detach", nil); left != test.left {
				t.Errorf("left the session: %t, want %t", left, test.left)
			}

			var session created
			fake.asks(t, "session.create", &session)

			_, err := client.Conversation(t.Context(), session.SessionID)
			if !errors.Is(err, ErrNoPlan) {
				t.Errorf("Conversation = %v, want ErrNoPlan for the failed plan", err)
			}
		})
	}
}

func TestDeny(t *testing.T) {
	requests := []sdk.PermissionRequest{
		&rpc.PermissionRequestWrite{FileName: "/work/notes.md"},
		&rpc.PermissionRequestShell{FullCommandText: "rm -rf /"},
		&rpc.PermissionRequestURL{URL: "https://example.com"},
	}

	for _, request := range requests {
		decision, err := deny(request, sdk.PermissionInvocation{SessionID: "session-1"})
		if _, rejected := decision.(*rpc.PermissionDecisionReject); !rejected || err != nil {
			t.Errorf("deny(%T) = %#v, %v, want a rejection", request, decision, err)
		}
	}
}
