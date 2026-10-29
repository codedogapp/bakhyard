package jira_test

import (
	"net/http"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/codedogapp/bakhyard/internal/jira"
)

func TestWorklogsSkipAndForgetATicketItAddedToThatJiraDoesntFind(t *testing.T) {
	fake := web9(t, "10001")
	client := clientOf(t, fake.ServeHTTP)
	addTo(t, client, "WEB-9", "2026-10-27T09:00:00+09:00")

	fake.failing = map[string]int{"issue/10009/worklog": http.StatusNotFound}

	got, trail := poll(t, client, fake)
	if want := []jira.Worklog{
		{Started: at(t, "2026-10-28T10:00:00+09:00"), Spent: 30 * time.Minute},
	}; !same(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if want := trailOf("10001", "10009"); !slices.Equal(trail, want) {
		t.Errorf("asked for %q, want %q", trail, want)
	}

	fake.failing = nil

	_, trail = poll(t, client, fake)
	if want := trailOf("10001"); !slices.Equal(trail, want) {
		t.Errorf("then asked for %q, want %q", trail, want)
	}
}

func TestWorklogsFailAndRememberATicketItAddedToThatFails(t *testing.T) {
	fake := web9(t)
	client := clientOf(t, fake.ServeHTTP)
	addTo(t, client, "WEB-9", "2026-10-27T09:00:00+09:00")

	fake.failing = map[string]int{"issue/10009/worklog": http.StatusInternalServerError}
	start, end := aWeek(t)

	_, err := client.Worklogs(t.Context(), start, end)
	if want := "read Jira worklogs: 500 Internal Server Error"; err == nil || err.Error() != want {
		t.Errorf("got the error %v, want %q", err, want)
	}

	fake.failing = nil

	got, _ := poll(t, client, fake)
	if want := []jira.Worklog{
		{Started: at(t, "2026-10-27T09:00:00+09:00"), Spent: time.Hour},
	}; !same(got, want) {
		t.Errorf("then got %v, want %v", got, want)
	}
}

func TestAddWorklogRemembersNothingFromAnAnswerWithoutTheTicket(t *testing.T) {
	bodies := map[string]string{
		"no body":       "",
		"cut short":     `{"issueId":"10009"`,
		"an array":      `["10009"]`,
		"null":          "null",
		"no ID":         `{"id":"100099"}`,
		"an empty ID":   `{"issueId":""}`,
		"a numeric ID":  `{"issueId":10009}`,
		"a misnamed ID": `{"issueID":"10009"}`,
	}
	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			fake := web9(t)
			fake.added["WEB-9"] = reply{http.StatusCreated, body}
			client := clientOf(t, fake.ServeHTTP)
			addTo(t, client, "WEB-9", "2026-10-27T09:00:00+09:00")

			_, trail := poll(t, client, fake)
			if want := trailOf(); !slices.Equal(trail, want) {
				t.Errorf("asked for %q, want %q", trail, want)
			}
		})
	}
}

func TestAddWorklogRemembersNothingWhenJiraRefusesIt(t *testing.T) {
	fake := web9(t)
	fake.added["WEB-9"] = reply{http.StatusBadRequest, `{"issueId":"10009"}`}
	client := clientOf(t, fake.ServeHTTP)

	err := client.AddWorklog(t.Context(), "WEB-9", jira.Worklog{
		Started: at(t, "2026-10-27T09:00:00+09:00"), Spent: time.Hour,
	})
	if err == nil {
		t.Fatal("got no error")
	}

	_, trail := poll(t, client, fake)
	if want := trailOf(); !slices.Equal(trail, want) {
		t.Errorf("asked for %q, want %q", trail, want)
	}
}

func TestWorklogsAndAddWorklogRunTogether(t *testing.T) {
	const tickets = 20

	fake := web9(t)
	client := clientOf(t, fake.ServeHTTP)
	worklog := jira.Worklog{Started: at(t, "2026-10-27T09:00:00+09:00"), Spent: time.Minute}

	var keys []string

	// Every other ticket is gone, so the reads forget tickets while the adds remember others.
	fake.failing = map[string]int{}

	for index := range tickets {
		issueID := strconv.Itoa(10101 + index)
		keys = append(keys, "WEB-"+issueID)
		fake.added["WEB-"+issueID] = createdOn(issueID)
		fake.worklogs[issueID+"@0"] = worklogs(
			1, logged(myAccount, "2026-10-27T09:00:00.000+0900", 60),
		)

		if index%2 == 1 {
			fake.failing["issue/"+issueID+"/worklog"] = http.StatusNotFound
		}
	}

	var adds sync.WaitGroup
	for _, key := range keys {
		adds.Go(func() {
			err := client.AddWorklog(t.Context(), key, worklog)
			if err != nil {
				t.Error(err)
			}
		})
	}

	start, end := aWeek(t)
	for range tickets {
		_, err := client.Worklogs(t.Context(), start, end)
		if err != nil {
			t.Error(err)
		}
	}

	adds.Wait()

	got, _ := poll(t, client, fake)
	if want := tickets / 2; len(got) != want {
		t.Errorf("got %d worklogs, want %d", len(got), want)
	}
}
