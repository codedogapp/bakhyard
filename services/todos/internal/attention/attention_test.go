package attention_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"testing"
	"testing/synctest"

	attentionv1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1"
	"github.com/codedogapp/bakhyard/services/todos/internal/attention"
	"github.com/codedogapp/bakhyard/services/todos/internal/jira"
	"github.com/codedogapp/bakhyard/services/todos/internal/todo"
)

// webNine is a ticket that is done in Jira.
var webNine = jira.Ticket{
	Key:      "WEB-9",
	Summary:  "Fix the login",
	Status:   "Done",
	Category: jira.Done,
}

// someTodos are todos of every kind: High or not, with a ticket that is done, open or neither,
// and done or not, one of them done with a ticket that no other todo names.
var someTodos = []todo.Todo{
	{ID: 1, Title: "Call the bank", Priority: todo.High},
	{ID: 2, Title: "Fix the login", Priority: todo.Normal, Ticket: "WEB-9"},
	{ID: 3, Title: "Ship the export", Priority: todo.High, Ticket: "WEB-7"},
	{ID: 4, Title: "Write the notes", Priority: todo.Normal},
	{ID: 5, Title: "Review the plan", Priority: todo.Low, Ticket: "WEB-8"},
	{ID: 6, Title: "Pay the rent", Priority: todo.High, Done: true},
	{ID: 7, Title: "Close the sprint", Priority: todo.Low, Ticket: "WEB-9", Done: true},
	{ID: 8, Title: "Rotate the keys", Priority: todo.Normal, Ticket: "OPS-2", Done: true},
	{ID: 12, Title: "Deploy the fix", Priority: todo.High, Ticket: "WEB-9"},
}

// The items of someTodos: the High ones and those whose ticket is done, unless they're done.
const (
	callTheBank   = "1 | Call the bank | High priority | no source | no due"
	fixTheLogin   = "2 | Fix the login | WEB-9 · Done in Jira | SOURCE_JIRA | no due"
	shipTheExport = "3 | Ship the export | WEB-7 · High priority | SOURCE_JIRA | no due"
	deployTheFix  = "12 | Deploy the fix | WEB-9 · Done in Jira | SOURCE_JIRA | no due"
	deployHigh    = "12 | Deploy the fix | WEB-9 · High priority | SOURCE_JIRA | no due"
)

func TestListItemsListsTheTodosThatNeedMe(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tickets := started(t, fakeJira{done: []jira.Ticket{webNine}})

		got := list(t, todosOf(someTodos...), unread(t), tickets)

		want := []string{callTheBank, fixTheLogin, shipTheExport, deployTheFix}
		if !slices.Equal(got, want) {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestListItemsListsNoneWithoutTodos(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tickets := started(t, fakeJira{done: []jira.Ticket{webNine}})

		if got := list(t, todosOf(), unread(t), tickets); len(got) != 0 {
			t.Errorf("got %q, want no items", got)
		}
	})
}

func TestListItemsListsTheHighTodosWithoutJira(t *testing.T) {
	got := list(t, todosOf(someTodos...), unread(t), nil)

	want := []string{callTheBank, shipTheExport, deployHigh}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestListItemsFailsWhenTheTodosDoNotLoad(t *testing.T) {
	down := errors.New("the database is down")
	todos := func(context.Context) ([]todo.Todo, error) { return nil, down }

	res, err := attention.NewLister(todos, unread(t), nil).ListItems(t.Context(),
		&attentionv1.ListItemsRequest{})
	if !errors.Is(err, down) || res != nil {
		t.Errorf("got %v and %v, want no answer and the database's error", res, err)
	}
}

// call is the key of a value that marks a call's context.
type call struct{}

func TestListItemsReadsWithinTheCall(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tickets := started(t, fakeJira{search: searching(payOne)})
		within := func(ctx context.Context, what string) {
			if ctx.Value(call{}) == nil {
				t.Errorf("read the %s outside the call's context", what)
			}
		}
		todos := func(ctx context.Context) ([]todo.Todo, error) {
			within(ctx, "todos")

			return nil, nil
		}
		dismissed := func(ctx context.Context) ([]string, error) {
			within(ctx, "dismissed tickets")

			return nil, nil
		}

		res, err := attention.NewLister(todos, dismissed, tickets).ListItems(
			context.WithValue(t.Context(), call{}, true), &attentionv1.ListItemsRequest{})
		if got := lines(res); err != nil || !slices.Equal(got, []string{payTheInvoice}) {
			t.Errorf("got %q and %v, want PAY-1's item", got, err)
		}
	})
}

// fakeJira is a fake Jira, whose searches answer with search, or find nothing when it's nil, and
// whose fetches find done, whatever the keys.
type fakeJira struct {
	search func(context.Context) ([]jira.Ticket, error)
	done   []jira.Ticket
}

func (f fakeJira) Search(ctx context.Context) ([]jira.Ticket, error) {
	if f.search == nil {
		return nil, nil
	}

	return f.search(ctx)
}

func (f fakeJira) Fetch(context.Context, []string) ([]jira.Ticket, error) {
	return f.done, nil
}

// started returns a poller of finder, and runs it until the test ends.
func started(t *testing.T, finder jira.Finder) *jira.Poller {
	t.Helper()

	none := func(context.Context) ([]string, error) { return nil, nil }

	poller := jira.NewPoller(finder, none, slog.New(slog.DiscardHandler))
	go poller.Run(t.Context())

	return poller
}

// todosOf returns a reader of todos, in their order.
func todosOf(todos ...todo.Todo) func(context.Context) ([]todo.Todo, error) {
	return func(context.Context) ([]todo.Todo, error) { return todos, nil }
}

// list returns the lines of the items that a lister of todos and of the dismissed tickets lists
// with what tickets has found.
func list(
	t *testing.T,
	todos func(context.Context) ([]todo.Todo, error),
	dismissed func(context.Context) ([]string, error),
	tickets *jira.Poller,
) []string {
	t.Helper()

	res, err := attention.NewLister(todos, dismissed, tickets).ListItems(t.Context(),
		&attentionv1.ListItemsRequest{})
	if err != nil {
		t.Fatal(err)
	}

	return lines(res)
}

// lines returns the items of res one line each: their ID, title, detail, source and due time.
func lines(res *attentionv1.ListItemsResponse) []string {
	var lines []string

	for _, item := range res.GetItems() {
		source, due := "no source", "no due"
		if item.HasSource() {
			source = item.GetSource().String()
		}

		if item.HasDue() {
			due = item.GetDue().AsTime().String()
		}

		lines = append(lines, fmt.Sprintf("%s | %s | %s | %s | %s", item.GetId(),
			item.GetTitle(), item.GetDetail(), source, due))
	}

	return lines
}
