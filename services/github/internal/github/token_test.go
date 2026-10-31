package github_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/codedogapp/bakhyard/services/github/internal/github"
)

// expirationHeader is the header in which GitHub tells when a token expires.
const expirationHeader = "GitHub-Authentication-Token-Expiration"

func TestExpiryAsksGitHubAboutTheUser(t *testing.T) {
	fake := &fakeGitHub{
		pages: map[string]reply{"/user": okay(t, map[string]any{"login": "you", "id": 1})},
		headers: map[string]http.Header{
			"/user": {expirationHeader: {"2027-10-27 22:00:00 UTC"}},
		},
	}

	got, err := clientOf(fake.serve(t)).Expiry(t.Context())

	want := time.Date(2027, time.October, 27, 22, 0, 0, 0, time.UTC)
	if err != nil || !got.Equal(want) || got.Location() != time.UTC {
		t.Errorf("got %v and %v, want %v", got, err, want)
	}

	asked := []asked{{
		"GET /user", "Bearer " + token, "application/vnd.github+json", "2026-03-10", "bakhyard",
	}}
	if notes := fake.notes(); !slices.Equal(notes, asked) {
		t.Errorf("got the requests %q, want %q", notes, asked)
	}
}

func TestExpiryReadsATimeWithAnOffset(t *testing.T) {
	fake := &fakeGitHub{
		pages: map[string]reply{"/user": okay(t, map[string]any{"id": 1})},
		headers: map[string]http.Header{
			"/user": {expirationHeader: {"2023-04-26 23:23:18 +0200"}},
		},
	}

	got, err := clientOf(fake.serve(t)).Expiry(t.Context())

	want := time.Date(2023, time.April, 26, 21, 23, 18, 0, time.UTC)
	if err != nil || !got.Equal(want) || got.Location() != time.UTC {
		t.Errorf("got %v and %v, want %v", got, err, want)
	}
}

func TestExpiryIsZeroForATokenThatNeverExpires(t *testing.T) {
	fake := &fakeGitHub{pages: map[string]reply{"/user": okay(t, map[string]any{"id": 1})}}

	got, err := clientOf(fake.serve(t)).Expiry(t.Context())
	if err != nil || !got.IsZero() {
		t.Errorf("got %v and %v, want the zero time", got, err)
	}
}

func TestExpiryFailsOnATimeThatIsNoUse(t *testing.T) {
	for _, value := range []string{
		"soon", "2027-10-27", "27/10/2027 22:00:00 UTC", "2027-10-27T22:00:00Z",
		"2027-10-27 22:00:00 CEST", "2027-10-27 22:00:00 +02:00",
	} {
		fake := &fakeGitHub{
			pages:   map[string]reply{"/user": okay(t, map[string]any{"id": 1})},
			headers: map[string]http.Header{"/user": {expirationHeader: {value}}},
		}

		got, err := clientOf(fake.serve(t)).Expiry(t.Context())

		want := "read GitHub's answer: " + expirationHeader + ` "` + value + `" is not a time`
		if err == nil || err.Error() != want || !got.IsZero() {
			t.Errorf("%q: got %v and %v, want the error %s", value, got, err, want)
		}
	}
}

func TestExpiryFailsWhenGitHubSaysNo(t *testing.T) {
	fake := &fakeGitHub{pages: map[string]reply{
		"/user": {http.StatusUnauthorized, `{"message":"Bad credentials"}`},
	}}

	got, err := clientOf(fake.serve(t)).Expiry(t.Context())

	want := "ask GitHub when the token expires: 401 Unauthorized"
	if err == nil || err.Error() != want || !got.IsZero() {
		t.Errorf("got %v and %v, want the error %s", got, err, want)
	}
}

func TestExpiryPollerKeepsTheLatestAnswer(t *testing.T) {
	const failure = "ask GitHub when the token expires: 502 Bad Gateway"

	expires := time.Date(2027, time.October, 27, 22, 0, 0, 0, time.UTC)
	answers := []error{errors.New(failure), nil, errors.New(failure)}

	var calls atomic.Int32

	expiry := func(context.Context) (time.Time, error) {
		err := answers[min(int(calls.Add(1))-1, len(answers)-1)]
		if err != nil {
			return time.Time{}, err
		}

		return expires, nil
	}

	synctest.Test(t, func(t *testing.T) {
		var logs bytes.Buffer

		poller := github.NewExpiryPoller(expiry, slog.New(slog.NewTextHandler(&logs, nil)))
		go poller.Run(t.Context())

		for _, test := range []struct {
			wait    time.Duration
			expires time.Time
			failure string
		}{
			{0, time.Time{}, failure},
			{interval, expires, ""},
			{interval, expires, failure},
		} {
			time.Sleep(test.wait)
			synctest.Wait()

			got, err := poller.Latest(t.Context())
			if err != nil || !got.Found.Expires.Equal(test.expires) || got.Failure != test.failure {
				t.Errorf("at %s: got %+v and %v, want %v and the failure %q", time.Now().UTC(),
					got, err, test.expires, test.failure)
			}
		}

		want := `level=ERROR msg="poll GitHub token" err="` + failure + `"`
		if lines := strings.Split(strings.TrimSuffix(logs.String(), "\n"), "\n"); len(lines) != 2 ||
			!strings.HasSuffix(lines[0], want) || !strings.HasSuffix(lines[1], want) {
			t.Errorf("got the logs %q, want two lines that each end %q", logs.String(), want)
		}
	})
}
