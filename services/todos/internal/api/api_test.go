package api_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/codedogapp/bakhyard/internal/database"
	"github.com/codedogapp/bakhyard/internal/databasetest"
	"github.com/codedogapp/bakhyard/services/todos/internal/api"
	"github.com/codedogapp/bakhyard/services/todos/internal/migrations"
	"github.com/codedogapp/bakhyard/services/todos/internal/todo"
)

func TestGetTodosListsTheTodos(t *testing.T) {
	router, pool, _ := todos(t)

	rec := get(t, router)
	if rec.Code != http.StatusOK || rec.Body.String() != `{"todos":[]}` ||
		rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("without todos: got %d %q (%s), want 200 and an empty list as JSON",
			rec.Code, rec.Body, rec.Header().Get("Content-Type"))
	}

	for _, title := range []string{"Buy milk", `Answer "Re: <the plan>"`} {
		_, err := pool.Exec(t.Context(), "INSERT INTO todos (title) VALUES ($1)", title)
		if err != nil {
			t.Fatal(err)
		}
	}

	rec = get(t, router)

	want := `{"todos":[{"id":2,"title":"Answer \"Re: <the plan>\"","done":false,` +
		`"priority":"normal"},{"id":1,"title":"Buy milk","done":false,"priority":"normal"}]}`
	if rec.Code != http.StatusOK || rec.Body.String() != want {
		t.Errorf("got %d %s, want 200 %s", rec.Code, rec.Body, want)
	}
}

func TestGetTodosFailsWithoutTheDatabase(t *testing.T) {
	router, pool, logs := todos(t)
	pool.Close()

	rec := get(t, router)
	if rec.Code != http.StatusInternalServerError ||
		rec.Body.String() != "Internal Server Error\n" {
		t.Errorf("got %d %q, want a plain 500", rec.Code, rec.Body)
	}

	if !strings.Contains(logs.String(), `msg="list the todos" err="read the todos: `) {
		t.Errorf("got the logs %q, want the cause", logs)
	}
}

// todos returns a router that serves the todos in a migrated schema of its own, the pool that
// reaches the schema, and the logs that the router writes.
func todos(t *testing.T) (http.Handler, *pgxpool.Pool, *bytes.Buffer) {
	t.Helper()

	schema := databasetest.NewSchema(t)

	pool, err := database.Open(t.Context(), schema.URL, schema.Password, migrations.FS())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(pool.Close)

	var logs bytes.Buffer

	router := chi.NewRouter()
	api.Register(router, todo.NewStore(pool), slog.New(slog.NewTextHandler(&logs, nil)))

	return router, pool, &logs
}

// get returns router's answer to GET /todos.
func get(t *testing.T, router http.Handler) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/todos", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}
