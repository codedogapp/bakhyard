package poll_test

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/codedogapp/bakhyard/internal/poll"
)

// after replies as then does, once wait has passed.
func after(wait time.Duration, then reply) reply {
	return func(ctx context.Context) (found, error) {
		time.Sleep(wait)

		return then(ctx)
	}
}

func TestPollNowPollsAtOnce(t *testing.T) {
	for _, test := range []struct {
		reply     reply
		want, log string
	}{
		{finding("Ada", "Bo"), `{"names":["Ada","Bo"],"searched":"2000-01-01T00:01:30Z"}`, ""},
		{
			failing("look Acme up: 503 Service Unavailable"),
			`{"names":["Ada"],"searched":"2000-01-01T00:00:00Z",` +
				`"failure":"look Acme up: 503 Service Unavailable"}`,
			`level=ERROR msg="poll Acme" err="look Acme up: 503 Service Unavailable"` + "\n",
		},
	} {
		synctest.Test(t, func(t *testing.T) {
			poller, fake, logs := polling(t.Context(), finding("Ada"),
				after(20*time.Second, test.reply))

			time.Sleep(90 * time.Second)

			start := time.Now()
			err := poller.PollNow(t.Context())
			waited := time.Since(start)

			got, calls := latest(t, poller), fake.calls.Load()
			if err != nil || waited != 20*time.Second || got != test.want || calls != 2 {
				t.Errorf("got %v after %s, then %s after %d look-ups, want nil after 20s, then %s "+
					"after 2", err, waited, got, calls, test.want)
			}

			lines := strings.Count(test.log, "\n")
			if !strings.HasSuffix(logs.String(), test.log) ||
				strings.Count(logs.String(), "\n") != lines {
				t.Errorf("got the logs %q, want %d line(s) that end %q", logs, lines, test.log)
			}
		})
	}
}

func TestPollNowKeepsTheTimedPollsOnTime(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		poller, fake, _ := polling(t.Context(), finding(), finding(), finding(), finding())

		time.Sleep(time.Minute)

		if err := poller.PollNow(t.Context()); err != nil {
			t.Fatal(err)
		}

		for _, test := range []struct {
			wait  time.Duration
			calls int32
		}{
			{0, 2}, {interval - time.Minute - time.Nanosecond, 2}, {time.Nanosecond, 3}, {interval, 4},
		} {
			time.Sleep(test.wait)
			synctest.Wait()

			if got := fake.calls.Load(); got != test.calls {
				t.Errorf("at %s: got %d look-ups, want %d", time.Now().UTC(), got, test.calls)
			}
		}
	})
}

func TestPollNowGivesUpWhenItsContextEnds(t *testing.T) {
	for _, test := range []struct {
		name    string
		replies []reply
	}{
		{"waiting for its turn", []reply{after(time.Minute, finding("Ada"))}},
		{"during its poll", []reply{finding("Ada"), cut}},
	} {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				poller, fake, logs := polling(t.Context(), test.replies...)

				synctest.Wait()

				ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
				defer cancel()

				start := time.Now()
				err := poller.PollNow(ctx)
				waited := time.Since(start)

				got := latest(t, poller)

				want := `{"names":["Ada"],"searched":"2000-01-01T00:00:00Z"}`
				if !errors.Is(err, context.DeadlineExceeded) || waited != 10*time.Second ||
					got != want || int(fake.calls.Load()) != len(test.replies) || logs.Len() != 0 {
					t.Errorf("got %v after %s, then %s after %d look-ups, with the logs %q, "+
						"want %v after 10s, then %s after %d, with none", err, waited, got,
						fake.calls.Load(), logs, context.DeadlineExceeded, want, len(test.replies))
				}
			})
		})
	}
}

func TestPollNowPollsWithoutRun(t *testing.T) {
	for _, test := range []struct {
		name  string
		every time.Duration
		stop  bool
	}{
		{"with no interval", 0, false},
		{"after Run has stopped", interval, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				fake := &fakeLookUp{replies: []reply{finding("Ada"), finding("Bo")}}
				poller := poll.New("Acme", test.every, fake.look, slog.New(slog.DiscardHandler))

				ctx, stop := context.WithCancel(t.Context())
				defer stop()

				go poller.Run(ctx)

				synctest.Wait()

				if test.stop {
					stop()
				}

				time.Sleep(time.Hour)

				err := poller.PollNow(t.Context())

				want := `{"names":["Bo"],"searched":"2000-01-01T01:00:00Z"}`
				if got := latest(t, poller); err != nil || got != want {
					t.Errorf("got %v, then %s, want nil, then %s", err, got, want)
				}
			})
		})
	}
}

func TestPollNowPollsBeforeRunRuns(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		fake := &fakeLookUp{replies: []reply{finding("Ada")}}
		poller := poll.New("Acme", interval, fake.look, slog.New(slog.DiscardHandler))

		if err := poller.PollNow(t.Context()); err != nil || fake.calls.Load() != 1 {
			t.Errorf("got %v after %d look-ups, want nil after 1", err, fake.calls.Load())
		}
	})
}
