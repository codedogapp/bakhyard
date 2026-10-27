package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/codedogapp/bakhyard/services/todos/internal/todo"
)

// dismissals is the body of GET /todos/dismissals: the keys of the tickets dismissed from the
// inbox.
type dismissals struct {
	Dismissals []string `json:"dismissals"`
}

// listDismissals returns the handler of GET /todos/dismissals, which answers the keys of the
// tickets dismissed from the inbox in byte order. When they can't be read, the client gets a 500
// and logger gets the cause.
func listDismissals(store *todo.Store, logger *slog.Logger) http.HandlerFunc {
	return serveRead(func(ctx context.Context) (dismissals, error) {
		keys, err := store.Dismissed(ctx)

		return dismissals{Dismissals: keys}, err
	}, "list the dismissals", logger)
}

// dismissTicket returns the handler of PUT /todos/dismissals/{key}, which dismisses the ticket
// from the inbox for good and answers 201, or 204 when it was dismissed already (RFC 9110
// §9.3.4); a body is ignored. A key that isn't a Jira key gets a 400 with the reason, and a
// database error a 500, whose cause logger gets.
func dismissTicket(store *todo.Store, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		added, err := store.Dismiss(r.Context(), r.PathValue("key"))

		switch {
		case errors.Is(err, todo.ErrTicket):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case err != nil:
			logger.ErrorContext(r.Context(), "dismiss the ticket", "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)
		case added:
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusNoContent)
		}
	}
}
