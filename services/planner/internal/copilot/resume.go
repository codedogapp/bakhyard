package copilot

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"

	sdk "github.com/github/copilot-sdk/go"
)

// planFile is the file in a session's workspace that the agent writes the plan in, and whose
// content exit_plan_mode brings.
const planFile = "plan.md"

// find returns the plan with the ID planID: one that the client holds, or else one of an earlier
// run that the CLI keeps, which the client brings back and holds from then on. It returns ErrNoPlan
// when the CLI keeps no such plan.
func (c *Client) find(ctx context.Context, planID string) (*plan, error) {
	if planned, held := c.held(planID); held {
		return planned, nil
	}

	// One plan comes back at a time, so that two requests for a plan don't both resume its
	// session, which would take the session's events from the plan that the first brought back.
	c.resuming.Lock()
	defer c.resuming.Unlock()

	if planned, held := c.held(planID); held {
		return planned, nil
	}

	planned, err := c.resume(ctx, planID)
	if err != nil {
		return nil, err
	}

	c.hold(planID, planned)

	return planned, nil
}

// held returns the plan with the ID planID, if the client holds it.
func (c *Client) held(planID string) (*plan, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	planned, found := c.plans[planID]

	return planned, found
}

// hold holds planned as the plan with the ID planID until the client stops.
func (c *Client) hold(planID string, planned *plan) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.plans[planID] = planned
}

// resume brings back the plan with the ID planID from the CLI, with the plan from its session's
// plan file, or returns ErrNoPlan when the CLI keeps no such plan. A tool call that was still
// running as the client last stopped ends as interrupted.
func (c *Client) resume(ctx context.Context, planID string) (*plan, error) {
	kept, err := c.copilot.GetSessionMetadata(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("look the plan up: %w", err)
	}

	if kept == nil {
		return nil, ErrNoPlan
	}

	planned := newPlan(c.stopped)

	session, err := c.copilot.ResumeSession(ctx, planID, &sdk.ResumeSessionConfig{
		Model:                  model,
		AvailableTools:         tools(),
		WorkingDirectory:       c.work,
		SkipCustomInstructions: new(true),
		Streaming:              new(true),
		ContinuePendingWork:    new(false),
		OnEvent:                planned.see,
		OnPermissionRequest:    deny,
		OnUserInputRequest:     planned.ask,
		OnExitPlanModeRequest:  planned.keep,
	})
	if err != nil {
		return nil, fmt.Errorf("resume the session: %w", err)
	}

	planned.session = session

	content, err := readPlan(session.WorkspacePath())
	if err != nil {
		return nil, errors.Join(err, session.Disconnect())
	}

	planned.mu.Lock()
	planned.content = content
	planned.mu.Unlock()

	return planned, nil
}

// readPlan returns the plan in the plan file of the session's workspace, or "" when the agent
// hasn't written it or the session has no workspace, which the SDK allows.
func readPlan(workspace string) (string, error) {
	if workspace == "" {
		return "", nil
	}

	root, err := os.OpenRoot(workspace)
	if err != nil {
		return "", fmt.Errorf("open the session's workspace: %w", err)
	}
	defer func() { _ = root.Close() }()

	content, err := root.ReadFile(planFile)

	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "", nil
	case err != nil:
		return "", fmt.Errorf("read the plan: %w", err)
	}

	return string(content), nil
}
