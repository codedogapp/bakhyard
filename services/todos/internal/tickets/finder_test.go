package tickets_test

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/codedogapp/bakhyard/internal/jira"
	"github.com/codedogapp/bakhyard/services/todos/internal/tickets"
)

// reply is how a call of a fake search replies.
type reply func(ctx context.Context) ([]jira.Ticket, error)

// fakeJira is a fake Jira, whose searches reply in turn as its replies say, and the fake list of
// the keys that the todos name.
type fakeJira struct {
	replies []reply
	calls   atomic.Int32
	// keys replies to each look for the todos' keys, with none when it's nil.
	keys func(ctx context.Context) ([]string, error)
	// fetches replies to each fetch, with no tickets when it's nil.
	fetches func(ctx context.Context, keys []string) ([]jira.Ticket, error)
	mu      sync.Mutex
	asked   []string
}

func (f *fakeJira) Search(ctx context.Context) ([]jira.Ticket, error) {
	call := int(f.calls.Add(1)) - 1
	if call >= len(f.replies) {
		return nil, errors.New("one search too many")
	}

	return f.replies[call](ctx)
}

func (f *fakeJira) Fetch(ctx context.Context, keys []string) ([]jira.Ticket, error) {
	f.mu.Lock()
	f.asked = append(f.asked, keys...)
	f.mu.Unlock()

	if f.fetches == nil {
		return []jira.Ticket{}, nil
	}

	return f.fetches(ctx, keys)
}

// linked returns the keys that the todos name, as a Store's Linked does.
func (f *fakeJira) linked(ctx context.Context) ([]string, error) {
	if f.keys == nil {
		return []string{}, nil
	}

	return f.keys(ctx)
}

// fetched returns the keys of every fetch so far, in turn.
func (f *fakeJira) fetched() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.asked
}

// found replies with the open tickets, an empty list when there are none, as a Client's are.
func found(open ...jira.Ticket) reply {
	return func(context.Context) ([]jira.Ticket, error) {
		return append([]jira.Ticket{}, open...), nil
	}
}

func failing(message string) reply {
	return func(context.Context) ([]jira.Ticket, error) { return nil, errors.New(message) }
}

// polling starts a poller, which searches as the replies say until ctx ends, and returns it, its
// fake Jira and its logs. The todos name no tickets.
func polling(ctx context.Context, replies ...reply) (*tickets.Poller, *fakeJira, *bytes.Buffer) {
	fake := &fakeJira{replies: replies}
	poller, logs := pollingWith(ctx, fake)

	return poller, fake, logs
}

// pollingWith starts a poller on fake until ctx ends, and returns it and its logs.
func pollingWith(ctx context.Context, fake *fakeJira) (*tickets.Poller, *bytes.Buffer) {
	var logs bytes.Buffer

	poller := tickets.NewPoller(fake, fake.linked, slog.New(slog.NewTextHandler(&logs, nil)))
	go poller.Run(ctx)

	return poller, &logs
}

// latest returns what poller has found, as JSON.
func latest(t *testing.T, poller *tickets.Poller) string {
	t.Helper()

	snapshot, err := poller.Latest(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	encoded, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}

	return string(encoded)
}

func payOne() jira.Ticket {
	return jira.Ticket{
		Key: "PAY-1", Summary: "Pay", Status: "To Do", Category: jira.ToDo,
		URL: "https://acme.example/browse/PAY-1",
	}
}
