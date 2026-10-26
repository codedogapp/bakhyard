package jira_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/services/todos/internal/jira"
)

func TestSearchFailsOnABadAnswer(t *testing.T) {
	// A page that is valid JSON and size bytes long.
	padded := func(size int) string {
		const start, end = `{"issues":[],"isLast":true,"pad":"`, `"}`

		return start + strings.Repeat("x", size-len(start)-len(end)) + end
	}

	for _, test := range []struct {
		name, body string
		status     int
		want       string
	}{
		{"401", "", http.StatusUnauthorized, "search Jira: 401 Unauthorized"},
		{"204", "", http.StatusNoContent, "search Jira: 204 No Content"},
		{
			"500", `{"errorMessages":["Something broke"]}`, http.StatusInternalServerError,
			"search Jira: 500 Internal Server Error",
		},
		{"not JSON", "<html>Sign in</html>", http.StatusOK, "read Jira's answer: "},
		{"cut short", `{"issues":[`, http.StatusOK, "read Jira's answer: "},
		{"over 4 MiB", padded(4<<20 + 1), http.StatusOK, "read Jira's answer: "},
		{"4 MiB", padded(4 << 20), http.StatusOK, ""},
	} {
		client := clientOf(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(test.status)
			_, _ = w.Write([]byte(test.body))
		})

		got, err := client.Search(t.Context())
		if test.want == "" {
			if err != nil || len(got) != 0 {
				t.Errorf("%s: got %v, %v, want no tickets and no error", test.name, got, err)
			}

			continue
		}

		if got != nil || err == nil || !strings.HasPrefix(err.Error(), test.want) ||
			strings.Contains(err.Error(), token) {
			t.Errorf("%s: got %v, %v, want an error that starts with %q", test.name, got, err,
				test.want)
		}
	}
}

func TestSearchFailsWithoutJira(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()

	client := jira.New(jira.Settings{
		Site: parse(t, "https://acme.example"), API: parse(t, server.URL),
		Email: email, Token: token,
	})

	got, err := client.Search(t.Context())
	if got != nil || err == nil || !strings.HasPrefix(err.Error(), "search Jira: ") ||
		strings.Contains(err.Error(), token) {
		t.Errorf("got %v, %v, want an error that names the search", got, err)
	}
}

func TestSearchStopsWhenItsContextEnds(t *testing.T) {
	requests := 0
	client := clientOf(t, func(_ http.ResponseWriter, _ *http.Request) { requests++ })

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	got, err := client.Search(ctx)
	if got != nil || !errors.Is(err, context.Canceled) || requests != 0 {
		t.Errorf("got %v, %v after %d requests, want context.Canceled before any", got, err,
			requests)
	}
}
