package copilot

import (
	"context"
	"fmt"
	"slices"
	"time"

	sdk "github.com/github/copilot-sdk/go"
)

// Entry is a plan in the list of plans.
type Entry struct {
	ID string `json:"id"`
	// Title is the CLI's name for the plan, which may be its prompt, or "" before it has one.
	Title     string    `json:"title"`
	StartedAt time.Time `json:"started_at"`
	// UpdatedAt is when the CLI's record of the plan last changed.
	UpdatedAt time.Time `json:"updated_at"`
}

// Plans returns the plans that the CLI keeps, those of the client's earlier runs among them, the
// last updated first.
func (c *Client) Plans(ctx context.Context) ([]Entry, error) {
	sessions, err := c.copilot.ListSessions(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("list the sessions: %w", err)
	}

	entries := make([]Entry, 0, len(sessions))
	for _, session := range sessions {
		entries = append(entries, entryOf(session))
	}

	slices.SortFunc(entries, func(a, b Entry) int { return b.UpdatedAt.Compare(a.UpdatedAt) })

	return entries, nil
}

// entryOf returns the entry of the plan whose session the CLI describes with session.
func entryOf(session sdk.SessionMetadata) Entry {
	entry := Entry{
		ID:        session.SessionID,
		StartedAt: session.StartTime,
		UpdatedAt: session.ModifiedTime,
	}
	if session.Summary != nil {
		entry.Title = *session.Summary
	}

	return entry
}
