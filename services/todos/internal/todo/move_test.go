package todo_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/codedogapp/bakhyard/services/todos/internal/todo"
)

func TestChangeMovesATodo(t *testing.T) {
	store, pool := newStore(t)
	for _, title := range []string{"Buy milk", "Answer mail", "Call home", "Pay rent", "Book"} {
		add(t, pool, title)
	}

	// Each move starts from the order that the one before it left.
	for _, test := range []struct {
		name   string
		todoID int64
		change todo.Change
		want   []int64
	}{
		{"up past two, before", 1, todo.Change{Before: new(int64(4))}, []int64{5, 1, 4, 3, 2}},
		{"to the top", 2, todo.Change{Before: new(int64(5))}, []int64{2, 5, 1, 4, 3}},
		{"down past two, after", 5, todo.Change{After: new(int64(4))}, []int64{2, 1, 4, 5, 3}},
		{"to the bottom", 2, todo.Change{After: new(int64(3))}, []int64{1, 4, 5, 3, 2}},
		{"up one, after", 3, todo.Change{After: new(int64(4))}, []int64{1, 4, 3, 5, 2}},
		{"down one, before", 1, todo.Change{Before: new(int64(3))}, []int64{4, 1, 3, 5, 2}},
		{"before itself", 5, todo.Change{Before: new(int64(5))}, []int64{4, 1, 3, 5, 2}},
		{"after itself", 1, todo.Change{After: new(int64(1))}, []int64{4, 1, 3, 5, 2}},
		{"where it is", 1, todo.Change{After: new(int64(4))}, []int64{4, 1, 3, 5, 2}},
		{"raised, in place", 3, todo.Change{Priority: new(todo.High)}, []int64{4, 1, 3, 5, 2}},
	} {
		got, err := store.Change(t.Context(), test.todoID, test.change)
		if err != nil || got.ID != test.todoID {
			t.Fatalf("%s: got %+v (%v), want todo %d", test.name, got, err, test.todoID)
		}

		if ids := order(t, store); !slices.Equal(ids, test.want) {
			t.Fatalf("%s: got the order %v, want %v", test.name, ids, test.want)
		}
	}

	add(t, pool, "Water plants")

	if ids := order(t, store); !slices.Equal(ids, []int64{6, 4, 1, 3, 5, 2}) {
		t.Errorf("got the order %v, want the new todo on top", ids)
	}
}

func TestChangeMovesAndChangesATodoAtOnce(t *testing.T) {
	store, pool := newStore(t)
	add(t, pool, "Buy milk")
	add(t, pool, "Call home")

	got, err := store.Change(t.Context(), 2, todo.Change{
		Title: new(" Call mum "), Done: new(true), Priority: new(todo.Low), After: new(int64(1)),
	})

	want := todo.Todo{ID: 2, Title: "Call mum", Done: true, Priority: todo.Low}
	if err != nil || got != want {
		t.Fatalf("got %+v (%v), want %+v", got, err, want)
	}

	if ids := order(t, store); !slices.Equal(ids, []int64{1, 2}) {
		t.Errorf("got the order %v, want [1 2]", ids)
	}
}

func TestChangeRefusesABadMove(t *testing.T) {
	store, pool := newStore(t)
	for _, title := range []string{"Buy milk", "Answer mail", "Call home"} {
		add(t, pool, title)
	}

	before := list(t, store)

	for _, test := range []struct {
		name         string
		todoID       int64
		change       todo.Change
		want         error
		wantTheError string
	}{
		{
			"before and after", 1,
			todo.Change{Title: new("Buy oat milk"), Before: new(int64(3)), After: new(int64(2))},
			todo.ErrMove, "invalid move: before and after together",
		},
		{
			"before a todo that doesn't exist", 1,
			todo.Change{Title: new("Buy oat milk"), Done: new(true), Before: new(int64(4))},
			todo.ErrNeighbour, "invalid before: no such todo",
		},
		{
			"after a todo that doesn't exist", 3,
			todo.Change{Priority: new(todo.High), After: new(int64(0))},
			todo.ErrNeighbour, "invalid after: no such todo",
		},
		{
			"a todo that doesn't exist, before another", 4,
			todo.Change{Before: new(int64(5))},
			todo.ErrNoTodo, "no such todo",
		},
		{
			"the title's reason first", 1,
			todo.Change{Title: new(" "), Before: new(int64(3)), After: new(int64(2))},
			todo.ErrTitle, "invalid title: empty once trimmed",
		},
	} {
		_, err := store.Change(t.Context(), test.todoID, test.change)
		if !errors.Is(err, test.want) || err.Error() != test.wantTheError {
			t.Errorf("%s: got %v, want %s", test.name, err, test.wantTheError)
		}

		if got := list(t, store); !slices.Equal(got, before) {
			t.Errorf("%s: got %+v, want every todo as it was", test.name, got)
		}
	}
}

func TestChangeChangesNothingWhenTheDatabaseFailsHalfway(t *testing.T) {
	// A check that the change breaks stands for a database that fails while it runs: the first one
	// fails the change of the title, the second the move, which comes after it.
	for _, test := range []struct {
		check  string
		change todo.Change
	}{
		{"title <> 'Call mum'", todo.Change{Title: new("Call mum")}},
		{"position > 0", todo.Change{Title: new("Call mum"), After: new(int64(1))}},
	} {
		store, pool := newStore(t)
		add(t, pool, "Buy milk")
		add(t, pool, "Call home")

		before := list(t, store)

		if _, err := pool.Exec(
			t.Context(),
			"ALTER TABLE todos ADD CHECK ("+test.check+")",
		); err != nil {
			t.Fatal(err)
		}

		_, err := store.Change(t.Context(), 2, test.change)
		if err == nil || !strings.HasPrefix(err.Error(), "change the todo: ") {
			t.Errorf("%s: got %v, want an error that starts with change the todo", test.check, err)
		}

		// The error carries the database's own, for the log.
		if refusal, ok := errors.AsType[*pgconn.PgError](err); !ok || refusal.Code != "23514" {
			t.Errorf("%s: got %v, want the check's violation", test.check, err)
		}

		if got := list(t, store); !slices.Equal(got, before) {
			t.Errorf("%s: got %+v, want every todo as it was", test.check, got)
		}
	}
}

// list returns store's todos in the list's order.
func list(t *testing.T, store *todo.Store) []todo.Todo {
	t.Helper()

	todos, err := store.List(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	return todos
}

// order returns the IDs of store's todos in the list's order.
func order(t *testing.T, store *todo.Store) []int64 {
	t.Helper()

	var ids []int64
	for _, item := range list(t, store) {
		ids = append(ids, item.ID)
	}

	return ids
}
