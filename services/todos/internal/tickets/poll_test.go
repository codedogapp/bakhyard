package tickets_test

import (
	"strings"
	"testing"
	"testing/synctest"
	"time"
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
