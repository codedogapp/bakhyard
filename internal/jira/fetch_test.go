package jira_test

import (
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"slices"
	"testing"

	"github.com/codedogapp/bakhyard/internal/jira"
)

func TestFetchAsksForTheTickets(t *testing.T) {
	var (
		asked *http.Request
		body  []byte
	)

	client := clientOf(t, func(w http.ResponseWriter, r *http.Request) {
		asked = r
		body, _ = io.ReadAll(r.Body)

		// Jira leaves out a key it can't find, answers for a moved ticket under its new key, and
		// may list errors that aren't about any key.
		answer(t, w, map[string]any{
			"issues": []map[string]any{
				issue("WEB-3", "Ship it", "Closed", "done"),
				issue("NEW-7", "Moved", "In Review", "indeterminate"),
			},
			"issueErrors": []map[string]any{{"errorMessages": []string{"Try again"}}},
		})
	})

	got, err := client.Fetch(t.Context(), []string{"WEB-3", "OLD-7", "GONE-1"})
	if err != nil {
		t.Fatal(err)
	}

	want := []jira.Ticket{
		{
			Key: "WEB-3", Summary: "Ship it", Status: "Closed", Category: jira.Done,
			URL: "https://acme.example/browse/WEB-3",
		},
		{
			Key: "NEW-7", Summary: "Moved", Status: "In Review", Category: jira.InProgress,
			URL: "https://acme.example/browse/NEW-7",
		},
	}
	if !slices.Equal(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}

	user, password, ok := asked.BasicAuth()

	for _, check := range []struct{ name, got, want string }{
		{"method", asked.Method, http.MethodPost},
		{"path", asked.URL.Path, apiPath + "/rest/api/3/issue/bulkfetch"},
		{"query", asked.URL.RawQuery, ""},
		{
			"body", string(body),
			`{"issueIdsOrKeys":["WEB-3","OLD-7","GONE-1"],"fields":["summary","status"]}`,
		},
		{"user", user, email},
		{"password", password, token},
		{"Accept", asked.Header.Get("Accept"), "application/json"},
		{"Content-Type", asked.Header.Get("Content-Type"), "application/json"},
	} {
		if check.got != check.want {
			t.Errorf("%s: got %q, want %q", check.name, check.got, check.want)
		}
	}

	if !ok {
		t.Error("got no Basic auth, want it")
	}
}

func TestFetchAsksForAHundredTicketsAtATime(t *testing.T) {
	for _, test := range []struct {
		keys    int
		batches []int
	}{
		{0, nil},
		{1, []int{1}},
		{100, []int{100}},
		{101, []int{100, 1}},
		{250, []int{100, 100, 50}},
	} {
		var batches []int

		client := clientOf(t, func(w http.ResponseWriter, r *http.Request) {
			var asked struct {
				Keys []string `json:"issueIdsOrKeys"`
			}

			err := json.UnmarshalRead(r.Body, &asked)
			if err != nil {
				t.Error(err)
			}

			batches = append(batches, len(asked.Keys))

			issues := []map[string]any{}
			for _, key := range asked.Keys {
				issues = append(issues, issue(key, "Done", "Done", "done"))
			}

			answer(t, w, map[string]any{"issues": issues})
		})

		keys := make([]string, test.keys)
		for i := range keys {
			keys[i] = fmt.Sprintf("WEB-%d", i+1)
		}

		got, err := client.Fetch(t.Context(), keys)
		if err != nil || got == nil || !slices.Equal(keysOf(got), keys) ||
			!slices.Equal(batches, test.batches) {
			t.Errorf("%d keys: got %d tickets, %v in batches of %v, want them all in batches of %v",
				test.keys, len(got), err, batches, test.batches)
		}
	}
}
