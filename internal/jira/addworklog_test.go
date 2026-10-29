package jira_test

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/codedogapp/bakhyard/internal/jira"
)

// worklogPath is where the fake Jira takes WEB-9's worklogs.
const worklogPath = apiPath + "/rest/api/3/issue/WEB-9/worklog"

// added is what the fake Jira got in a request.
type added struct {
	method, path, query, contentType, accept, user, password, body string
}

// adder returns a client whose fake Jira answers every request with status and body, and the
// requests it got.
func adder(t *testing.T, status int, body string) (*jira.Client, *[]added) {
	t.Helper()

	var got []added

	client := clientOf(t, func(w http.ResponseWriter, r *http.Request) {
		sent, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}

		user, password, _ := r.BasicAuth()
		got = append(got, added{
			r.Method, r.URL.Path, r.URL.RawQuery, r.Header.Get("Content-Type"),
			r.Header.Get("Accept"), user, password, string(sent),
		})

		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	})

	return client, &got
}

func zone(t *testing.T, name string) *time.Location {
	t.Helper()

	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}

	return loc
}

func TestAddWorklogPostsTheWorklog(t *testing.T) {
	client, got := adder(t, http.StatusCreated, `{"id":"100028","timeSpentSeconds":5400}`)
	start := time.Date(2026, 10, 26, 9, 0, 0, 0, zone(t, "Europe/Budapest"))

	err := client.AddWorklog(t.Context(), "WEB-9", jira.Worklog{
		Started: start, Spent: 90 * time.Minute,
	})

	want := added{
		http.MethodPost, worklogPath, "", "application/json", "application/json", email, token,
		`{"started":"2026-10-26T09:00:00.000+0100","timeSpentSeconds":5400}`,
	}
	if err != nil || len(*got) != 1 || (*got)[0] != want {
		t.Errorf("got %v and the requests %+v, want no error and one request %+v", err, *got,
			want)
	}
}

func TestAddWorklogSendsTheStartAsItIs(t *testing.T) {
	for _, test := range []struct {
		started time.Time
		spent   time.Duration
		want    string
	}{
		{
			time.Date(2026, 10, 26, 8, 0, 0, 0, time.UTC), time.Minute,
			`{"started":"2026-10-26T08:00:00.000+0000","timeSpentSeconds":60}`,
		},
		{
			time.Date(2026, 7, 1, 9, 0, 0, 0, zone(t, "America/St_Johns")), 24 * time.Hour,
			`{"started":"2026-07-01T09:00:00.000-0230","timeSpentSeconds":86400}`,
		},
		{
			time.Date(2026, 10, 26, 9, 0, 0, 250_900_000, zone(t, "Europe/Budapest")),
			1500 * time.Millisecond,
			`{"started":"2026-10-26T09:00:00.250+0100","timeSpentSeconds":1}`,
		},
	} {
		client, got := adder(t, http.StatusCreated, "")

		err := client.AddWorklog(t.Context(), "WEB-9", jira.Worklog{
			Started: test.started, Spent: test.spent,
		})
		if err != nil || len(*got) != 1 || (*got)[0].body != test.want {
			t.Errorf("%v for %v: got %v and the requests %+v, want the body %s", test.started,
				test.spent, err, *got, test.want)
		}
	}
}
