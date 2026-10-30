package copilot

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/github/copilot-sdk/go"
)

// ErrNoPlan means that the client hasn't started a plan with the ID asked for.
var ErrNoPlan = errors.New("no such plan")

// Author is who wrote a message of a plan's conversation.
type Author string

const (
	// User is the user, who asks for the plan.
	User Author = "user"
	// Assistant is the agent, which works out the plan.
	Assistant Author = "assistant"
)

// Message is a message of a plan's conversation.
type Message struct {
	From Author `json:"from"`
	Text string `json:"text"`
}

// Conversation is a plan's conversation so far, and the plan once the agent has brought it.
type Conversation struct {
	Messages []Message `json:"messages"`
	Plan     string    `json:"plan"`
}

// Conversation returns the conversation of the plan id so far, which the CLI keeps, or ErrNoPlan
// when the client hasn't started that plan.
func (c *Client) Conversation(ctx context.Context, id string) (Conversation, error) {
	c.mu.Lock()
	planned, found := c.plans[id]
	c.mu.Unlock()

	if !found {
		return Conversation{}, ErrNoPlan
	}

	events, err := planned.session.GetEvents(ctx)
	if err != nil {
		return Conversation{}, fmt.Errorf("read the plan's messages: %w", err)
	}

	return Conversation{Messages: messages(events), Plan: planned.text()}, nil
}

// messages returns the messages in events, in their order: the user's own, which have no source
// or the user's, unlike those that the CLI adds, and the assistant's that have text, unlike those
// that only call tools.
func messages(events []sdk.SessionEvent) []Message {
	var list []Message

	for _, event := range events {
		switch data := event.Data.(type) {
		case *sdk.UserMessageData:
			if data.Source == nil || *data.Source == string(sdk.MessageSourceUser) {
				list = append(list, Message{From: User, Text: data.Content})
			}
		case *sdk.AssistantMessageData:
			if data.Content != "" {
				list = append(list, Message{From: Assistant, Text: data.Content})
			}
		}
	}

	return list
}
