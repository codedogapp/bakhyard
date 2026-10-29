package api

import (
	"encoding/json/v2"
	"errors"
	"log/slog"
	"net/http"

	"github.com/codedogapp/bakhyard/internal/jsonbody"
	"github.com/codedogapp/bakhyard/services/todos/internal/todo"
)

// addTodo returns the handler of POST /todos, which adds a todo and answers 201 with it. A body
// that isn't a JSON object with just a title and a ticket, or a title or a ticket that breaks its
// rule, gets a 400 with the reason, the title's first; a body over 64 KiB gets a 413, and a
// database error a 500, whose cause logger gets. An empty, null or missing ticket names none.
func addTodo(store *todo.Store, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var addition todo.Addition
		if !jsonbody.Read(w, r, &addition,
			"the body isn't a JSON object with just a title and a ticket") {
			return
		}

		added, err := store.Add(r.Context(), addition)
		if errors.Is(err, todo.ErrTitle) || errors.Is(err, todo.ErrTicket) {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		if err != nil {
			logger.ErrorContext(r.Context(), "add the todo", "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		// The title passed the rule and came back from a UTF8 database, so the todo always
		// encodes: an error means the client has gone.
		_ = json.MarshalWrite(w, added)
	}
}
