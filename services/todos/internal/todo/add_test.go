package todo_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/services/todos/internal/todo"
)

func TestAddStoresATrimmedTitle(t *testing.T) {
	store, _ := newStore(t)

	first, err := store.Add(t.Context(), "  Buy milk  ")
	if err != nil || first != (todo.Todo{ID: 1, Title: "Buy milk"}) {
		t.Fatalf("got %+v (%v), want todo 1, Buy milk", first, err)
	}

	second, err := store.Add(t.Context(), "\tCall home\n")
	if err != nil || second != (todo.Todo{ID: 2, Title: "Call home"}) {
		t.Fatalf("got %+v (%v), want todo 2, Call home", second, err)
	}

	got, err := store.List(t.Context())
	if err != nil || !slices.Equal(got, []todo.Todo{second, first}) {
		t.Errorf("got %+v (%v), want the new todos, newest first", got, err)
	}
}

func TestAddCountsCodePoints(t *testing.T) {
	tests := []struct{ name, title, want string }{
		{"500 letters", strings.Repeat("é", 500), strings.Repeat("é", 500)},
		{"500 emoji", strings.Repeat("👍", 500), strings.Repeat("👍", 500)},
		{
			"500 after trimming",
			" \u00a0" + strings.Repeat("a", 500) + "\n",
			strings.Repeat("a", 500),
		},
		{"white space inside", "Buy\u00a0milk  now", "Buy\u00a0milk  now"},
		{"format characters", "👨\u200d👩\u200d👧", "👨\u200d👩\u200d👧"},
	}

	store, _ := newStore(t)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := store.Add(t.Context(), test.title)
			if err != nil || got.Title != test.want {
				t.Errorf("got %q (%v), want %q", got.Title, err, test.want)
			}
		})
	}
}

func TestAddRefusesABadTitle(t *testing.T) {
	tests := []struct{ name, title, want string }{
		{"empty", "", "invalid title: empty once trimmed"},
		{"white space", " \t\n\u00a0\u3000", "invalid title: empty once trimmed"},
		{"501 letters", strings.Repeat("é", 501), "invalid title: over 500 characters"},
		{"501 emoji", strings.Repeat("👍", 501), "invalid title: over 500 characters"},
		{"tab", "Buy\tmilk", "invalid title: has a control character"},
		{"newline", "Buy\nmilk", "invalid title: has a control character"},
		{"NUL", "Buy\x00milk", "invalid title: has a control character"},
		{"DEL", "Buy\x7fmilk", "invalid title: has a control character"},
		{"NEL", "Buy\u0085milk", "invalid title: has a control character"},
		{"C1 at the end", "Buy milk\u009f", "invalid title: has a control character"},
	}

	store, _ := newStore(t)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := store.Add(t.Context(), test.title)
			if !errors.Is(err, todo.ErrTitle) || err.Error() != test.want ||
				got != (todo.Todo{}) {
				t.Errorf("got %+v (%v), want no todo and %q", got, err, test.want)
			}
		})
	}

	listed, err := store.List(t.Context())
	if err != nil || len(listed) != 0 {
		t.Errorf("got %+v (%v), want nothing stored", listed, err)
	}
}

func TestAddFailsWithoutTheDatabase(t *testing.T) {
	store, pool := newStore(t)
	pool.Close()

	_, err := store.Add(t.Context(), "Buy milk")
	if err == nil || errors.Is(err, todo.ErrTitle) ||
		!strings.HasPrefix(err.Error(), "add the todo: ") {
		t.Errorf("got %v, want an error that starts with add the todo", err)
	}
}
