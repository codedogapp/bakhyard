package todo_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/codedogapp/bakhyard/services/todos/internal/todo"
)

func TestChangeLinksAndUnlinksATicket(t *testing.T) {
	store, pool := newStore(t)
	add(t, pool, "Buy milk")
	add(t, pool, "Call home")

	for _, test := range []struct {
		name   string
		change todo.Change
		want   todo.Todo
	}{
		{
			"link",
			todo.Change{Ticket: new("ABC-123")},
			todo.Todo{ID: 1, Title: "Buy milk", Priority: todo.Normal, Ticket: "ABC-123"},
		},
		{
			"keep",
			todo.Change{Done: new(true)},
			todo.Todo{ID: 1, Title: "Buy milk", Done: true, Priority: todo.Normal, Ticket: "ABC-123"},
		},
		{
			"relink, with a title and a priority",
			todo.Change{Title: new(" Buy oat milk"), Priority: new(todo.High), Ticket: new("OPS-7")},
			todo.Todo{ID: 1, Title: "Buy oat milk", Done: true, Priority: todo.High, Ticket: "OPS-7"},
		},
		{
			"unlink",
			todo.Change{Ticket: new("")},
			todo.Todo{ID: 1, Title: "Buy oat milk", Done: true, Priority: todo.High},
		},
		{
			"unlink again",
			todo.Change{Done: new(false), Ticket: new("")},
			todo.Todo{ID: 1, Title: "Buy oat milk", Priority: todo.High},
		},
	} {
		got, err := store.Change(t.Context(), 1, test.change)
		if err != nil || got != test.want {
			t.Fatalf("%s: got %+v (%v), want %+v", test.name, got, err, test.want)
		}

		want := []todo.Todo{{ID: 2, Title: "Call home", Priority: todo.Normal}, test.want}
		if got := list(t, store); !slices.Equal(got, want) {
			t.Fatalf("%s: got the list %+v, want %+v", test.name, got, want)
		}
	}

	var none bool

	err := pool.QueryRow(t.Context(), "SELECT jira_key IS NULL FROM todos WHERE id = 1").Scan(&none)
	if err != nil || !none {
		t.Errorf("got jira_key IS NULL %v (%v), want the unlinked todo to store NULL, not ''",
			none, err)
	}
}

func TestChangeTakesAnyJiraKey(t *testing.T) {
	store, pool := newStore(t)
	add(t, pool, "Buy milk")

	for _, ticket := range []string{"AB-1", "ABC-10", "A1-2", "A_B-3", "Z9_-2026", "ABCDEFGHIJ-99999"} {
		got, err := store.Change(t.Context(), 1, todo.Change{Ticket: new(ticket)})
		if err != nil || got.Ticket != ticket {
			t.Errorf("%q: got %+v (%v), want the todo linked to it", ticket, got, err)
		}
	}
}

func TestChangeRefusesAnotherTicket(t *testing.T) {
	store, pool := newStore(t)
	add(t, pool, "Buy milk")

	if _, err := store.Change(t.Context(), 1, todo.Change{Ticket: new("ABC-1")}); err != nil {
		t.Fatal(err)
	}

	before := list(t, store)

	for _, ticket := range []string{
		"abc-1", "Abc-1", "aBC-1", "A-1", "1AB-2", "_AB-2", "AB-0", "AB-01", "AB--1", "AB-1-2",
		"AB_1", "AB-", "AB", "-1", " AB-1", "AB-1 ", "AB-1\n", "AB 1", "ÄB-1", "AB-١", "AB-1١",
		"AB-1x",
	} {
		_, err := store.Change(t.Context(), 1, todo.Change{
			Title:    new("Buy oat milk"),
			Done:     new(true),
			Priority: new(todo.High),
			Ticket:   new(ticket),
		})
		if !errors.Is(err, todo.ErrTicket) ||
			err.Error() != "invalid ticket: not a Jira key such as ABC-123" {
			t.Errorf("%q: got %v, want %v", ticket, err, todo.ErrTicket)
		}
	}

	if got := list(t, store); !slices.Equal(got, before) {
		t.Errorf("got %+v, want the todo unchanged", got)
	}
}

func TestChangeChecksTheTicketAfterThePriorityAndBeforeTheMove(t *testing.T) {
	store, pool := newStore(t)
	add(t, pool, "Buy milk")
	add(t, pool, "Call home")

	for _, test := range []struct {
		name   string
		todoID int64
		change todo.Change
		want   error
	}{
		{"a bad title", 1, todo.Change{Title: new(" "), Ticket: new("abc-1")}, todo.ErrTitle},
		{
			"a bad priority", 1,
			todo.Change{Priority: new(todo.Priority("urgent")), Ticket: new("abc-1")},
			todo.ErrPriority,
		},
		{
			"before and after", 1,
			todo.Change{Ticket: new("abc-1"), Before: new(int64(2)), After: new(int64(2))},
			todo.ErrTicket,
		},
		{
			"a todo that doesn't exist", 3,
			todo.Change{Ticket: new("abc-1"), Before: new(int64(4))},
			todo.ErrTicket,
		},
	} {
		_, err := store.Change(t.Context(), test.todoID, test.change)
		if !errors.Is(err, test.want) {
			t.Errorf("%s: got %v, want %v", test.name, err, test.want)
		}
	}
}
