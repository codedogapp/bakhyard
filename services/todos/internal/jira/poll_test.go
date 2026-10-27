package jira_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/codedogapp/bakhyard/services/todos/internal/jira"
)

// interval is the poller's time between two polls. A bubble's clock starts at midnight UTC
// on 1 January 2000.
const interval = 5 * time.Minute

// payOneJSON is payOne as JSON.
const payOneJSON = `{"key":"PAY-1","summary":"Pay","status":"To Do","category":"to-do",` +
	`"url":"https://acme.example/browse/PAY-1"}`

func TestPollerSearchesAtOnceAndEveryFiveMinutes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		_, fake, _ := polling(t.Context(), found(), found(), found())

		for _, test := range []struct {
			wait  time.Duration
			calls int32
		}{
			{0, 1}, {interval - time.Nanosecond, 1}, {time.Nanosecond, 2}, {interval, 3},
		} {
			time.Sleep(test.wait)
			synctest.Wait()

			if got := fake.calls.Load(); got != test.calls {
				t.Errorf("at %s: got %d searches, want %d", time.Now().UTC(), got, test.calls)
			}
		}
	})
}

func TestPollerKeepsTheTicketsWhenASearchFails(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		poller, _, logs := polling(t.Context(),
			found(payOne()), failing("search Jira: 500 Internal Server Error"), found())

		for _, test := range []struct {
			wait time.Duration
			want string
		}{
			{0, `{"tickets":[` + payOneJSON + `],"searched":"2000-01-01T00:00:00Z"}`},
			{interval, `{"tickets":[` + payOneJSON + `],"searched":"2000-01-01T00:00:00Z",` +
				`"failure":"search Jira: 500 Internal Server Error"}`},
			{interval, `{"tickets":[],"searched":"2000-01-01T00:10:00Z"}`},
		} {
			time.Sleep(test.wait)
			synctest.Wait()

			if got := latest(t, poller); got != test.want {
				t.Errorf("at %s: got %s, want %s", time.Now().UTC(), got, test.want)
			}
		}

		want := `level=ERROR msg="poll Jira" err="search Jira: 500 Internal Server Error"` + "\n"
		if !strings.HasSuffix(logs.String(), want) || strings.Count(logs.String(), "\n") != 1 {
			t.Errorf("got the logs %q, want one line that ends %q", logs, want)
		}
	})
}

func TestPollerTellsWhyTheFirstSearchFailed(t *testing.T) {
	for _, test := range []struct{ message, want string }{
		{"search Jira: 401 Unauthorized", `{"failure":"search Jira: 401 Unauthorized"}`},
		// A status line may hold bytes that aren't UTF-8, which JSON can't carry.
		{"search Jira: 500 \xffBroken", "{\"failure\":\"search Jira: 500 \uFFFDBroken\"}"},
	} {
		synctest.Test(t, func(t *testing.T) {
			poller, _, _ := polling(t.Context(), failing(test.message))

			if got := latest(t, poller); got != test.want {
				t.Errorf("got %s, want %s", got, test.want)
			}
		})
	}
}

func TestPollerNotesWhenTheSearchBegan(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		time.Sleep(1500 * time.Millisecond)

		poller, _, _ := polling(t.Context(), func(ctx context.Context) ([]jira.Ticket, error) {
			time.Sleep(2 * time.Second)

			return found()(ctx)
		})

		snapshot, err := poller.Latest(t.Context())
		want := time.Date(2000, time.January, 1, 0, 0, 1, 0, time.UTC)

		if err != nil || !snapshot.Searched.Equal(want) ||
			snapshot.Searched.Location() != time.UTC {
			t.Errorf("got %v, %v, want the search noted at %s", snapshot.Searched, err, want)
		}
	})
}

func TestPollerIgnoresASearchCutShortByStopping(t *testing.T) {
	for _, test := range []struct {
		replies []reply
		want    string
	}{
		{[]reply{cut}, `{}`},
		{
			[]reply{found(payOne()), cut},
			`{"tickets":[` + payOneJSON + `],"searched":"2000-01-01T00:00:00Z"}`,
		},
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
				t.Errorf("got %s after %d searches, with the logs %q, want %s", got, calls, logs,
					test.want)
			}
		})
	}
}

func TestLatestWaitsForTheFirstSearch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		poller, _, _ := polling(t.Context(), func(ctx context.Context) ([]jira.Ticket, error) {
			time.Sleep(30 * time.Second)

			return found(payOne())(ctx)
		})

		got := latest(t, poller)
		waited := time.Since(start)

		want := `{"tickets":[` + payOneJSON + `],"searched":"2000-01-01T00:00:00Z"}`
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
