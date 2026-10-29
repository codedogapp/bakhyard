package jira_test

import (
	"fmt"
	"slices"
	"testing"
)

func TestWorklogsReadEveryPage(t *testing.T) {
	fake := &fakeJira{
		t: t,
		pages: map[string]map[string]any{
			"":   next("p2", found("1")),
			"p2": last(found("2")),
		},
		worklogs: map[string]map[string]any{
			"1@0": worklogs(3,
				logged(myAccount, "2026-10-26T09:00:00.000+0900", 60),
				logged(myAccount, "2026-10-26T10:00:00.000+0900", 60),
			),
			"1@2": worklogs(3, logged(myAccount, "2026-10-26T11:00:00.000+0900", 60)),
			"2@0": worklogs(1, logged(myAccount, "2026-10-26T12:00:00.000+0900", 60)),
		},
	}
	start, end := aWeek(t)

	got, err := clientOf(t, fake.ServeHTTP).Worklogs(t.Context(), start, end)
	if err != nil || len(got) != 4 {
		t.Errorf("got %v, %v, want four worklogs", got, err)
	}

	want := []string{
		"myself ", "search/jql ", "search/jql p2",
		"issue/1/worklog 0", "issue/1/worklog 2", "issue/2/worklog 0",
	}
	if trail := fake.trail(); !slices.Equal(trail, want) {
		t.Errorf("got the requests %q, want %q", trail, want)
	}
}

func TestWorklogsStopAtTheLastWorklogPage(t *testing.T) {
	mine := logged(myAccount, "2026-10-26T09:00:00.000+0900", 60)

	for _, test := range []struct {
		name string
		page map[string]any
		kept int
	}{
		{"all read", worklogs(2, mine, mine), 2},
		{"more than the total", worklogs(1, mine, mine), 2},
		{"an empty page", worklogs(5), 0},
		{"no worklogs", map[string]any{"worklogs": nil, "total": 3}, 0},
		{"nothing", map[string]any{}, 0},
	} {
		fake := &fakeJira{
			t:        t,
			pages:    map[string]map[string]any{"": last(found("1"))},
			worklogs: map[string]map[string]any{"1@0": test.page},
		}
		start, end := aWeek(t)

		got, err := clientOf(t, fake.ServeHTTP).Worklogs(t.Context(), start, end)
		if err != nil || len(got) != test.kept || len(fake.asked) != 3 {
			t.Errorf("%s: got %v, %v after %d requests, want %d worklogs after three",
				test.name, got, err, len(fake.asked), test.kept)
		}
	}
}

func TestWorklogsGiveUpAfterTenPages(t *testing.T) {
	fake := &fakeJira{
		t:        t,
		pages:    map[string]map[string]any{"": last(found("1"))},
		worklogs: map[string]map[string]any{},
	}
	for i := range 10 {
		fake.worklogs[fmt.Sprintf("1@%d", i)] = worklogs(100,
			logged(myAccount, "2026-10-26T09:00:00.000+0900", 60))
	}

	start, end := aWeek(t)

	got, err := clientOf(t, fake.ServeHTTP).Worklogs(t.Context(), start, end)
	if got != nil || err == nil || err.Error() != "read Jira worklogs: more than 10 pages" ||
		len(fake.asked) != 12 {
		t.Errorf("got %v, %v after %d requests, want an error after 12", got, err,
			len(fake.asked))
	}
}

func TestWorklogsAskForTheTicketByItsID(t *testing.T) {
	fake := &fakeJira{
		t:        t,
		pages:    map[string]map[string]any{"": last(found("1/2"))},
		worklogs: map[string]map[string]any{"1%2F2@0": worklogs(0)},
	}
	start, end := aWeek(t)

	_, err := clientOf(t, fake.ServeHTTP).Worklogs(t.Context(), start, end)
	if err != nil {
		t.Fatal(err)
	}

	got := fake.asked[len(fake.asked)-1].URL.EscapedPath()
	if want := apiPath + "/rest/api/3/issue/1%2F2/worklog"; got != want {
		t.Errorf("got the path %s, want %s", got, want)
	}
}
