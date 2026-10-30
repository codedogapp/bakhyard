package copilot

import (
	"context"
	"slices"

	sdk "github.com/github/copilot-sdk/go"
)

// backlog is how many updates a follower can fall behind before it's dropped: as many events as
// the SDK queues for a session.
const backlog = 128

// Change is what an update to a plan's conversation brings.
type Change string

const (
	// ChangeDelta is more text of a message that the assistant is writing.
	ChangeDelta Change = "delta"
	// ChangeMessage is a whole message, the user's or the assistant's.
	ChangeMessage Change = "message"
	// ChangePlan is the plan, which exit_plan_mode has brought.
	ChangePlan Change = "plan"
	// ChangeQuestion is a question of the agent's, which waits for the user's answer.
	ChangeQuestion Change = "question"
	// ChangeAnswered is a question that has had its answer, and no longer waits.
	ChangeAnswered Change = "answered"
	// ChangeStatus is a change of whether the agent is working, or of the error that stopped it.
	ChangeStatus Change = "status"
)

// Update is a change to a plan's conversation.
type Update struct {
	Change Change
	// Message is the whole message, or for ChangeDelta the message's ID and its new text.
	Message Message
	// Plan is the plan, for ChangePlan.
	Plan string
	// Question is the question, for ChangeQuestion and ChangeAnswered.
	Question Question
	// Status is the agent's new status, for ChangeStatus.
	Status Status
}

// Follow returns the conversation of the plan id so far, as Conversation does, and its updates
// from then on, until ctx ends or the follower falls backlog updates behind: then the channel
// closes. Updates that come while Follow reads the conversation can repeat its messages, which
// Conversation.Repeats tells.
func (c *Client) Follow(ctx context.Context, id string) (Conversation, <-chan Update, error) {
	planned, err := c.find(ctx, id)
	if err != nil {
		return Conversation{}, nil, err
	}

	updates, talk := planned.follow()

	talk, err = planned.read(ctx, talk)
	if err != nil {
		planned.unfollow(updates)

		return Conversation{}, nil, err
	}

	context.AfterFunc(ctx, func() { planned.unfollow(updates) })

	return talk, updates, nil
}

// follow adds a follower and returns its updates, with what the plan has as it joins, which the
// updates go on from.
func (p *plan) follow() (chan Update, Conversation) {
	p.mu.Lock()
	defer p.mu.Unlock()

	updates := make(chan Update, backlog)
	p.followers[updates] = struct{}{}

	return updates, p.now()
}

// unfollow drops the follower of updates, unless it's gone already.
func (p *plan) unfollow(updates chan Update) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, following := p.followers[updates]; following {
		p.drop(updates)
	}
}

// see takes in the session's events: it keeps the text of the messages that the assistant is
// writing, and tells the followers about it, about each whole message, the agent's questions and
// the user's answers among them, and about the agent's status.
func (p *plan) see(event sdk.SessionEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch data := event.Data.(type) {
	case *sdk.AssistantMessageDeltaData:
		p.write(data.MessageID, data.DeltaContent)
	case *sdk.AssistantMessageData:
		p.writing = slices.DeleteFunc(p.writing, withID(data.MessageID))
	}

	p.mind(event)

	if message, ok := messageOf(event); ok {
		p.tell(Update{Change: ChangeMessage, Message: message})
	}

	for _, message := range p.calls.see(event) {
		p.tell(Update{Change: ChangeMessage, Message: message})
	}
}

// write adds text to the message messageID that the assistant is writing, and tells the
// followers. p.mu is held.
func (p *plan) write(messageID, text string) {
	if text == "" {
		return
	}

	piece := Message{ID: messageID, From: Assistant, Text: text}
	p.tell(Update{Change: ChangeDelta, Message: piece})

	index := slices.IndexFunc(p.writing, withID(messageID))
	if index < 0 {
		p.writing = append(p.writing, piece)

		return
	}

	p.writing[index].Text += text
}

// tell sends update to every follower without waiting, since the SDK holds the session's next
// events until the handler returns: a follower that has fallen backlog updates behind is
// dropped. p.mu is held.
func (p *plan) tell(update Update) {
	for updates := range p.followers {
		select {
		case updates <- update:
		default:
			p.drop(updates)
		}
	}
}

// drop removes the follower of updates and closes them. p.mu is held.
func (p *plan) drop(updates chan Update) {
	delete(p.followers, updates)
	close(updates)
}
