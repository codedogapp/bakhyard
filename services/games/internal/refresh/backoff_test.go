package refresh_test

import (
	"slices"
	"testing"
	"testing/synctest"
	"time"
)

func TestPollsThatFailInARowWaitLongerAndLonger(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		replies := []reply{finding("A")}

		for range 8 {
			replies = append(replies, failing("Acme is down"))
		}

		replies = append(replies, finding("B"), failing("Acme is down"), failing("Acme is down"))
		onRequest, fake := started(t.Context(), replies...)

		// Someone asks every minute for 18 hours and 45 minutes.
		for range 18*60 + 45 + 1 {
			get(t.Context(), onRequest)
			time.Sleep(time.Minute)
		}

		want := []time.Duration{
			0,         // finds A
			time.Hour, // fails, as do the next seven polls, each waiting twice as long as the last
			time.Hour + 5*time.Minute,
			time.Hour + 15*time.Minute,
			time.Hour + 35*time.Minute,
			2*time.Hour + 15*time.Minute,
			3*time.Hour + 35*time.Minute,
			6*time.Hour + 15*time.Minute,
			11*time.Hour + 35*time.Minute,
			17*time.Hour + 35*time.Minute, // waits no more than 6 hours, and finds B
			18*time.Hour + 35*time.Minute, // fails
			18*time.Hour + 40*time.Minute, // waits 5 minutes again
		}

		got := fake.calls()
		if !slices.EqualFunc(got, want, func(began time.Time, after time.Duration) bool {
			return began.Equal(start.Add(after))
		}) {
			t.Errorf("polled at %v, want after %v", got, want)
		}

		const latest = `{"names":["B"],"searched":"2000-01-01T17:35:00Z","failure":"Acme is down"}`
		if answer := get(t.Context(), onRequest); answer != latest {
			t.Errorf("got %s, want %s", answer, latest)
		}
	})
}
