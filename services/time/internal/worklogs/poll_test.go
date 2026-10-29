package worklogs_test

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/codedogapp/bakhyard/internal/jira"
	"github.com/codedogapp/bakhyard/services/time/internal/worklogs"
)

// interval is the poller's time between two polls. A bubble's clock starts at midnight UTC on
// 1 January 2000, which is 14:00 on Saturday in Kiritimati (UTC+14), the pollers' zone.
const interval = 5 * time.Minute

func TestPollerReadsAtOnceAndEveryFiveMinutes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		fake := &fakeJira{}
		polling(t, fake)

		for _, test := range []struct {
			wait  time.Duration
			reads int
		}{
			{0, 1}, {interval - time.Nanosecond, 1}, {time.Nanosecond, 2}, {interval, 3},
		} {
			time.Sleep(test.wait)
			synctest.Wait()

			if got := len(fake.read()); got != test.reads {
				t.Errorf("at %s: got %d reads, want %d", time.Now().UTC(), got, test.reads)
			}
		}
	})
}

func TestPollerReadsTheLastSevenDaysOfItsZone(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		fake := &fakeJira{}
		poller, _ := polling(t, fake)

		// Sunday, 2 January, starts at 10:00 UTC in Kiritimati.
		for _, test := range []struct {
			wait        time.Duration
			span, first string
		}{
			{0, "1999-12-25T10:00:00Z 2000-01-01T10:00:00Z", "1999-12-26"},
			{10*time.Hour - interval, "1999-12-25T10:00:00Z 2000-01-01T10:00:00Z", "1999-12-26"},
			{interval, "1999-12-26T10:00:00Z 2000-01-02T10:00:00Z", "1999-12-27"},
		} {
			time.Sleep(test.wait)
			synctest.Wait()

			spans := fake.read()
			if got := spans[len(spans)-1]; got != test.span {
				t.Errorf("at %s: got the span %s, want %s", time.Now().UTC(), got, test.span)
			}

			want := `{"days":[{"date":"` + test.first + `",`
			if got := latest(t, poller); !strings.HasPrefix(got, want) {
				t.Errorf("at %s: got %s, want it to start %s", time.Now().UTC(), got, want)
			}
		}
	})
}

func TestPollerWorksTheReportOutInItsZone(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		poller, _ := polling(t, &fakeJira{logged: []jira.Worklog{
			// 00:30 on Thursday, 30 December, in Kiritimati.
			{Started: time.Date(1999, 12, 29, 10, 30, 0, 0, time.UTC), Spent: time.Hour},
			// 12:00 on Saturday, 1 January: today.
			{Started: time.Date(1999, 12, 31, 22, 0, 0, 0, time.UTC), Spent: 90 * time.Minute},
		}})

		synctest.Wait()

		want := `{"days":[` +
			`{"date":"1999-12-26","loggedHours":0,"missing":false},` +
			`{"date":"1999-12-27","loggedHours":0,"missing":true},` +
			`{"date":"1999-12-28","loggedHours":0,"missing":true},` +
			`{"date":"1999-12-29","loggedHours":0,"missing":true},` +
			`{"date":"1999-12-30","loggedHours":1,"missing":false},` +
			`{"date":"1999-12-31","loggedHours":0,"missing":true},` +
			`{"date":"2000-01-01","loggedHours":1.5,"missing":false}],` +
			`"week":{"loggedHours":2.5,"targetHours":40},"searched":"2000-01-01T00:00:00Z"}`
		if got := latest(t, poller); got != want {
			t.Errorf("got %s, want %s", got, want)
		}
	})
}

func TestPollerKeepsWhyAReadFailed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		poller, logs := polling(t, &fakeJira{
			err: errors.New("read the Jira account: 401 Unauthorized"),
		})

		synctest.Wait()

		want := `{"failure":"read the Jira account: 401 Unauthorized"}`
		if got := latest(t, poller); got != want {
			t.Errorf("got %s, want %s", got, want)
		}

		line := `level=ERROR msg="poll Jira worklogs" ` +
			`err="read the Jira account: 401 Unauthorized"` + "\n"
		if !strings.HasSuffix(logs.String(), line) || strings.Count(logs.String(), "\n") != 1 {
			t.Errorf("got the logs %q, want one line that ends %q", logs, line)
		}
	})
}

func TestPollerCutsAReadShortWhenItStops(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cut := make(chan error, 1)
		read := func(ctx context.Context, _, _ time.Time) ([]jira.Worklog, error) {
			<-ctx.Done()

			cut <- ctx.Err()

			return nil, ctx.Err()
		}

		ctx, stop := context.WithCancel(t.Context())
		go worklogs.NewPoller(read, time.UTC, slog.New(slog.DiscardHandler)).Run(ctx)

		synctest.Wait()
		stop()
		synctest.Wait()

		select {
		case err := <-cut:
			if !errors.Is(err, context.Canceled) {
				t.Errorf("got %v, want the read cut short by the stop", err)
			}
		default:
			t.Error("the read goes on after the poller stopped")
		}
	})
}
