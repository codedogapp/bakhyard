package jira_test

import (
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/codedogapp/bakhyard/internal/jira"
)

func TestWorklogsReadTheTicketsItAddedToThatTheSearchMisses(t *testing.T) {
	fake := web9(t, "10001")
	client := clientOf(t, fake.ServeHTTP)
	addTo(t, client, "WEB-9", "2026-10-27T09:00:00+09:00")

	want := []jira.Worklog{
		{Started: at(t, "2026-10-27T09:00:00+09:00"), Spent: time.Hour},
		{Started: at(t, "2026-10-28T10:00:00+09:00"), Spent: 30 * time.Minute},
	}

	for range 2 {
		got, trail := poll(t, client, fake)
		if !same(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}

		if want := trailOf("10001", "10009"); !slices.Equal(trail, want) {
			t.Errorf("asked for %q, want %q", trail, want)
		}
	}
}

func TestWorklogsReadTheTicketsItAddedToAfterTheSearchedOnesInIDOrder(t *testing.T) {
	fake := web9(t, "10001")
	client := clientOf(t, fake.ServeHTTP)
	started := at(t, "2026-10-28T10:00:00+09:00")
	want := []jira.Worklog{{Started: started, Spent: 30 * time.Minute}}

	for _, minutes := range []int{5, 3, 7, 4, 2} {
		key, id := "WEB-"+strconv.Itoa(minutes), strconv.Itoa(10000+minutes)
		fake.added[key] = createdOn(id)
		fake.worklogs[id+"@0"] = worklogs(1,
			logged(myAccount, "2026-10-28T10:00:00.000+0900", int64(minutes)*60))

		addTo(t, client, key, "2026-10-28T10:00:00+09:00")
	}

	for _, minutes := range []int{2, 3, 4, 5, 7} {
		spent := time.Duration(minutes) * time.Minute
		want = append(want, jira.Worklog{Started: started, Spent: spent})
	}

	got, trail := poll(t, client, fake)
	if !same(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	read := trailOf("10001", "10002", "10003", "10004", "10005", "10007")
	if !slices.Equal(trail, read) {
		t.Errorf("asked for %q, want %q", trail, read)
	}
}

func TestWorklogsReadTheTicketsItAddedToOnceWhetherTheSearchFindsThemOrNot(t *testing.T) {
	fake := web9(t, "10009")
	client := clientOf(t, fake.ServeHTTP)
	addTo(t, client, "WEB-9", "2026-10-27T09:00:00+09:00")

	want := []jira.Worklog{{Started: at(t, "2026-10-27T09:00:00+09:00"), Spent: time.Hour}}

	got, trail := poll(t, client, fake)
	if !same(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if read := trailOf("10009"); !slices.Equal(trail, read) {
		t.Errorf("asked for %q, want %q", trail, read)
	}

	// The search may have found WEB-9 through an older worklog, and lose it when that worklog
	// leaves its days, before it finds the new one.
	fake.pages[""] = last()

	got, trail = poll(t, client, fake)
	if !same(got, want) {
		t.Errorf("then got %v, want %v", got, want)
	}

	if read := trailOf("10009"); !slices.Equal(trail, read) {
		t.Errorf("then asked for %q, want %q", trail, read)
	}
}

func TestWorklogsForgetTheTicketsItAddedToBeforeTheSpan(t *testing.T) {
	tests := []struct {
		name, started string
		read          []string
	}{
		{"before the span", "2026-10-25T23:59:59.999+09:00", nil},
		{"at its start", "2026-10-26T00:00:00+09:00", []string{"10009"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fake := web9(t)
			client := clientOf(t, fake.ServeHTTP)
			addTo(t, client, "WEB-9", test.started)

			_, trail := poll(t, client, fake)
			if want := trailOf(test.read...); !slices.Equal(trail, want) {
				t.Errorf("asked for %q, want %q", trail, want)
			}

			fake.asked = nil

			_, err := client.Worklogs(t.Context(),
				at(t, "2026-10-19T00:00:00+09:00"), at(t, "2026-10-26T00:00:00+09:00"))
			if err != nil {
				t.Fatal(err)
			}

			if want := trailOf(test.read...); !slices.Equal(fake.trail(), want) {
				t.Errorf("then asked for %q for the week before, want %q", fake.trail(), want)
			}
		})
	}
}

func TestWorklogsKeepTheLatestStartOfTheWorklogsItAdded(t *testing.T) {
	inTheSpan, before := "2026-10-27T09:00:00+09:00", "2026-10-20T09:00:00+09:00"

	tests := map[string][]string{
		"the latest first": {inTheSpan, before},
		"the latest last":  {before, inTheSpan},
	}
	for name, starts := range tests {
		t.Run(name, func(t *testing.T) {
			fake := web9(t)
			client := clientOf(t, fake.ServeHTTP)

			for _, started := range starts {
				addTo(t, client, "WEB-9", started)
			}

			_, trail := poll(t, client, fake)
			if want := trailOf("10009"); !slices.Equal(trail, want) {
				t.Errorf("asked for %q, want %q", trail, want)
			}
		})
	}
}
