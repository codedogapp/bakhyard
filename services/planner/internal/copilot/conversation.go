package copilot

import (
	"context"
	"errors"
	"fmt"
	"slices"

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
	// ID is the ID of the user's message's event, or of the assistant's message, which the pieces
	// of its text name as they come.
	ID   string `json:"id"`
	From Author `json:"from"`
	Text string `json:"text"`
}

// Conversation is a plan's conversation so far, and the plan once the agent has brought it.
type Conversation struct {
	Messages []Message `json:"messages"`
	// Writing are the messages that the assistant is still writing, with their text so far.
	Writing []Message `json:"writing"`
	Plan    string    `json:"plan"`
}

// Conversation returns the conversation of the plan id so far, from the CLI's record and the
// messages being written, or ErrNoPlan when the client hasn't started that plan.
func (c *Client) Conversation(ctx context.Context, id string) (Conversation, error) {
	planned, err := c.find(id)
	if err != nil {
		return Conversation{}, err
	}

	return planned.read(ctx, planned.snapshot())
}

// Repeats reports whether update brings a message that c already has whole: an update that came
// while Follow read c.
func (c Conversation) Repeats(update Update) bool {
	return update.Change != ChangePlan && c.has(update.Message.ID)
}

// has reports whether c has the whole message id.
func (c Conversation) has(id string) bool {
	return slices.ContainsFunc(c.Messages, withID(id))
}

// read adds the plan's messages, which the CLI keeps, to talk, what the plan had before, and
// leaves out the messages being written that the CLI has since kept whole.
func (p *plan) read(ctx context.Context, talk Conversation) (Conversation, error) {
	events, err := p.session.GetEvents(ctx)
	if err != nil {
		return Conversation{}, fmt.Errorf("read the plan's messages: %w", err)
	}

	talk.Messages = messages(events)
	talk.Writing = slices.DeleteFunc(talk.Writing, func(message Message) bool {
		return talk.has(message.ID)
	})

	return talk, nil
}

// messages returns the messages in events, in their order.
func messages(events []sdk.SessionEvent) []Message {
	var list []Message

	for _, event := range events {
		if message, ok := messageOf(event); ok {
			list = append(list, message)
		}
	}

	return list
}

// messageOf returns the message that event brings, if any: the user's own, which have no source
// or the user's, unlike those that the CLI adds, and the assistant's that have text, unlike those
// that only call tools.
func messageOf(event sdk.SessionEvent) (Message, bool) {
	switch data := event.Data.(type) {
	case *sdk.UserMessageData:
		mine := data.Source == nil || *data.Source == string(sdk.MessageSourceUser)

		return Message{ID: event.ID, From: User, Text: data.Content}, mine
	case *sdk.AssistantMessageData:
		said := Message{ID: data.MessageID, From: Assistant, Text: data.Content}

		return said, data.Content != ""
	default:
		return Message{}, false
	}
}

// withID returns a test of whether a message has the ID id.
func withID(id string) func(Message) bool {
	return func(message Message) bool { return message.ID == id }
}
