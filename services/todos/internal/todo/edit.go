package todo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// changeSQL sets the title of todo $1 to $2, its done mark to $3, its priority to $4 and its
// ticket to $5, and returns the todo as stored. A NULL keeps the value stored when the UPDATE
// runs, even one that another UPDATE has just changed, so changes from two devices, such as a tick
// and a rename, don't undo each other. An empty ticket unlinks the todo's.
const changeSQL = `UPDATE todos
SET title = COALESCE($2, title), done = COALESCE($3, done), priority = COALESCE($4, priority),
	jira_key = CASE WHEN $5::text IS NULL THEN jira_key ELSE NULLIF($5, '') END
WHERE id = $1 RETURNING id, title, done, priority, COALESCE(jira_key, '')`

// deleteSQL deletes todo $1.
const deleteSQL = `DELETE FROM todos WHERE id = $1`

// ErrNoTodo is the error for a todo that doesn't exist.
var ErrNoTodo = errors.New("no such todo")

// Change is what to change in a todo: its title, its done mark, its priority, its ticket, its
// place in the list or any of them. A nil field keeps its value, and an empty Ticket unlinks the
// todo's (decision 125). Before or After names the todo to go right before or right after in the
// list, never both (decision 118).
type Change struct {
	Title    *string   `json:"title"`
	Done     *bool     `json:"done"`
	Priority *Priority `json:"priority"`
	Ticket   *string   `json:"ticket"`
	Before   *int64    `json:"before"`
	After    *int64    `json:"after"`
}

// Change makes change to the todo todoID and returns the todo. A new title is trimmed and must
// follow Add's rule, or the error wraps ErrTitle; a priority that isn't High, Normal or Low gets
// ErrPriority, a ticket that is neither a Jira key nor empty ErrTicket, a change with both Before
// and After ErrMove, a todo that doesn't exist ErrNoTodo, and a todo to go next to that doesn't
// exist an error that wraps ErrNeighbour. Either way, nothing changes.
func (s *Store) Change(ctx context.Context, todoID int64, change Change) (Todo, error) {
	change, err := checked(change)
	if err != nil {
		return Todo{}, err
	}

	var changed Todo

	err = pgx.BeginFunc(ctx, s.pool, func(txn pgx.Tx) error {
		if err := change.wait(ctx, txn); err != nil {
			return err
		}

		err := txn.QueryRow(ctx, changeSQL,
			todoID, change.Title, change.Done, change.Priority, change.Ticket).
			Scan(&changed.ID, &changed.Title, &changed.Done, &changed.Priority, &changed.Ticket)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNoTodo
		}

		if err != nil {
			return err
		}

		return change.move(ctx, txn, todoID)
	})

	switch {
	case errors.Is(err, ErrNoTodo), errors.Is(err, ErrNeighbour):
		return Todo{}, err
	case err != nil:
		return Todo{}, fmt.Errorf("change the todo: %w", err)
	}

	return changed, nil
}

// checked returns change with its title trimmed, or the error of the first rule that it breaks:
// the title's, the priority's, the ticket's, then the move's.
func checked(change Change) (Change, error) {
	if change.Title != nil {
		title, err := cleanTitle(*change.Title)
		if err != nil {
			return Change{}, err
		}

		change.Title = new(title)
	}

	if change.Priority != nil && !change.Priority.valid() {
		return Change{}, ErrPriority
	}

	if change.Ticket != nil && !validTicket(*change.Ticket) {
		return Change{}, ErrTicket
	}

	if change.Before != nil && change.After != nil {
		return Change{}, ErrMove
	}

	return change, nil
}

// Delete deletes the todo todoID for good; a todo that doesn't exist gets ErrNoTodo.
func (s *Store) Delete(ctx context.Context, todoID int64) error {
	tag, err := s.pool.Exec(ctx, deleteSQL, todoID)
	if err != nil {
		return fmt.Errorf("delete the todo: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrNoTodo
	}

	return nil
}
