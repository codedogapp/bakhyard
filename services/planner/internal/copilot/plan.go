package copilot

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	sdk "github.com/github/copilot-sdk/go"
	"github.com/github/copilot-sdk/go/rpc"
)

// model works out the plans (decision 75).
const model = "claude-opus-5.5"

// exitOnly answers exit_plan_mode: plan mode ends, and the agent doesn't start on the plan.
const exitOnly = "exit_only"

// tools returns the tools that the agent may use. It writes the plan with create and edit,
// without which exit_plan_mode brings no plan.
func tools() []string {
	return []string{"ask_user", "view", "glob", "grep", "create", "edit", "exit_plan_mode"}
}

// plan is a plan that the agent works out in a session of its own.
type plan struct {
	session *sdk.Session
	// stopped closes as the client stops, which ends the questions' waits for their answers.
	stopped <-chan struct{}

	mu sync.Mutex
	// writing are the messages that the assistant is writing, with their text so far.
	writing []Message
	content string
	// questions are the agent's questions that wait for their answers, in the order asked.
	questions []waiting
	// calls are the questions of the agent's ask_user calls that have started and not yet ended.
	calls     askCalls
	followers map[chan Update]struct{}
}

func newPlan(stopped <-chan struct{}) *plan {
	return &plan{stopped: stopped, calls: askCalls{}, followers: map[chan Update]struct{}{}}
}

// Plan starts a session that works out a plan for prompt, in plan mode, and returns the
// session's ID. The agent goes on working after Plan returns.
func (c *Client) Plan(ctx context.Context, prompt string) (string, error) {
	planned := newPlan(c.stopped)

	session, err := c.copilot.CreateSession(ctx, &sdk.SessionConfig{
		Model:                  model,
		AvailableTools:         tools(),
		WorkingDirectory:       c.work,
		SkipCustomInstructions: new(true),
		Streaming:              new(true),
		OnEvent:                planned.see,
		OnPermissionRequest:    deny,
		OnUserInputRequest:     planned.ask,
		OnExitPlanModeRequest:  planned.keep,
	})
	if err != nil {
		return "", fmt.Errorf("start a session: %w", err)
	}

	planned.session = session

	err = planned.send(ctx, prompt)
	if err != nil {
		return "", errors.Join(fmt.Errorf("send the prompt: %w", err), session.Disconnect())
	}

	c.mu.Lock()
	c.plans[session.SessionID] = planned
	c.mu.Unlock()

	return session.SessionID, nil
}

// find returns the plan id, or ErrNoPlan when the client hasn't started that plan.
func (c *Client) find(id string) (*plan, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	planned, found := c.plans[id]
	if !found {
		return nil, ErrNoPlan
	}

	return planned, nil
}

// snapshot returns what the plan has, short of the messages that the CLI keeps: the messages
// that the assistant is writing, the questions that wait, and the plan once exit_plan_mode has
// brought it.
func (p *plan) snapshot() Conversation {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.now()
}

// now is snapshot, with p.mu held.
func (p *plan) now() Conversation {
	return Conversation{Writing: slices.Clone(p.writing), Questions: p.asking(), Plan: p.content}
}

// keep keeps the plan that exit_plan_mode brings, tells the followers, and ends plan mode.
func (p *plan) keep(
	request sdk.ExitPlanModeRequest,
	_ sdk.ExitPlanModeInvocation,
) (sdk.ExitPlanModeResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.content = request.PlanContent
	p.tell(Update{Change: ChangePlan, Plan: request.PlanContent})

	return sdk.ExitPlanModeResult{Approved: true, SelectedAction: exitOnly}, nil
}

// deny turns down every permission request, so the agent writes only the session's plan, which
// the CLI allows by itself.
func deny(sdk.PermissionRequest, sdk.PermissionInvocation) (rpc.PermissionDecision, error) {
	return &rpc.PermissionDecisionReject{}, nil
}
