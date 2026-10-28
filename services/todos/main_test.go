package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/internal/databasetest"
	"github.com/codedogapp/bakhyard/internal/secrets"
)

func TestTodosServesTheTodos(t *testing.T) {
	database, items := todosEnv(t)

	router, err := routes(t.Context(), database, secretsWith(items))
	if err != nil {
		t.Fatal(err)
	}

	// A recorder keeps what the GET handler writes for HEAD; a server would drop it.
	for _, test := range []struct {
		method, target, request string
		code                    int
		body                    string
	}{
		{http.MethodGet, "/todos", "", http.StatusOK, `{"todos":[]}`},
		{http.MethodHead, "/todos", "", http.StatusOK, `{"todos":[]}`},
		{
			http.MethodPost, "/todos", `{"title":"Buy milk"}`,
			http.StatusCreated, `{"id":1,"title":"Buy milk","done":false,"priority":"normal"}`,
		},
		{
			http.MethodPatch, "/todos/1", `{"done":true}`,
			http.StatusOK, `{"id":1,"title":"Buy milk","done":true,"priority":"normal"}`,
		},
		{http.MethodDelete, "/todos/1", "", http.StatusNoContent, ""},
		{http.MethodPut, "/todos", "", http.StatusMethodNotAllowed, ""},
		{http.MethodGet, "/todos/tickets", "", http.StatusNotFound, "Jira is not set up\n"},
		{http.MethodPut, "/todos/dismissals/WEB-9", "", http.StatusCreated, ""},
		{http.MethodGet, "/todos/dismissals", "", http.StatusOK, `{"dismissals":["WEB-9"]}`},
		{http.MethodGet, "/healthz", "", http.StatusOK, `{"status":"ok"}`},
	} {
		req := httptest.NewRequestWithContext(t.Context(), test.method, test.target,
			strings.NewReader(test.request))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != test.code || rec.Body.String() != test.body {
			t.Errorf("%s %s: got %d %q, want %d %q", test.method, test.target, rec.Code,
				rec.Body, test.code, test.body)
		}
	}
}

func TestTodosStartsAgainOnItsMigratedDatabase(t *testing.T) {
	database, items := todosEnv(t)

	// The second start finds the migrations done; running them again would fail.
	for range 2 {
		_, err := routes(t.Context(), database, secretsWith(items))
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestTodosNeedsItsDatabase(t *testing.T) {
	// Nothing listens on port 1.
	const unreachable = "postgres://bakhyard_todos@127.0.0.1:1/bakhyard_todos"

	items := map[string]string{"postgres-todos": "secret"}
	for _, test := range []struct {
		database string
		items    map[string]string
		want     string
	}{
		{"", items, "BAKHYARD_DATABASE is not set"},
		{unreachable, nil, "read the database password: no secret postgres-todos"},
		{unreachable, items, "reach the database: "},
	} {
		_, err := routes(t.Context(), databaseAt(test.database), secretsWith(test.items))
		if err == nil || !strings.HasPrefix(err.Error(), test.want) {
			t.Errorf("with the database %q and the items %v: got %v, want an error that starts "+
				"with %s", test.database, test.items, err, test.want)
		}
	}
}

// todosEnv returns todos' environment and secrets, with the database in a schema of its own.
func todosEnv(t *testing.T) (func(string) string, map[string]string) {
	t.Helper()

	schema := databasetest.NewSchema(t)

	return databaseAt(schema.URL), map[string]string{"postgres-todos": schema.Password}
}

// databaseAt returns an environment whose BAKHYARD_DATABASE is address.
func databaseAt(address string) func(string) string {
	return func(name string) string { return map[string]string{databaseVar: address}[name] }
}

// secretsWith returns a fake secrets reader that has only items, by name.
func secretsWith(items map[string]string) secrets.Reader {
	return func(_ context.Context, name string) (string, error) {
		secret, ok := items[name]
		if !ok {
			return "", fmt.Errorf("no secret %s", name)
		}

		return secret, nil
	}
}
