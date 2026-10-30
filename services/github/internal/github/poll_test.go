package github_test

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/codedogapp/bakhyard/services/github/internal/github"
)

// interval is the time between two searches. A bubble's clock starts at midnight UTC on
// 1 January 2000.
const interval = 5 * time.Minute

// finding is how a call of a fake search, such as a Client's Mine, answers: with items, or with
// err when it isn't nil.
type finding struct {
	items []github.Item
	err   error
}

// fakeSearch is a fake Client's Mine, Reviews or Issues, whose calls answer in turn as its
// findings say.
type fakeSearch struct {
	findings []finding
	calls    atomic.Int32
}

func (f *fakeSearch) search(context.Context) ([]github.Item, error) {
	call := int(f.calls.Add(1)) - 1
	if call >= len(f.findings) {
		return nil, errors.New("one search too many")
	}

	return f.findings[call].items, f.findings[call].err
}

func TestPollerSearchesAtOnceAndEveryFiveMinutes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		empty := finding{items: []github.Item{}}
		_, fake, _ := polling(t.Context(), empty, empty, empty)

		for _, test := range []struct {
			wait  time.Duration
			calls int32
		}{
			{0, 1},
			{interval - time.Nanosecond, 1},
			{time.Nanosecond, 2},
			{interval, 3},
		} {
			time.Sleep(test.wait)
			synctest.Wait()

			if got := fake.calls.Load(); got != test.calls {
				t.Errorf("at %s: got %d searches, want %d", time.Now().UTC(), got, test.calls)
			}
		}
	})
}

func TestPollerAnswersTheItemsItFinds(t *testing.T) {
	const failure = "search GitHub for my open PRs: 500 Internal Server Error"

	item := github.Item{
		ID:      7,
		Title:   "Pay",
		Repo:    "acme/web",
		URL:     "https://github.com/acme/web/pull/7",
		Draft:   true,
		Created: time.Date(1999, 12, 31, 9, 0, 0, 0, time.UTC),
		Updated: time.Date(1999, 12, 31, 23, 0, 0, 0, time.UTC),
	}
	itemJSON := `{"id":7,"title":"Pay","repo":"acme/web",` +
		`"url":"https://github.com/acme/web/pull/7","draft":true,` +
		`"created":"1999-12-31T09:00:00Z","updated":"1999-12-31T23:00:00Z"}`
	found := `{"items":[` + itemJSON + `],"searched":"` + searched(interval) + `"}`

	synctest.Test(t, func(t *testing.T) {
		poller, _, logs := polling(t.Context(), finding{err: errors.New(failure)},
			finding{items: []github.Item{item}}, finding{err: errors.New(failure)},
			finding{items: []github.Item{}})

		for _, test := range []struct {
			wait time.Duration
			want string
		}{
			{0, `{"failure":"` + failure + `"}`},
			{interval, found},
			{interval, strings.TrimSuffix(found, "}") + `,"failure":"` + failure + `"}`},
			{interval, `{"items":[],"searched":"` + searched(3*interval) + `"}`},
		} {
			time.Sleep(test.wait)
			synctest.Wait()

			if got := latest(t, poller); got != test.want {
				t.Errorf("at %s: got %s, want %s", time.Now().UTC(), got, test.want)
			}
		}

		want := `level=ERROR msg="poll GitHub" err="` + failure + `"`

		lines := strings.Split(strings.TrimSuffix(logs.String(), "\n"), "\n")
		if len(lines) != 2 || !strings.HasSuffix(lines[0], want) ||
			!strings.HasSuffix(lines[1], want) {
			t.Errorf("got the logs %q, want two lines that each end %q", logs, want)
		}
	})
}

// searched returns the time at which a search began, after in a bubble, in RFC 3339.
func searched(after time.Duration) string {
	return time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC).Add(after).Format(time.RFC3339)
}

// polling starts a poller of the source GitHub, which searches as the findings say until ctx ends,
// and returns it, its fake search and its logs.
func polling(
	ctx context.Context, findings ...finding,
) (*github.Poller, *fakeSearch, *bytes.Buffer) {
	var logs bytes.Buffer

	fake := &fakeSearch{findings: findings}

	poller := github.NewPoller("GitHub", fake.search, slog.New(slog.NewTextHandler(&logs, nil)))
	go poller.Run(ctx)

	return poller, fake, &logs
}

// latest returns what poller has found, as JSON.
func latest(t *testing.T, poller *github.Poller) string {
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
