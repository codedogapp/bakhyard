package poll_test

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/codedogapp/bakhyard/internal/poll"
)

// interval is the time between two polls of the tests' pollers. A bubble's clock starts at
// midnight UTC on 1 January 2000.
const interval = 7 * time.Minute

// found is what the tests' look-ups find.
type found struct {
	Names []string `json:"names,omitzero"`
}

// reply is how a call of a fake look-up replies.
type reply func(ctx context.Context) (found, error)

// fakeLookUp is a look-up whose calls reply in turn as its replies say.
type fakeLookUp struct {
	replies []reply
	calls   atomic.Int32
}

func (f *fakeLookUp) look(ctx context.Context) (found, error) {
	call := int(f.calls.Add(1)) - 1
	if call >= len(f.replies) {
		return found{}, errors.New("one look-up too many")
	}

	return f.replies[call](ctx)
}

// finding replies with names, which are an empty list when there are none.
func finding(names ...string) reply {
	return func(context.Context) (found, error) {
		return found{Names: append([]string{}, names...)}, nil
	}
}

func failing(message string) reply {
	return func(context.Context) (found, error) { return found{}, errors.New(message) }
}

// cut replies once ctx ends, as a look-up does when the service stops.
func cut(ctx context.Context) (found, error) {
	<-ctx.Done()

	return found{}, fmt.Errorf("look Acme up: %w", ctx.Err())
}

// polling starts a poller of Acme, which looks up as the replies say until ctx ends, and returns
// it, its fake look-up and its logs.
func polling(
	ctx context.Context,
	replies ...reply,
) (*poll.Poller[found], *fakeLookUp, *bytes.Buffer) {
	var logs bytes.Buffer

	fake := &fakeLookUp{replies: replies}
	poller := poll.New("Acme", interval, fake.look, slog.New(slog.NewTextHandler(&logs, nil)))

	go poller.Run(ctx)

	return poller, fake, &logs
}

// latest returns what poller has found, as JSON.
func latest(t *testing.T, poller *poll.Poller[found]) string {
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

func TestPollerLooksUpAtOnceAndEveryInterval(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		_, fake, _ := polling(t.Context(), finding(), finding(), finding())

		for _, test := range []struct {
			wait  time.Duration
			calls int32
		}{
			{0, 1}, {interval - time.Nanosecond, 1}, {time.Nanosecond, 2}, {interval, 3},
		} {
			time.Sleep(test.wait)
			synctest.Wait()

			if got := fake.calls.Load(); got != test.calls {
				t.Errorf("at %s: got %d look-ups, want %d", time.Now().UTC(), got, test.calls)
			}
		}
	})
}

func TestPollerWithNoIntervalLooksUpOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		fake := &fakeLookUp{replies: []reply{finding(), finding()}}
		poller := poll.New("Acme", 0, fake.look, slog.New(slog.DiscardHandler))

		go poller.Run(t.Context())

		time.Sleep(24 * time.Hour)
		synctest.Wait()

		if got := fake.calls.Load(); got != 1 {
			t.Errorf("got %d look-ups, want 1", got)
		}
	})
}

func TestPollerKeepsWhatItFoundWhenALookUpFails(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		poller, _, logs := polling(t.Context(),
			finding("Ada"), failing("look Acme up: 500 Internal Server Error"), finding())

		for _, test := range []struct {
			wait time.Duration
			want string
		}{
			{0, `{"names":["Ada"],"searched":"2000-01-01T00:00:00Z"}`},
			{interval, `{"names":["Ada"],"searched":"2000-01-01T00:00:00Z",` +
				`"failure":"look Acme up: 500 Internal Server Error"}`},
			{interval, `{"names":[],"searched":"2000-01-01T00:14:00Z"}`},
		} {
			time.Sleep(test.wait)
			synctest.Wait()

			if got := latest(t, poller); got != test.want {
				t.Errorf("at %s: got %s, want %s", time.Now().UTC(), got, test.want)
			}
		}

		want := `level=ERROR msg="poll Acme" err="look Acme up: 500 Internal Server Error"` + "\n"
		if !strings.HasSuffix(logs.String(), want) || strings.Count(logs.String(), "\n") != 1 {
			t.Errorf("got the logs %q, want one line that ends %q", logs, want)
		}
	})
}

func TestPollerTellsWhyTheFirstLookUpFailed(t *testing.T) {
	for _, test := range []struct{ message, want string }{
		{"look Acme up: 401 Unauthorized", `{"failure":"look Acme up: 401 Unauthorized"}`},
		// A status line may hold bytes that aren't UTF-8, which JSON can't carry.
		{"look Acme up: 500 \xffBroken", "{\"failure\":\"look Acme up: 500 \uFFFDBroken\"}"},
	} {
		synctest.Test(t, func(t *testing.T) {
			poller, _, _ := polling(t.Context(), failing(test.message))

			if got := latest(t, poller); got != test.want {
				t.Errorf("got %s, want %s", got, test.want)
			}
		})
	}
}

func TestPollerNotesWhenTheLookUpBegan(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		time.Sleep(1500 * time.Millisecond)

		poller, _, _ := polling(t.Context(), func(ctx context.Context) (found, error) {
			time.Sleep(2 * time.Second)

			return finding()(ctx)
		})

		snapshot, err := poller.Latest(t.Context())
		want := time.Date(2000, time.January, 1, 0, 0, 1, 0, time.UTC)

		if err != nil || !snapshot.Searched.Equal(want) ||
			snapshot.Searched.Location() != time.UTC {
			t.Errorf("got %v, %v, want the look-up noted at %s", snapshot.Searched, err, want)
		}
	})
}

func TestPollerIgnoresALookUpCutShortByStopping(t *testing.T) {
	for _, test := range []struct {
		replies []reply
		want    string
	}{
		{[]reply{cut}, `{}`},
		{[]reply{finding("Ada"), cut}, `{"names":["Ada"],"searched":"2000-01-01T00:00:00Z"}`},
	} {
		synctest.Test(t, func(t *testing.T) {
			ctx, stop := context.WithCancel(t.Context())
			poller, fake, logs := polling(ctx, test.replies...)

			time.Sleep(time.Duration(len(test.replies)-1) * interval)
			synctest.Wait()
			stop()
			synctest.Wait()

			got, calls := latest(t, poller), int(fake.calls.Load())
			if got != test.want || calls != len(test.replies) || logs.Len() != 0 {
				t.Errorf("got %s after %d look-ups, with the logs %q, want %s", got, calls, logs,
					test.want)
			}
		})
	}
}

func TestLatestWaitsForTheFirstLookUp(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		poller, _, _ := polling(t.Context(), func(ctx context.Context) (found, error) {
			time.Sleep(30 * time.Second)

			return finding("Ada")(ctx)
		})

		got := latest(t, poller)
		waited := time.Since(start)

		want := `{"names":["Ada"],"searched":"2000-01-01T00:00:00Z"}`
		if got != want || waited != 30*time.Second {
			t.Errorf("got %s after %s, want %s after 30s", got, waited, want)
		}
	})
}

func TestLatestGivesUpWhenItsContextEnds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		poller, _, _ := polling(t.Context(), cut)

		ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
		defer cancel()

		_, err := poller.Latest(ctx)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("got %v, want %v", err, context.DeadlineExceeded)
		}
	})
}
