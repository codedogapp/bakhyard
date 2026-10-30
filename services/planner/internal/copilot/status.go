package copilot

import sdk "github.com/github/copilot-sdk/go"

// Status is whether a plan's agent is working, and why it stopped, when an error stopped it.
type Status struct {
	// Working is whether the agent is working: from the start of a turn until the CLI says that
	// the session is idle, or that an error stopped it.
	Working bool `json:"working"`
	// Failure is the CLI's message for the error that stopped the agent, until it works again, or
	// "" when none did.
	Failure string `json:"failure"`
}

// mind takes in the events that start or stop the agent, and tells the followers when they
// change its status. Once the agent stops, it won't finish the messages it was writing, so they're
// dropped. p.mu is held.
func (p *plan) mind(event sdk.SessionEvent) {
	status := p.status

	switch data := event.Data.(type) {
	case *sdk.AssistantTurnStartData:
		status = Status{Working: true}
	case *sdk.SessionIdleData:
		status.Working = false
	case *sdk.SessionErrorData:
		status = Status{Failure: data.Message}
	default:
		return
	}

	if !status.Working {
		p.writing = nil
	}

	if status != p.status {
		p.status = status
		p.tell(Update{Change: ChangeStatus, Status: status})
	}
}
