package bitbucket_test

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

	"github.com/codedogapp/bakhyard/services/pulls/internal/bitbucket"
)

// pollers are the constructors of the pollers, with their times between two polls and the names
// with which they log a failed poll. A bubble's clock starts at midnight UTC on 1 January 2000.
var pollers = []struct {
	name     string
	start    constructor
	interval time.Duration
	log      string
}{
	{"mine", bitbucket.NewPoller, 5 * time.Minute, "poll Bitbucket"},
	{"reviews", bitbucket.NewReviewsPoller, 15 * time.Minute, "poll Bitbucket reviews"},
}

// constructor makes a poller that lists with list and logs to logger, as NewPoller does.
type constructor = func(
	list func(context.Context) ([]bitbucket.Pull, error), logger *slog.Logger,
) *bitbucket.Poller

// listing is how a call of a fake listing, such as a Client's Mine, answers: with pulls, or with
// err when it isn't nil.
type listing struct {
	pulls []bitbucket.Pull
	err   error
}

// fakeLister is a fake Client's Mine or Reviews, whose calls answer in turn as its listings say.
type fakeLister struct {
	listings []listing
	calls    atomic.Int32
}

func (f *fakeLister) list(context.Context) ([]bitbucket.Pull, error) {
	call := int(f.calls.Add(1)) - 1
	if call >= len(f.listings) {
		return nil, errors.New("one listing too many")
	}

	return f.listings[call].pulls, f.listings[call].err
}

func TestPollerListsAtOnceAndAtItsInterval(t *testing.T) {
	for _, poller := range pollers {
		t.Run(poller.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				empty := listing{pulls: []bitbucket.Pull{}}
				_, fake, _ := polling(t.Context(), poller.start, empty, empty, empty)

				for _, test := range []struct {
					wait  time.Duration
					calls int32
				}{
					{0, 1},
					{poller.interval - time.Nanosecond, 1},
					{time.Nanosecond, 2},
					{poller.interval, 3},
				} {
					time.Sleep(test.wait)
					synctest.Wait()

					if got := fake.calls.Load(); got != test.calls {
						t.Errorf("at %s: got %d listings, want %d", time.Now().UTC(), got,
							test.calls)
					}
				}
			})
		})
	}
}

func TestPollerAnswersThePullsItFinds(t *testing.T) {
	const failure = "list my Bitbucket pull requests: 500 Internal Server Error"

	pull := bitbucket.Pull{
		ID:      7,
		Title:   "Pay",
		Repo:    "acme/web",
		URL:     "https://bitbucket.org/acme/web/pull-requests/7",
		Draft:   true,
		Created: time.Date(1999, 12, 31, 9, 0, 0, 0, time.UTC),
		Updated: time.Date(1999, 12, 31, 23, 0, 0, 0, time.UTC),
	}
	pullJSON := `{"id":7,"title":"Pay","repo":"acme/web",` +
		`"url":"https://bitbucket.org/acme/web/pull-requests/7","draft":true,` +
		`"created":"1999-12-31T09:00:00Z","updated":"1999-12-31T23:00:00Z"}`

	for _, started := range pollers {
		t.Run(started.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				poller, _, logs := polling(t.Context(), started.start,
					listing{err: errors.New(failure)}, listing{pulls: []bitbucket.Pull{pull}},
					listing{pulls: []bitbucket.Pull{}})

				for _, test := range []struct {
					wait time.Duration
					want string
				}{
					{0, `{"failure":"` + failure + `"}`},
					{started.interval, `{"pulls":[` + pullJSON + `],"searched":"` +
						searched(started.interval) + `"}`},
					{started.interval, `{"pulls":[],"searched":"` +
						searched(2*started.interval) + `"}`},
				} {
					time.Sleep(test.wait)
					synctest.Wait()

					if got := latest(t, poller); got != test.want {
						t.Errorf("at %s: got %s, want %s", time.Now().UTC(), got, test.want)
					}
				}

				want := `level=ERROR msg="` + started.log + `" err="` + failure + `"` + "\n"
				if !strings.HasSuffix(logs.String(), want) ||
					strings.Count(logs.String(), "\n") != 1 {
					t.Errorf("got the logs %q, want one line that ends %q", logs, want)
				}
			})
		})
	}
}

// searched returns the time at which a poll began, after in a bubble, in RFC 3339.
func searched(after time.Duration) string {
	return time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC).Add(after).Format(time.RFC3339)
}

// polling starts a poller made by start, which lists as the listings say until ctx ends, and
// returns it, its fake listing and its logs.
func polling(
	ctx context.Context, start constructor, listings ...listing,
) (*bitbucket.Poller, *fakeLister, *bytes.Buffer) {
	var logs bytes.Buffer

	fake := &fakeLister{listings: listings}

	poller := start(fake.list, slog.New(slog.NewTextHandler(&logs, nil)))
	go poller.Run(ctx)

	return poller, fake, &logs
}

// latest returns what poller has found, as JSON.
func latest(t *testing.T, poller *bitbucket.Poller) string {
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
