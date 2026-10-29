package jira_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/codedogapp/bakhyard/internal/jira"
)

func TestAddWorklogFailsUnlessJiraAddsIt(t *testing.T) {
	for _, status := range []int{
		http.StatusOK, http.StatusNoContent, http.StatusBadRequest, http.StatusUnauthorized,
		http.StatusForbidden, http.StatusNotFound, http.StatusRequestEntityTooLarge,
		http.StatusInternalServerError,
	} {
		client, _ := adder(t, status, `{"errorMessages":["No"],"errors":{}}`)

		err := client.AddWorklog(t.Context(), "WEB-9", jira.Worklog{
			Started: time.Date(2026, 10, 26, 9, 0, 0, 0, time.UTC), Spent: time.Hour,
		})

		want := "add a Jira worklog: " + strconv.Itoa(status) + " " + http.StatusText(status)
		if err == nil || err.Error() != want {
			t.Errorf("%d: got %v, want %s", status, err, want)
		}
	}
}

func TestAddWorklogRefusesWhatIsntAKey(t *testing.T) {
	for _, key := range []string{
		"", ".", "..", "web-9", "WEB-0", "WEB-9/x", " WEB-9", "WEB-9\n", "../../myself",
	} {
		client, got := adder(t, http.StatusCreated, "")

		err := client.AddWorklog(t.Context(), key, jira.Worklog{
			Started: time.Date(2026, 10, 26, 9, 0, 0, 0, time.UTC), Spent: time.Hour,
		})

		want := "add a Jira worklog: " + strconv.Quote(key) + " isn't a Jira key"
		if err == nil || err.Error() != want || len(*got) != 0 {
			t.Errorf("%q: got %v after the requests %+v, want %s before any", key, err, *got,
				want)
		}
	}
}

func TestAddWorklogFailsWithoutJira(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()

	client := jira.New(jira.Settings{
		Site: parse(t, "https://acme.example"), API: parse(t, server.URL),
		Email: email, Token: token,
	})

	err := client.AddWorklog(t.Context(), "WEB-9", jira.Worklog{
		Started: time.Date(2026, 10, 26, 9, 0, 0, 0, time.UTC), Spent: time.Hour,
	})
	if err == nil || !strings.HasPrefix(err.Error(), "add a Jira worklog: ") ||
		strings.Contains(err.Error(), token) {
		t.Errorf("got %v, want an error that names the worklog", err)
	}
}

func TestAddWorklogStopsWhenItsContextEnds(t *testing.T) {
	client, got := adder(t, http.StatusCreated, "")

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := client.AddWorklog(ctx, "WEB-9", jira.Worklog{
		Started: time.Date(2026, 10, 26, 9, 0, 0, 0, time.UTC), Spent: time.Hour,
	})
	if !errors.Is(err, context.Canceled) || len(*got) != 0 {
		t.Errorf("got %v after the requests %+v, want context.Canceled before any", err, *got)
	}
}

func TestAddWorklogFailsOnAnAddressItCantAsk(t *testing.T) {
	api := parse(t, "https://acme.example")
	api.Host = "acme example"

	client := jira.New(jira.Settings{
		Site: parse(t, "https://acme.example"), API: api, Email: email, Token: token,
	})

	err := client.AddWorklog(t.Context(), "WEB-9", jira.Worklog{
		Started: time.Date(2026, 10, 26, 9, 0, 0, 0, time.UTC), Spent: time.Hour,
	})
	if err == nil || !strings.HasPrefix(err.Error(), `add a Jira worklog: parse "https://acme`) {
		t.Errorf("got %v, want an error that names the worklog", err)
	}
}
