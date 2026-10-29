package jira

import (
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"
)

// TestOwnWritesRunTogether runs the memory's methods together, so that the race detector fails it
// when one of them doesn't take the lock. The client's own tests can't show that: the race
// detector counts each write and read on a socket as synchronising the goroutines.
func TestOwnWritesRunTogether(t *testing.T) {
	const tickets = 20

	var (
		writes ownWrites
		calls  sync.WaitGroup
	)

	start := time.Date(2026, time.October, 21, 0, 0, 0, 0, time.UTC)

	var want []string

	for index := range tickets {
		issueID := strconv.Itoa(index)
		started := start.Add(time.Duration(index-tickets/2) * time.Hour)

		if !started.Before(start) {
			want = append(want, issueID)
		}

		calls.Go(func() { writes.remember(issueID, started) })
		calls.Go(func() { writes.unfound([]issue{{ID: issueID}}, start) })
		calls.Go(func() { writes.forget("gone" + issueID) })
	}

	calls.Wait()

	slices.Sort(want)

	if got := writes.unfound(nil, start); !slices.Equal(got, want) {
		t.Errorf("then remembered %q, want %q", got, want)
	}
}
