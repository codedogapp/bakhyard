package api_test

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/codedogapp/bakhyard/services/time/internal/api"
)

// The tests' clock starts at 2000-01-01 00:00 UTC, the start of a Saturday.

// tokyo is 9 hours ahead of UTC, so the tests' first day there starts at 09:00.
var tokyo = time.FixedZone("Tokyo", 9*60*60)

func TestAddingAWorklogAddsItAt9InTimesZone(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		jira := &standIn{}

		rec := newTimeAPI(jira.backends(tokyo), tokyo).post(t.Context(),
			`{"ticket":"WEB-9","date":"2000-01-01","minutes":90}`)

		if rec.Code != http.StatusCreated || rec.Body.Len() != 0 {
			t.Fatalf("got %d %q, want 201 and no body", rec.Code, rec.Body)
		}

		keys, added, _ := jira.calls()
		if !slices.Equal(keys, []string{"WEB-9"}) || len(added) != 1 {
			t.Fatalf("added %v to %v, want one worklog to WEB-9", added, keys)
		}

		started := added[0].Started.Format(time.RFC3339)
		if started != "2000-01-01T09:00:00+09:00" || added[0].Spent != 90*time.Minute {
			t.Errorf("added a worklog of %v from %s, want 1h30m from 09:00 in time's zone",
				added[0].Spent, started)
		}
	})
}

func TestAddingAWorklogPollsBeforeAnswering(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		jira := &standIn{}
		backends := jira.backends(tokyo)

		go backends.Reports.Run(t.Context())

		synctest.Wait()

		newTimeAPI(backends, tokyo).post(t.Context(),
			`{"ticket":"WEB-9","date":"2000-01-01","minutes":90}`)

		// Run's first poll, then the add's, both before the answer.
		if _, _, reads := jira.calls(); reads != 2 {
			t.Errorf("got %d reads by the answer, want 2", reads)
		}

		latest, err := backends.Reports.Latest(t.Context())
		if err != nil {
			t.Fatal(err)
		}

		if today := latest.Found.Days[6]; today.Date != "2000-01-01" || today.LoggedHours != 1.5 {
			t.Errorf("got today %+v in the report, want 1.5 hours on 2000-01-01", today)
		}
	})
}

func TestAddingAWorklogTakesTheLast7DaysInTimesZone(t *testing.T) {
	// 2000-01-01 00:00 UTC is 1999-12-31 14:00 in hawaii, so its last 7 days end a day earlier.
	hawaii := time.FixedZone("Hawaii", -10*60*60)

	for _, test := range []struct {
		loc  *time.Location
		date string
		code int
	}{
		{time.UTC, "1999-12-25", http.StatusBadRequest},
		{time.UTC, "1999-12-26", http.StatusCreated},
		{time.UTC, "2000-01-01", http.StatusCreated},
		{time.UTC, "2000-01-02", http.StatusBadRequest},
		{hawaii, "1999-12-24", http.StatusBadRequest},
		{hawaii, "1999-12-25", http.StatusCreated},
		{hawaii, "1999-12-31", http.StatusCreated},
		{hawaii, "2000-01-01", http.StatusBadRequest},
	} {
		synctest.Test(t, func(t *testing.T) {
			jira := &standIn{}

			rec := newTimeAPI(jira.backends(test.loc), test.loc).post(t.Context(),
				`{"ticket":"WEB-9","date":"`+test.date+`","minutes":60}`)

			want := ""
			if test.code == http.StatusBadRequest {
				want = "invalid date: not one of the last 7 days\n"
			}

			if rec.Code != test.code || rec.Body.String() != want {
				t.Errorf("%s in %s: got %d %q, want %d %q", test.date, test.loc, rec.Code,
					rec.Body, test.code, want)
			}
		})
	}
}

func TestAddingAWorklogTakesFrom1To1440Minutes(t *testing.T) {
	for minutes, code := range map[string]int{
		"1": http.StatusCreated, "1440": http.StatusCreated, "0": http.StatusBadRequest,
		"-1": http.StatusBadRequest, "1441": http.StatusBadRequest,
	} {
		synctest.Test(t, func(t *testing.T) {
			jira := &standIn{}

			rec := newTimeAPI(jira.backends(time.UTC), time.UTC).post(t.Context(),
				`{"ticket":"WEB-9","date":"2000-01-01","minutes":`+minutes+`}`)

			want := ""
			if code == http.StatusBadRequest {
				want = "invalid minutes: not from 1 to 1440\n"
			}

			_, added, _ := jira.calls()
			if rec.Code != code || rec.Body.String() != want ||
				(code == http.StatusCreated) != (len(added) == 1) {
				t.Errorf("%s minutes: got %d %q and %d worklogs added, want %d %q", minutes,
					rec.Code, rec.Body, len(added), code, want)
			}
		})
	}
}

func TestAddingAWorklogGoesOnWhenTheClientLeaves(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		jira := &standIn{}
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		rec := newTimeAPI(jira.backends(time.UTC), time.UTC).post(ctx,
			`{"ticket":"WEB-9","date":"2000-01-01","minutes":60}`)

		keys, _, reads := jira.calls()
		if rec.Code != http.StatusCreated || len(keys) != 1 || reads != 1 {
			t.Errorf("got %d, %d adds and %d reads, want 201, 1 add and 1 read", rec.Code,
				len(keys), reads)
		}
	})
}

func TestAddingAWorklogFailsWhenJiraFails(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		jira := &standIn{failure: errors.New("Jira answered 404 Not Found")}
		timeAPI := newTimeAPI(jira.backends(time.UTC), time.UTC)

		rec := timeAPI.post(t.Context(), `{"ticket":"WEB-9","date":"2000-01-01","minutes":60}`)

		if rec.Code != http.StatusBadGateway || rec.Body.String() != "Bad Gateway\n" {
			t.Errorf("got %d %q, want 502 Bad Gateway", rec.Code, rec.Body)
		}

		logs := timeAPI.logs.String()
		if !strings.Contains(logs, `msg="add the worklog" err="Jira answered 404 Not Found"`) {
			t.Errorf("got the logs %q, want the add's failure", logs)
		}

		if _, _, reads := jira.calls(); reads != 0 {
			t.Errorf("got %d reads after a failed add, want none", reads)
		}
	})
}

func TestAddingAWorklogNeedsJira(t *testing.T) {
	rec := newTimeAPI(api.Backends{}, time.UTC).post(t.Context(),
		`{"ticket":"WEB-9","date":"2000-01-01","minutes":60}`)

	if rec.Code != http.StatusNotFound || rec.Body.String() != "Jira is not set up\n" {
		t.Errorf("got %d %q, want 404 Jira is not set up", rec.Code, rec.Body)
	}
}
