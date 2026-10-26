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

	first, err := store.Add(t.Context(), todo.Addition{Title: "  Buy milk  "})
	if err != nil || first != (todo.Todo{ID: 1, Title: "Buy milk", Priority: todo.Normal}) {
		t.Fatalf("got %+v (%v), want todo 1, Buy milk, Normal", first, err)
	}

	second, err := store.Add(t.Context(), todo.Addition{Title: "\tCall home\n"})
	if err != nil || second != (todo.Todo{ID: 2, Title: "Call home", Priority: todo.Normal}) {
		t.Fatalf("got %+v (%v), want todo 2, Call home, Normal", second, err)
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
			got, err := store.Add(t.Context(), todo.Addition{Title: test.title})
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
			got, err := store.Add(t.Context(), todo.Addition{Title: test.title})
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

	_, err := store.Add(t.Context(), todo.Addition{Title: "Buy milk"})
	if err == nil || errors.Is(err, todo.ErrTitle) ||
		!strings.HasPrefix(err.Error(), "add the todo: ") {
		t.Errorf("got %v, want an error that starts with add the todo", err)
	}
}

func TestAddLinksATicket(t *testing.T) {
	store, pool := newStore(t)

	want := todo.Todo{ID: 1, Title: "Fix the date picker", Priority: todo.Normal, Ticket: "WEB-9"}

	linked, err := store.Add(t.Context(),
		todo.Addition{Title: " Fix the date picker", Ticket: "WEB-9"})
	if err != nil || linked != want {
		t.Fatalf("got %+v (%v), want %+v", linked, err, want)
	}

	plain, err := store.Add(t.Context(), todo.Addition{Title: "Buy milk", Ticket: ""})
	if err != nil || plain != (todo.Todo{ID: 2, Title: "Buy milk", Priority: todo.Normal}) {
		t.Fatalf("got %+v (%v), want todo 2 without a ticket", plain, err)
	}

	if got := list(t, store); !slices.Equal(got, []todo.Todo{plain, linked}) {
		t.Errorf("got %+v, want both todos, the ticket's listed with it", got)
	}

	var none bool

	err = pool.QueryRow(t.Context(), "SELECT jira_key IS NULL FROM todos WHERE id = 2").Scan(&none)
	if err != nil || !none {
		t.Errorf("got jira_key IS NULL %v (%v), want a todo without a ticket to store NULL, not ''",
			none, err)
	}
}

func TestAddRefusesAnotherTicket(t *testing.T) {
	store, _ := newStore(t)

	for _, ticket := range []string{"abc-1", "A-1", "AB-0", " AB-1", "AB-1\n", "AB 1", "AB-1١"} {
		got, err := store.Add(t.Context(), todo.Addition{Title: "Buy milk", Ticket: ticket})
		if !errors.Is(err, todo.ErrTicket) || got != (todo.Todo{}) {
			t.Errorf("%q: got %+v (%v), want no todo and %v", ticket, got, err, todo.ErrTicket)
		}
	}

	if got := list(t, store); len(got) != 0 {
		t.Errorf("got %+v, want nothing stored", got)
	}
}

func TestAddChecksTheTitleBeforeTheTicket(t *testing.T) {
	store, _ := newStore(t)

	_, err := store.Add(t.Context(), todo.Addition{Title: " ", Ticket: "abc-1"})
	if !errors.Is(err, todo.ErrTitle) {
		t.Errorf("got %v, want %v", err, todo.ErrTitle)
	}
}
