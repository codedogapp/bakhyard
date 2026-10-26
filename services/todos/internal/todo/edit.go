package todo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// changeSQL sets the title of todo $1 to $2 and its done mark to $3, and returns the todo as
// stored. A NULL keeps the value stored when the UPDATE runs, even one that another UPDATE has
// just changed, so a tick and a rename from two devices don't undo each other.
const changeSQL = `UPDATE todos SET title = COALESCE($2, title), done = COALESCE($3, done)
WHERE id = $1 RETURNING id, title, done`

// deleteSQL deletes todo $1.
const deleteSQL = `DELETE FROM todos WHERE id = $1`

// ErrNoTodo is the error for a todo that doesn't exist.
var ErrNoTodo = errors.New("no such todo")

// Change is what to change in a todo: its title, its done mark or both. A nil field keeps its
// value.
type Change struct {
	Title *string `json:"title"`
	Done  *bool   `json:"done"`
}

// Change makes change to the todo todoID and returns the todo. A new title is trimmed and must
// follow Add's rule, or the error wraps ErrTitle; a todo that doesn't exist gets ErrNoTodo. Either
// way, nothing changes.
func (s *Store) Change(ctx context.Context, todoID int64, change Change) (Todo, error) {
	if change.Title != nil {
		title, err := cleanTitle(*change.Title)
		if err != nil {
			return Todo{}, err
		}

		change.Title = new(title)
	}

	var changed Todo

	err := s.pool.QueryRow(ctx, changeSQL, todoID, change.Title, change.Done).
		Scan(&changed.ID, &changed.Title, &changed.Done)
	if errors.Is(err, pgx.ErrNoRows) {
		return Todo{}, ErrNoTodo
	}

	if err != nil {
		return Todo{}, fmt.Errorf("change the todo: %w", err)
	}

	return changed, nil
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
