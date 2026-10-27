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

func TestFetchFailsOnABadAnswer(t *testing.T) {
	// An answer that is valid JSON and size bytes long.
	padded := func(size int) string {
		const start, end = `{"issues":[],"pad":"`, `"}`

		return start + strings.Repeat("x", size-len(start)-len(end)) + end
	}

	for _, test := range []struct {
		name, body string
		status     int
		want       string
	}{
		{"401", "", http.StatusUnauthorized, "fetch Jira tickets: 401 Unauthorized"},
		{
			"400", `{"errorMessages":["Too many issues"]}`, http.StatusBadRequest,
			"fetch Jira tickets: 400 Bad Request",
		},
		{"204", "", http.StatusNoContent, "fetch Jira tickets: 204 No Content"},
		{"not JSON", "<html>Sign in</html>", http.StatusOK, "read Jira's answer: "},
		{"over 4 MiB", padded(4<<20 + 1), http.StatusOK, "read Jira's answer: "},
		{"4 MiB", padded(4 << 20), http.StatusOK, ""},
	} {
		client := clientOf(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(test.status)
			_, _ = w.Write([]byte(test.body))
		})

		got, err := client.Fetch(t.Context(), []string{"WEB-3"})
		if test.want == "" {
			if err != nil || got == nil || len(got) != 0 {
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

func TestFetchFailsWhenALaterBatchFails(t *testing.T) {
	requests := 0
	client := clientOf(t, func(w http.ResponseWriter, _ *http.Request) {
		requests++
		if requests > 1 {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		answer(t, w, map[string]any{"issues": []map[string]any{issue("WEB-1", "", "", "done")}})
	})

	keys := make([]string, 101)
	for i := range keys {
		keys[i] = "WEB-1"
	}

	got, err := client.Fetch(t.Context(), keys)
	if got != nil || err == nil || err.Error() != "fetch Jira tickets: 500 Internal Server Error" {
		t.Errorf("got %v, %v, want the second batch's error", got, err)
	}
}

func TestFetchFailsWithoutJira(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()

	client := jira.New(jira.Settings{
		Site: parse(t, "https://acme.example"), API: parse(t, server.URL),
		Email: email, Token: token,
	})

	got, err := client.Fetch(t.Context(), []string{"WEB-3"})
	if got != nil || err == nil || !strings.HasPrefix(err.Error(), "fetch Jira tickets: ") ||
		strings.Contains(err.Error(), token) {
		t.Errorf("got %v, %v, want an error that names the fetch", got, err)
	}
}

func TestFetchSendsNothingItCantSend(t *testing.T) {
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()

	for _, test := range []struct {
		name string
		ctx  context.Context
		key  string
		want func(error) bool
	}{
		{
			"a key that isn't UTF-8", t.Context(), "WEB-\xff",
			func(err error) bool {
				return err != nil && strings.HasPrefix(err.Error(), "fetch Jira tickets: ")
			},
		},
		{
			"an ended context", cancelled, "WEB-3",
			func(err error) bool { return errors.Is(err, context.Canceled) },
		},
	} {
		requests := 0
		client := clientOf(t, func(_ http.ResponseWriter, _ *http.Request) { requests++ })

		got, err := client.Fetch(test.ctx, []string{test.key})
		if got != nil || !test.want(err) || requests != 0 {
			t.Errorf("%s: got %v, %v after %d requests, want an error before any", test.name,
				got, err, requests)
		}
	}
}
