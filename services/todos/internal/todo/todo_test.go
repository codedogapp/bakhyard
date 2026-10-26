package todo_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/codedogapp/bakhyard/internal/database"
	"github.com/codedogapp/bakhyard/internal/databasetest"
	"github.com/codedogapp/bakhyard/services/todos/internal/migrations"
	"github.com/codedogapp/bakhyard/services/todos/internal/todo"
)

func TestListListsTheNewestFirst(t *testing.T) {
	store, pool := newStore(t)
	// The rows are stored oldest first, and the titles sort in neither order.
	for _, title := range []string{"Buy milk", "Answer mail", "Call home"} {
		add(t, pool, title)
	}

	got, err := store.List(t.Context())

	want := []todo.Todo{
		{ID: 3, Title: "Call home", Priority: todo.Normal},
		{ID: 2, Title: "Answer mail", Priority: todo.Normal},
		{ID: 1, Title: "Buy milk", Priority: todo.Normal},
	}
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("got %+v (%v), want %+v", got, err, want)
	}
}

func TestListListsNothingWithoutTodos(t *testing.T) {
	store, _ := newStore(t)

	got, err := store.List(t.Context())
	if err != nil || got == nil || len(got) != 0 {
		t.Errorf("got %#v (%v), want an empty list", got, err)
	}
}

func TestListFailsWithoutTheDatabase(t *testing.T) {
	store, pool := newStore(t)
	pool.Close()

	_, err := store.List(t.Context())
	if err == nil || !strings.HasPrefix(err.Error(), "read the todos: ") {
		t.Errorf("got %v, want an error that starts with read the todos", err)
	}
}

// newStore returns a store in a migrated schema of its own, and the pool that reaches it.
func newStore(t *testing.T) (*todo.Store, *pgxpool.Pool) {
	t.Helper()

	schema := databasetest.NewSchema(t)

	pool, err := database.Open(t.Context(), schema.URL, schema.Password, migrations.FS())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(pool.Close)

	return todo.NewStore(pool), pool
}

// add stores a todo with title in pool's database, as nothing in todos can yet.
func add(t *testing.T, pool *pgxpool.Pool, title string) {
	t.Helper()

	_, err := pool.Exec(t.Context(), "INSERT INTO todos (title) VALUES ($1)", title)
	if err != nil {
		t.Fatal(err)
	}
}
