// Package api serves the user's todos over HTTP.
package api

import (
	"encoding/json/v2"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/services/todos/internal/todo"
)

// list is the body of GET /todos: every todo, newest first.
type list struct {
	Todos []todo.Todo `json:"todos"`
}

// Register serves the todos in store on router: GET /todos lists them. Other methods get 405,
// and logger gets the errors that reading the todos runs into.
func Register(router chi.Router, store *todo.Store, logger *slog.Logger) {
	router.Get("/todos", listTodos(store, logger))
}

// listTodos returns the handler of GET /todos. When the todos can't be read, the client gets a
// 500 and logger gets the cause.
func listTodos(store *todo.Store, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		todos, err := store.List(r.Context())
		if err != nil {
			logger.ErrorContext(r.Context(), "list the todos", "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		// The titles come from a UTF8 database, so the list always encodes: an error means the
		// client has gone.
		_ = json.MarshalWrite(w, list{Todos: todos})
	}
}
