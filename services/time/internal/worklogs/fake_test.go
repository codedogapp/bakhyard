package worklogs_test

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"log/slog"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/codedogapp/bakhyard/internal/jira"
	"github.com/codedogapp/bakhyard/services/time/internal/worklogs"
)

// fakeJira is a fake Jira, whose reads of worklogs all reply the same, and which keeps the span
// of each read.
type fakeJira struct {
	// logged are the worklogs that each read finds, when err is nil.
	logged []jira.Worklog
	// err fails each read when it isn't nil.
	err   error
	mu    sync.Mutex
	spans []string
}

// worklogs reads the worklogs from start up to end, as a jira.Client's Worklogs does.
func (f *fakeJira) worklogs(_ context.Context, start, end time.Time) ([]jira.Worklog, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.spans = append(f.spans,
		start.UTC().Format(time.RFC3339Nano)+" "+end.UTC().Format(time.RFC3339Nano))
	if f.err != nil {
		return nil, f.err
	}

	return append([]jira.Worklog{}, f.logged...), nil
}

// read returns the span of each read so far, in turn, in UTC.
func (f *fakeJira) read() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return slices.Clone(f.spans)
}

// polling starts a poller in Kiritimati's zone, which reads worklogs from fake until the test
// ends, and returns it and its logs.
func polling(t *testing.T, fake *fakeJira) (*worklogs.Poller, *bytes.Buffer) {
	t.Helper()

	kiritimati, err := time.LoadLocation("Pacific/Kiritimati")
	if err != nil {
		t.Fatal(err)
	}

	var logs bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&logs, nil))
	poller := worklogs.NewPoller(fake.worklogs, kiritimati, logger)

	go poller.Run(t.Context())

	return poller, &logs
}

// latest returns what poller has found, as JSON.
func latest(t *testing.T, poller *worklogs.Poller) string {
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
