package copilot

import (
	"context"
	"errors"
	"fmt"
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

// Plan is a plan that the agent works out in a session of its own.
type Plan struct {
	id      string
	mu      sync.Mutex
	content string
}

// Plan starts a session that works out a plan for prompt, in plan mode. The agent goes on
// working after Plan returns.
func (c *Client) Plan(ctx context.Context, prompt string) (*Plan, error) {
	plan := &Plan{}

	session, err := c.copilot.CreateSession(ctx, &sdk.SessionConfig{
		Model:                  model,
		AvailableTools:         tools(),
		WorkingDirectory:       c.work,
		SkipCustomInstructions: new(true),
		OnPermissionRequest:    deny,
		OnExitPlanModeRequest:  plan.keep,
	})
	if err != nil {
		return nil, fmt.Errorf("start a session: %w", err)
	}

	plan.id = session.SessionID

	_, err = session.Send(ctx, sdk.MessageOptions{Prompt: prompt, AgentMode: sdk.AgentModePlan})
	if err != nil {
		return nil, errors.Join(fmt.Errorf("send the prompt: %w", err), session.Disconnect())
	}

	return plan, nil
}

// ID returns the ID of the plan's session.
func (p *Plan) ID() string {
	return p.id
}

// Content returns the plan that exit_plan_mode brought, or "" while the agent works on it.
func (p *Plan) Content() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.content
}

// keep keeps the plan that exit_plan_mode brings, and ends plan mode.
func (p *Plan) keep(
	request sdk.ExitPlanModeRequest,
	_ sdk.ExitPlanModeInvocation,
) (sdk.ExitPlanModeResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.content = request.PlanContent

	return sdk.ExitPlanModeResult{Approved: true, SelectedAction: exitOnly}, nil
}

// deny turns down every permission request, so the agent writes only the session's plan, which
// the CLI allows by itself.
func deny(sdk.PermissionRequest, sdk.PermissionInvocation) (rpc.PermissionDecision, error) {
	return &rpc.PermissionDecisionReject{}, nil
}
