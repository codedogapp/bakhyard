package todo_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/services/todos/internal/todo"
)

func TestChangeChangesOnlyWhatItNames(t *testing.T) {
	store, pool := newStore(t)
	add(t, pool, "Buy milk")
	add(t, pool, "Call home")

	for _, test := range []struct {
		name   string
		change todo.Change
		want   todo.Todo
	}{
		{"tick", todo.Change{Done: new(true)}, todo.Todo{ID: 1, Title: "Buy milk", Done: true}},
		{
			"rename, trimmed",
			todo.Change{Title: new("\t Buy oat milk \n")},
			todo.Todo{ID: 1, Title: "Buy oat milk", Done: true},
		},
		{"nothing", todo.Change{}, todo.Todo{ID: 1, Title: "Buy oat milk", Done: true}},
		{
			"both",
			todo.Change{Title: new("Buy milk"), Done: new(false)},
			todo.Todo{ID: 1, Title: "Buy milk"},
		},
	} {
		got, err := store.Change(t.Context(), 1, test.change)
		if err != nil || got != test.want {
			t.Fatalf("%s: got %+v (%v), want %+v", test.name, got, err, test.want)
		}

		list, err := store.List(t.Context())
		want := []todo.Todo{{ID: 2, Title: "Call home"}, test.want}

		if err != nil || !slices.Equal(list, want) {
			t.Fatalf("%s: got the list %+v (%v), want %+v", test.name, list, err, want)
		}
	}
}

func TestChangeRefusesABadTitle(t *testing.T) {
	store, pool := newStore(t)
	add(t, pool, "Buy milk")

	for _, test := range []struct{ name, title, want string }{
		{"blank", " \t\n", "invalid title: empty once trimmed"},
		{"long", strings.Repeat("é", 501), "invalid title: over 500 characters"},
		{"a control character", "Buy\u0007milk", "invalid title: has a control character"},
	} {
		_, err := store.Change(t.Context(), 1, todo.Change{Title: new(test.title), Done: new(true)})
		if !errors.Is(err, todo.ErrTitle) || err.Error() != test.want {
			t.Errorf("%s: got %v, want %s", test.name, err, test.want)
		}
	}

	got, err := store.List(t.Context())
	if want := []todo.Todo{{ID: 1, Title: "Buy milk"}}; err != nil || !slices.Equal(got, want) {
		t.Errorf("got %+v (%v), want the todo unchanged, not done", got, err)
	}
}

func TestChangeAndDeleteFindNoTodo(t *testing.T) {
	store, pool := newStore(t)
	add(t, pool, "Buy milk")

	for _, todoID := range []int64{0, -1, 2} {
		_, err := store.Change(t.Context(), todoID, todo.Change{Done: new(true)})
		if !errors.Is(err, todo.ErrNoTodo) {
			t.Errorf("change todo %d: got %v, want %v", todoID, err, todo.ErrNoTodo)
		}

		err = store.Delete(t.Context(), todoID)
		if !errors.Is(err, todo.ErrNoTodo) {
			t.Errorf("delete todo %d: got %v, want %v", todoID, err, todo.ErrNoTodo)
		}
	}

	got, err := store.List(t.Context())
	if want := []todo.Todo{{ID: 1, Title: "Buy milk"}}; err != nil || !slices.Equal(got, want) {
		t.Errorf("got %+v (%v), want the todo unchanged", got, err)
	}
}

func TestDeleteDeletesOneTodoForGood(t *testing.T) {
	store, pool := newStore(t)
	for _, title := range []string{"Buy milk", "Answer mail", "Call home"} {
		add(t, pool, title)
	}

	if err := store.Delete(t.Context(), 2); err != nil {
		t.Fatal(err)
	}

	got, err := store.List(t.Context())

	want := []todo.Todo{{ID: 3, Title: "Call home"}, {ID: 1, Title: "Buy milk"}}
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("got %+v (%v), want %+v", got, err, want)
	}

	if err := store.Delete(t.Context(), 2); !errors.Is(err, todo.ErrNoTodo) {
		t.Errorf("deleting it again: got %v, want %v", err, todo.ErrNoTodo)
	}
}

func TestChangeAndDeleteFailWithoutTheDatabase(t *testing.T) {
	store, pool := newStore(t)
	add(t, pool, "Buy milk")
	pool.Close()

	_, err := store.Change(t.Context(), 1, todo.Change{Done: new(true)})
	if err == nil || !strings.HasPrefix(err.Error(), "change the todo: ") {
		t.Errorf("got %v, want an error that starts with change the todo", err)
	}

	err = store.Delete(t.Context(), 1)
	if err == nil || !strings.HasPrefix(err.Error(), "delete the todo: ") {
		t.Errorf("got %v, want an error that starts with delete the todo", err)
	}
}
