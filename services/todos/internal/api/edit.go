package api

import (
	"encoding/json/v2"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/codedogapp/bakhyard/services/todos/internal/todo"
)

// changeTodo returns the handler of PATCH /todos/{id}, which changes the todo's title, its done
// mark, its priority, its ticket, its place in the list or any of them and answers 200 with the
// todo; a member that is null or missing keeps its value, an empty ticket unlinks the todo's
// (decision 125), and before or after names the todo to go right before or right after (decision
// 118). A body that isn't a JSON object with just those members, a title, a priority or a ticket
// that breaks its rule, both before and after, or a todo to go next to that doesn't exist gets a
// 400 with the reason, a body over maxBody a 413, a todo that doesn't exist a 404, and a database
// error a 500, whose cause logger gets.
func changeTodo(store *todo.Store, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		todoID, ok := idInPath(w, r)
		if !ok {
			return
		}

		var change todo.Change
		if !readBody(w, r, &change, "the body isn't a JSON object with just a title, a done mark, "+
			"a priority, a ticket and the ID of a todo to go before or after") {
			return
		}

		changed, err := store.Change(r.Context(), todoID, change)

		switch {
		case errors.Is(err, todo.ErrTitle), errors.Is(err, todo.ErrPriority),
			errors.Is(err, todo.ErrTicket), errors.Is(err, todo.ErrMove),
			errors.Is(err, todo.ErrNeighbour):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, todo.ErrNoTodo):
			http.Error(w, err.Error(), http.StatusNotFound)
		case err != nil:
			logger.ErrorContext(r.Context(), "change the todo", "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)
		default:
			w.Header().Set("Content-Type", "application/json")
			// The title passed the rule and came back from a UTF8 database, so the todo always
			// encodes: an error means the client has gone.
			_ = json.MarshalWrite(w, changed)
		}
	}
}

// deleteTodo returns the handler of DELETE /todos/{id}, which deletes the todo for good and
// answers 204. A todo that doesn't exist gets a 404, and a database error a 500, whose cause
// logger gets.
func deleteTodo(store *todo.Store, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		todoID, ok := idInPath(w, r)
		if !ok {
			return
		}

		err := store.Delete(r.Context(), todoID)

		switch {
		case errors.Is(err, todo.ErrNoTodo):
			http.Error(w, err.Error(), http.StatusNotFound)
		case err != nil:
			logger.ErrorContext(r.Context(), "delete the todo", "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusNoContent)
		}
	}
}

// idInPath returns the todo's ID in r's path. An ID that isn't a number gets a 404, as a todo
// that doesn't exist does, and idInPath returns false.
func idInPath(w http.ResponseWriter, r *http.Request) (int64, bool) {
	todoID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, todo.ErrNoTodo.Error(), http.StatusNotFound)

		return 0, false
	}

	return todoID, true
}
