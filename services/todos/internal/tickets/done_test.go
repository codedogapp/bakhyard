package tickets_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/codedogapp/bakhyard/internal/jira"
)

// webThreeJSON is webThree as JSON.
const webThreeJSON = `{"key":"WEB-3","summary":"Ship WEB-3","status":"Closed",` +
	`"category":"done","url":"https://acme.example/browse/WEB-3"}`

func TestPollerFindsTheDoneTicketsThatTheTodosName(t *testing.T) {
	open := `{"tickets":[` + payOneJSON + `],`
	searched := `"searched":"2000-01-01T00:00:00Z"}`

	for _, test := range []struct {
		name    string
		keys    []string
		fetched []jira.Ticket
		asked   []string
		want    string
	}{
		{"no keys", []string{}, nil, nil, open + searched},
		{"open keys", []string{"PAY-1"}, nil, nil, open + searched},
		{
			"keys of each category",
			[]string{"PAY-1", "WEB-1", "WEB-2", "WEB-3", "WEB-4", "WEB-5"},
			[]jira.Ticket{
				ticketIn("WEB-1", jira.ToDo), ticketIn("WEB-2", jira.InProgress), webThree(),
				ticketIn("WEB-4", jira.Unknown),
			},
			[]string{"WEB-1", "WEB-2", "WEB-3", "WEB-4", "WEB-5"},
			open + `"done":[` + webThreeJSON + `],` + searched,
		},
	} {
		synctest.Test(t, func(t *testing.T) {
			fake := &fakeJira{
				replies: []reply{found(payOne())},
				keys:    func(context.Context) ([]string, error) { return test.keys, nil },
				fetches: func(context.Context, []string) ([]jira.Ticket, error) {
					return test.fetched, nil
				},
			}
			poller, logs := pollingWith(t.Context(), fake)

			got, asked := latest(t, poller), fake.fetched()
			if got != test.want || !slices.Equal(asked, test.asked) || logs.Len() != 0 {
				t.Errorf("%s: got %s after fetching %q, with the logs %q, want %s after %q",
					test.name, got, asked, logs, test.want, test.asked)
			}
		})
	}
}

func TestPollerKeepsWhatItFoundWhenTheTodosOrAFetchFail(t *testing.T) {
	before := `{"tickets":[` + payOneJSON + `],"done":[` + webThreeJSON + `],` +
		`"searched":"2000-01-01T00:00:00Z"`

	for _, test := range []struct {
		failure string
		inFetch bool
	}{
		{"read the linked tickets: no database", false},
		{"fetch Jira tickets: 500 Internal Server Error", true},
	} {
		synctest.Test(t, func(t *testing.T) {
			looks, fetches := 0, 0
			fake := &fakeJira{
				replies: []reply{found(payOne()), found(payOne())},
				keys: func(context.Context) ([]string, error) {
					looks++
					if !test.inFetch && looks > 1 {
						return nil, errors.New(test.failure)
					}

					return []string{"WEB-3"}, nil
				},
				fetches: func(context.Context, []string) ([]jira.Ticket, error) {
					fetches++
					if test.inFetch && fetches > 1 {
						return nil, errors.New(test.failure)
					}

					return []jira.Ticket{webThree()}, nil
				},
			}
			poller, logs := pollingWith(t.Context(), fake)

			first := latest(t, poller)

			time.Sleep(interval)
			synctest.Wait()

			got, want := latest(t, poller), before+`,"failure":"`+test.failure+`"}`
			line := `level=ERROR msg="poll Jira" err="` + test.failure + `"` + "\n"

			if first != before+"}" || got != want || !strings.HasSuffix(logs.String(), line) ||
				strings.Count(logs.String(), "\n") != 1 {
				t.Errorf("got %s then %s, with the logs %q, want %s} then %s, with %q", first, got,
					logs, before, want, line)
			}
		})
	}
}

func TestPollerIgnoresALookCutShortByStopping(t *testing.T) {
	for _, inFetch := range []bool{false, true} {
		synctest.Test(t, func(t *testing.T) {
			ctx, stop := context.WithCancel(t.Context())
			fake := &fakeJira{
				replies: []reply{found(payOne())},
				keys: func(ctx context.Context) ([]string, error) {
					if inFetch {
						return []string{"WEB-3"}, nil
					}

					<-ctx.Done()

					return nil, fmt.Errorf("read the linked tickets: %w", ctx.Err())
				},
				fetches: func(ctx context.Context, _ []string) ([]jira.Ticket, error) {
					<-ctx.Done()

					return nil, fmt.Errorf("fetch Jira tickets: %w", ctx.Err())
				},
			}
			poller, logs := pollingWith(ctx, fake)

			synctest.Wait()
			stop()
			synctest.Wait()

			if got := latest(t, poller); got != `{}` || logs.Len() != 0 {
				t.Errorf("cut in the fetch %t: got %s, with the logs %q, want {}", inFetch, got,
					logs)
			}
		})
	}
}

// ticketIn returns the ticket key, in a status of category.
func ticketIn(key string, category jira.Category) jira.Ticket {
	return jira.Ticket{
		Key: key, Summary: "Ship " + key, Status: string(category), Category: category,
		URL: "https://acme.example/browse/" + key,
	}
}

func webThree() jira.Ticket {
	ticket := ticketIn("WEB-3", jira.Done)
	ticket.Status = "Closed"

	return ticket
}
