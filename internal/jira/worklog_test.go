package jira_test

import (
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/codedogapp/bakhyard/internal/jira"
)

func TestWorklogsReturnsMyWorklogsInTheSpan(t *testing.T) {
	fake := &fakeJira{
		t:     t,
		pages: map[string]map[string]any{"": last(found("10001"), found("10002"))},
		worklogs: map[string]map[string]any{
			"10001@0": worklogs(5,
				logged(myAccount, "2026-10-29T09:00:00.000+0900", 600),
				logged(myAccount, "2026-10-26T00:00:00.000+0900", 3600),
				logged(myAccount, "2026-10-25T23:59:59.999+0900", 60),
				logged(myAccount, "2026-11-02T00:00:00.000+0900", 60),
				logged(otherAccount, "2026-10-27T10:00:00.000+0900", 60),
			),
			"10002@0": worklogs(2,
				logged(myAccount, "2026-10-27T01:30:00.000+0000", 5400),
				logged(myAccount, "2026-10-26T12:00:00.000+0900", 1800),
			),
		},
	}
	start, end := aWeek(t)

	got, err := clientOf(t, fake.ServeHTTP).Worklogs(t.Context(), start, end)
	if err != nil {
		t.Fatal(err)
	}

	want := []jira.Worklog{
		{Started: at(t, "2026-10-26T00:00:00+09:00"), Spent: time.Hour},
		{Started: at(t, "2026-10-26T12:00:00+09:00"), Spent: 30 * time.Minute},
		{Started: at(t, "2026-10-27T01:30:00Z"), Spent: 90 * time.Minute},
		{Started: at(t, "2026-10-29T09:00:00+09:00"), Spent: 10 * time.Minute},
	}
	if !same(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestWorklogsAsksJiraAsTheUser(t *testing.T) {
	fake := &fakeJira{
		t:        t,
		pages:    map[string]map[string]any{"": last(found("10001"))},
		worklogs: map[string]map[string]any{"10001@0": worklogs(0)},
	}
	start, end := aWeek(t)

	_, err := clientOf(t, fake.ServeHTTP).Worklogs(t.Context(), start, end)
	if err != nil {
		t.Fatal(err)
	}

	search := url.Values{
		"jql": {`worklogAuthor = currentUser() AND worklogDate >= "2026-10-24" AND ` +
			`worklogDate < "2026-11-03" ORDER BY key`},
		"fields":     {"id"},
		"maxResults": {"100"},
	}
	wants := []struct{ path, query string }{
		{"/rest/api/3/myself", ""},
		{"/rest/api/3/search/jql", search.Encode()},
		{
			"/rest/api/3/issue/10001/worklog",
			"maxResults=1000&startAt=0&startedAfter=1792940399999&startedBefore=1793545200000",
		},
	}

	if len(fake.asked) != len(wants) {
		t.Fatalf("got %d requests, want %d", len(fake.asked), len(wants))
	}

	for index, want := range wants {
		asked := fake.asked[index]
		user, password, ok := asked.BasicAuth()

		got := fmt.Sprintf("%s %s?%s as %s:%s (%t), Accept %s", asked.Method,
			asked.URL.EscapedPath(), asked.URL.RawQuery, user, password, ok,
			asked.Header.Get("Accept"))
		if wanted := fmt.Sprintf("GET %s?%s as %s:%s (true), Accept application/json",
			apiPath+want.path, want.query, email, token); got != wanted {
			t.Errorf("request %d: got %s, want %s", index, got, wanted)
		}
	}
}

func TestWorklogsSearchesADayMoreOnEachSide(t *testing.T) {
	for _, test := range []struct{ start, end, from, until string }{
		{"2026-10-23T00:00:00+14:00", "2026-10-30T00:00:00+14:00", "2026-10-21", "2026-10-31"},
		{"2026-10-23T00:00:00-12:00", "2026-10-30T00:00:00-12:00", "2026-10-22", "2026-11-01"},
		{"2026-10-26T00:00:00Z", "2026-11-02T00:00:00Z", "2026-10-25", "2026-11-04"},
	} {
		fake := &fakeJira{t: t, pages: map[string]map[string]any{"": last()}}

		_, err := clientOf(t, fake.ServeHTTP).Worklogs(t.Context(), at(t, test.start),
			at(t, test.end))
		if err != nil || len(fake.asked) != 2 {
			t.Fatalf("%s: got %v after %d requests, want two requests", test.start, err,
				len(fake.asked))
		}

		got := fake.asked[1].URL.Query().Get("jql")
		if want := fmt.Sprintf(`worklogAuthor = currentUser() AND worklogDate >= "%s" AND `+
			`worklogDate < "%s" ORDER BY key`, test.from, test.until); got != want {
			t.Errorf("from %s to %s: got %s, want %s", test.start, test.end, got, want)
		}
	}
}

func TestWorklogsThatStartTogetherKeepJirasOrder(t *testing.T) {
	fake := &fakeJira{
		t:        t,
		pages:    map[string]map[string]any{},
		worklogs: map[string]map[string]any{},
	}

	var tickets []map[string]any

	byHour := map[string][]jira.Worklog{}

	// More than 12, and not in order: Go's unstable sort keeps ties in place for 12 or fewer,
	// and for a list that's already sorted.
	for order := range 14 {
		id := strconv.Itoa(order)
		tickets = append(tickets, found(id))

		hour := []string{"10", "09"}[order%2]
		fake.worklogs[id+"@0"] = worklogs(
			1,
			logged(myAccount, "2026-10-26T"+hour+":00:00.000+0900", int64(order)),
		)
		byHour[hour] = append(byHour[hour], jira.Worklog{
			Started: at(t, "2026-10-26T"+hour+":00:00+09:00"),
			Spent:   time.Duration(order) * time.Second,
		})
	}

	fake.pages[""] = last(tickets...)
	start, end := aWeek(t)

	got, err := clientOf(t, fake.ServeHTTP).Worklogs(t.Context(), start, end)
	if want := slices.Concat(byHour["09"], byHour["10"]); err != nil || !same(got, want) {
		t.Errorf("got %v, %v, want each hour's worklogs in the tickets' order", got, err)
	}
}

func TestWorklogsWithoutMineAreEmpty(t *testing.T) {
	for name, fake := range map[string]*fakeJira{
		"no tickets": {pages: map[string]map[string]any{"": last()}},
		"someone else's": {
			pages: map[string]map[string]any{"": last(found("1"))},
			worklogs: map[string]map[string]any{
				"1@0": worklogs(1, logged(otherAccount, "2026-10-26T09:00:00.000+0900", 60)),
			},
		},
	} {
		fake.t = t
		start, end := aWeek(t)

		got, err := clientOf(t, fake.ServeHTTP).Worklogs(t.Context(), start, end)
		if err != nil || got == nil || len(got) != 0 {
			t.Errorf("%s: got %#v, %v, want an empty list", name, got, err)
		}
	}
}
