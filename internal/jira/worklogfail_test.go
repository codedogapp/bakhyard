package jira_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/internal/jira"
)

func TestWorklogsFailOnABadAnswer(t *testing.T) {
	for _, test := range []struct {
		name, path, body string
		status           int
		want             string
	}{
		{
			"account 401", "myself", "", http.StatusUnauthorized,
			"read the Jira account: 401 Unauthorized",
		},
		{
			"no account ID", "myself", `{"displayName":"Me"}`, http.StatusOK,
			"read Jira's answer: no account ID",
		},
		{
			"blank account ID", "myself", `{"accountId":""}`, http.StatusOK,
			"read Jira's answer: no account ID",
		},
		{
			"account not JSON", "myself", "<html>Sign in</html>", http.StatusOK,
			"read Jira's answer: ",
		},
		{
			"search 500", "search/jql", `{"errorMessages":["Broke"]}`,
			http.StatusInternalServerError, "search Jira: 500 Internal Server Error",
		},
		{
			"worklogs 404", "issue/1/worklog", "", http.StatusNotFound,
			"read Jira worklogs: 404 Not Found",
		},
		{
			"worklogs not JSON", "issue/1/worklog", `{"worklogs":[`, http.StatusOK,
			"read Jira's answer: ",
		},
	} {
		fake := &fakeJira{
			t:        t,
			pages:    map[string]map[string]any{"": last(found("1"))},
			worklogs: map[string]map[string]any{"1@0": worklogs(0)},
		}
		client := clientOf(t, func(w http.ResponseWriter, r *http.Request) {
			if pathOf(r) != test.path {
				fake.ServeHTTP(w, r)

				return
			}

			w.WriteHeader(test.status)
			_, _ = w.Write([]byte(test.body))
		})
		start, end := aWeek(t)

		got, err := client.Worklogs(t.Context(), start, end)
		if got != nil || err == nil || !strings.HasPrefix(err.Error(), test.want) ||
			strings.Contains(err.Error(), token) {
			t.Errorf("%s: got %v, %v, want an error that starts with %q", test.name, got, err,
				test.want)
		}
	}
}

func TestWorklogsReadOnlyMyWorklogsStarts(t *testing.T) {
	for _, test := range []struct {
		author, want string
	}{
		{myAccount, `read Jira's answer: parsing time "2026-10-26T09:00:00+09:00"`},
		{otherAccount, ""},
	} {
		fake := &fakeJira{
			t:     t,
			pages: map[string]map[string]any{"": last(found("1"))},
			worklogs: map[string]map[string]any{
				"1@0": worklogs(1, logged(test.author, "2026-10-26T09:00:00+09:00", 60)),
			},
		}
		start, end := aWeek(t)

		got, err := clientOf(t, fake.ServeHTTP).Worklogs(t.Context(), start, end)
		if test.want == "" {
			if err != nil || len(got) != 0 {
				t.Errorf("%s: got %v, %v, want no worklogs and no error", test.author, got, err)
			}

			continue
		}

		if got != nil || err == nil || !strings.HasPrefix(err.Error(), test.want) {
			t.Errorf("%s: got %v, %v, want an error that starts with %q", test.author, got, err,
				test.want)
		}
	}
}

func TestWorklogsFailWithoutJira(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()

	client := jira.New(jira.Settings{
		Site: parse(t, "https://acme.example"), API: parse(t, server.URL),
		Email: email, Token: token,
	})
	start, end := aWeek(t)

	got, err := client.Worklogs(t.Context(), start, end)
	if got != nil || err == nil || !strings.HasPrefix(err.Error(), "read the Jira account: ") ||
		strings.Contains(err.Error(), token) {
		t.Errorf("got %v, %v, want an error that names the account", got, err)
	}
}

func TestWorklogsStopWhenTheirContextEnds(t *testing.T) {
	requests := 0
	client := clientOf(t, func(_ http.ResponseWriter, _ *http.Request) { requests++ })

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	start, end := aWeek(t)

	got, err := client.Worklogs(ctx, start, end)
	if got != nil || !errors.Is(err, context.Canceled) || requests != 0 {
		t.Errorf("got %v, %v after %d requests, want context.Canceled before any", got, err,
			requests)
	}
}
