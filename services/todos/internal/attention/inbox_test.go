package attention_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"testing/synctest"

	attentionv1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1"
	"github.com/codedogapp/bakhyard/internal/jira"
	"github.com/codedogapp/bakhyard/services/todos/internal/attention"
)

// Open tickets of the user's: PAY-1 and DOC-5, which no todo names, WEB-7, which a todo left to do
// names, OPS-2, which only a done todo names, and APP-4, which the user has dismissed.
var (
	payOne = jira.Ticket{
		Key: "PAY-1", Summary: "Pay the invoice", Status: "In Progress", Category: jira.InProgress,
	}
	webSeven = jira.Ticket{Key: "WEB-7", Summary: "Ship it", Status: "To Do", Category: jira.ToDo}
	opsTwo   = jira.Ticket{Key: "OPS-2", Summary: "Rotate", Status: "To Do", Category: jira.ToDo}
	appFour  = jira.Ticket{Key: "APP-4", Summary: "Icons", Status: "To Do", Category: jira.ToDo}
	docFive  = jira.Ticket{Key: "DOC-5", Summary: "Write the guide", Category: jira.ToDo}
)

// The items of the inbox's tickets.
const (
	payTheInvoice = "PAY-1 | Pay the invoice | PAY-1 · In the Jira inbox | SOURCE_JIRA | no due"
	writeTheGuide = "DOC-5 | Write the guide | DOC-5 · In the Jira inbox | SOURCE_JIRA | no due"
)

func TestListItemsListsTheInboxBeforeTheTodos(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tickets := started(t, fakeJira{
			search: searching(payOne, webSeven, opsTwo, appFour, docFive),
			done:   []jira.Ticket{webNine},
		})

		got := list(t, todosOf(someTodos...), dismissedOf("APP-4", "OLD-1"), tickets)

		want := []string{
			payTheInvoice, writeTheGuide, callTheBank, fixTheLogin, shipTheExport, deployTheFix,
		}
		if !slices.Equal(got, want) {
			t.Errorf("got %q, want %q", got, want)
		}

		latest, err := tickets.Latest(t.Context())

		found := []jira.Ticket{payOne, webSeven, opsTwo, appFour, docFive}
		if err != nil || !slices.Equal(latest.Found.Tickets, found) {
			t.Errorf("the poller holds %v and %v, want the tickets as found", latest.Found.Tickets,
				err)
		}
	})
}

func TestListItemsFailsWhenTheDismissalsDoNotLoad(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tickets := started(t, fakeJira{search: searching(payOne)})
		down := errors.New("the database is down")
		dismissed := func(context.Context) ([]string, error) { return nil, down }

		res, err := attention.NewLister(todosOf(someTodos...), dismissed, tickets).ListItems(
			t.Context(), &attentionv1.ListItemsRequest{})
		if !errors.Is(err, down) || res != nil {
			t.Errorf("got %v and %v, want no answer and the database's error", res, err)
		}
	})
}

// searching returns a search of Jira that finds tickets.
func searching(tickets ...jira.Ticket) func(context.Context) ([]jira.Ticket, error) {
	return func(context.Context) ([]jira.Ticket, error) { return tickets, nil }
}

// dismissedOf returns a reader of the dismissed tickets' keys.
func dismissedOf(keys ...string) func(context.Context) ([]string, error) {
	return func(context.Context) ([]string, error) { return keys, nil }
}

// unread returns a reader of the dismissed tickets that fails the test when it reads them.
func unread(t *testing.T) func(context.Context) ([]string, error) {
	return func(context.Context) ([]string, error) {
		t.Error("read the dismissed tickets")

		return nil, nil
	}
}
