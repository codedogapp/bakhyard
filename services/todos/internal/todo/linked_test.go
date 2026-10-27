package todo_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/services/todos/internal/todo"
)

func TestLinkedListsTheTicketsThatTheTodosNotDoneName(t *testing.T) {
	store, pool := newStore(t)
	// ICU's root collation puts punctuation before letters, so A_B-1 would come first. The test
	// database's own collation sorts byte by byte, and couldn't tell.
	_, err := pool.Exec(t.Context(),
		`ALTER TABLE todos ALTER COLUMN jira_key TYPE text COLLATE "und-x-icu"`)
	if err != nil {
		t.Fatal(err)
	}

	// The keys are stored in neither order; AB-10's second todo and OPS-31's only one get ticked.
	for _, addition := range []todo.Addition{
		{Title: "Ship it", Ticket: "WEB-9"},
		{Title: "Review it", Ticket: "AB-9"},
		{Title: "Fix it", Ticket: "A_B-1"},
		{Title: "Ship it again", Ticket: "WEB-9"},
		{Title: "Call home"},
		{Title: "Plan it", Ticket: "AB-10"},
		{Title: "Planned it", Ticket: "AB-10"},
		{Title: "Run it", Ticket: "OPS-31"},
	} {
		if _, err := store.Add(t.Context(), addition); err != nil {
			t.Fatal(err)
		}
	}

	for _, todoID := range []int64{7, 8} {
		if _, err := store.Change(t.Context(), todoID, todo.Change{Done: new(true)}); err != nil {
			t.Fatal(err)
		}
	}

	got, err := store.Linked(t.Context())

	want := []string{"AB-10", "AB-9", "A_B-1", "WEB-9"}
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("got %q (%v), want %q", got, err, want)
	}
}

func TestLinkedListsNothingWithoutLinkedTodos(t *testing.T) {
	store, pool := newStore(t)
	add(t, pool, "Buy milk")

	got, err := store.Linked(t.Context())
	if err != nil || got == nil || len(got) != 0 {
		t.Errorf("got %#v (%v), want an empty list", got, err)
	}
}

func TestLinkedFailsWithoutTheDatabase(t *testing.T) {
	store, pool := newStore(t)
	pool.Close()

	_, err := store.Linked(t.Context())
	if err == nil || !strings.HasPrefix(err.Error(), "read the linked tickets: ") {
		t.Errorf("got %v, want an error that starts with read the linked tickets", err)
	}
}
