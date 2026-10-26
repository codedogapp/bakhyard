package api

import (
	"encoding/json/v2"
	"errors"
	"log/slog"
	"net/http"

	"github.com/codedogapp/bakhyard/services/todos/internal/todo"
)

// maxAddition is the largest body POST /todos reads: 64 KiB, many times a 500-character title's.
const maxAddition = 64 << 10

// addition is the body of POST /todos: the title of the todo to add.
type addition struct {
	Title string `json:"title"`
}

// addTodo returns the handler of POST /todos, which adds a todo and answers 201 with it. A body
// that isn't a JSON object with just a title, or a title that breaks the rule, gets a 400 with
// the reason, a body over maxAddition a 413, and a database error a 500, whose cause logger gets.
func addTodo(store *todo.Store, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body addition

		err := json.UnmarshalRead(http.MaxBytesReader(w, r.Body, maxAddition), &body,
			json.RejectUnknownMembers(true))
		if _, tooLarge := errors.AsType[*http.MaxBytesError](err); tooLarge {
			http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge),
				http.StatusRequestEntityTooLarge)

			return
		}

		if err != nil {
			http.Error(w, "the body isn't a JSON object with just a title", http.StatusBadRequest)

			return
		}

		added, err := store.Add(r.Context(), body.Title)
		if errors.Is(err, todo.ErrTitle) {
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
