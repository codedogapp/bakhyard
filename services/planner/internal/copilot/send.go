package copilot

import (
	"context"
	"fmt"

	sdk "github.com/github/copilot-sdk/go"
)

// Send sends text to the plan id as the user's next message, in plan mode, so that the agent works
// the plan out further; while the agent works, the CLI queues it. Send returns ErrNoPlan when the
// CLI keeps no such plan.
func (c *Client) Send(ctx context.Context, id, text string) error {
	planned, err := c.find(ctx, id)
	if err != nil {
		return err
	}

	err = planned.send(ctx, text)
	if err != nil {
		return fmt.Errorf("send the message: %w", err)
	}

	return nil
}

// send sends text to the plan's session as the user's message, in plan mode: the session starts in
// another mode, and exit_plan_mode ends plan mode as it brings the plan.
func (p *plan) send(ctx context.Context, text string) error {
	_, err := p.session.Send(ctx, sdk.MessageOptions{Prompt: text, AgentMode: sdk.AgentModePlan})

	return err
}
