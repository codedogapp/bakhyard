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

// interval is the poller's time between two polls. A bubble's clock starts at midnight UTC on
// 1 January 2000.
const interval = 5 * time.Minute

// listing is how a call of a fake Mine answers: with pulls, or with err when it isn't nil.
type listing struct {
	pulls []bitbucket.Pull
	err   error
}

// fakeMine is a fake Client's Mine, whose calls answer in turn as its listings say.
type fakeMine struct {
	listings []listing
	calls    atomic.Int32
}

func (f *fakeMine) mine(context.Context) ([]bitbucket.Pull, error) {
	call := int(f.calls.Add(1)) - 1
	if call >= len(f.listings) {
		return nil, errors.New("one listing too many")
	}

	return f.listings[call].pulls, f.listings[call].err
}

func TestPollerListsAtOnceAndEveryFiveMinutes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		none := listing{pulls: []bitbucket.Pull{}}
		_, fake, _ := polling(t.Context(), none, none, none)

		for _, test := range []struct {
			wait  time.Duration
			calls int32
		}{
			{0, 1}, {interval - time.Nanosecond, 1}, {time.Nanosecond, 2}, {interval, 3},
		} {
			time.Sleep(test.wait)
			synctest.Wait()

			if got := fake.calls.Load(); got != test.calls {
				t.Errorf("at %s: got %d listings, want %d", time.Now().UTC(), got, test.calls)
			}
		}
	})
}

func TestPollerAnswersMyPulls(t *testing.T) {
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

	synctest.Test(t, func(t *testing.T) {
		poller, _, logs := polling(t.Context(), listing{err: errors.New(failure)},
			listing{pulls: []bitbucket.Pull{pull}}, listing{pulls: []bitbucket.Pull{}})

		for _, test := range []struct {
			wait time.Duration
			want string
		}{
			{0, `{"failure":"` + failure + `"}`},
			{interval, `{"pulls":[` + pullJSON + `],"searched":"2000-01-01T00:05:00Z"}`},
			{interval, `{"pulls":[],"searched":"2000-01-01T00:10:00Z"}`},
		} {
			time.Sleep(test.wait)
			synctest.Wait()

			if got := latest(t, poller); got != test.want {
				t.Errorf("at %s: got %s, want %s", time.Now().UTC(), got, test.want)
			}
		}

		want := `level=ERROR msg="poll Bitbucket" err="` + failure + `"` + "\n"
		if !strings.HasSuffix(logs.String(), want) || strings.Count(logs.String(), "\n") != 1 {
			t.Errorf("got the logs %q, want one line that ends %q", logs, want)
		}
	})
}

// polling starts a poller, which lists as the listings say until ctx ends, and returns it, its
// fake Mine and its logs.
func polling(
	ctx context.Context, listings ...listing,
) (*bitbucket.Poller, *fakeMine, *bytes.Buffer) {
	var logs bytes.Buffer

	fake := &fakeMine{listings: listings}

	poller := bitbucket.NewPoller(fake.mine, slog.New(slog.NewTextHandler(&logs, nil)))
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
