package poll_test

import (
	"context"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

func TestPollNowWaitsForThePollThatRuns(t *testing.T) {
	for _, test := range []struct {
		name    string
		replies []reply
		ask     time.Duration
		want    string
	}{
		{
			"the first",
			[]reply{after(30*time.Second, finding("Ada")), finding("Bo")},
			10 * time.Second, `{"names":["Bo"],"searched":"2000-01-01T00:00:30Z"}`,
		},
		{
			"a timed one",
			[]reply{finding("Ada"), after(30*time.Second, finding("Ada")), finding("Bo")},
			interval + 10*time.Second, `{"names":["Bo"],"searched":"2000-01-01T00:07:30Z"}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				poller, fake, _ := polling(t.Context(), test.replies...)

				time.Sleep(test.ask)

				start := time.Now()
				err := poller.PollNow(t.Context())
				waited := time.Since(start)

				got, calls := latest(t, poller), int(fake.calls.Load())
				if err != nil || waited != 20*time.Second || got != test.want ||
					calls != len(test.replies) {
					t.Errorf("got %v after %s, then %s after %d look-ups, want nil after 20s, "+
						"then %s after %d", err, waited, got, calls, test.want, len(test.replies))
				}
			})
		})
	}
}

func TestPollsTakeTurns(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var running, overlaps atomic.Int32

		slow := func(ctx context.Context) (found, error) {
			if running.Add(1) > 1 {
				overlaps.Add(1)
			}
			defer running.Add(-1)

			// The bubble's clock stops when the test ends, which ends ctx.
			select {
			case <-time.After(2 * time.Minute):
			case <-ctx.Done():
			}

			return finding()(ctx)
		}
		poller, _, _ := polling(t.Context(), slices.Repeat([]reply{slow}, 8)...)

		synctest.Wait()

		start := time.Now()

		var asks sync.WaitGroup
		for range 3 {
			asks.Go(func() {
				if err := poller.PollNow(t.Context()); err != nil {
					t.Error(err)
				}
			})
		}

		asks.Wait()

		// The first poll and the asked ones run in turn, and the timed one, due at 7m, waits for
		// the last.
		if waited := time.Since(start); waited != 8*time.Minute || overlaps.Load() != 0 {
			t.Errorf("the asks ended after %s, with %d look-ups that overlapped, want 8m0s and 0",
				waited, overlaps.Load())
		}
	})
}
