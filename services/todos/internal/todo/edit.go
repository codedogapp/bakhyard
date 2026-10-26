package todo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// changeSQL sets the title of todo $1 to $2, its done mark to $3 and its priority to $4, and
// returns the todo as stored. A NULL keeps the value stored when the UPDATE runs, even one that
// another UPDATE has just changed, so changes from two devices, such as a tick and a rename, don't
// undo each other.
const changeSQL = `UPDATE todos
SET title = COALESCE($2, title), done = COALESCE($3, done), priority = COALESCE($4, priority)
WHERE id = $1 RETURNING id, title, done, priority`

// deleteSQL deletes todo $1.
const deleteSQL = `DELETE FROM todos WHERE id = $1`

// ErrNoTodo is the error for a todo that doesn't exist.
var ErrNoTodo = errors.New("no such todo")

// Change is what to change in a todo: its title, its done mark, its priority or any of them. A nil
// field keeps its value.
type Change struct {
	Title    *string   `json:"title"`
	Done     *bool     `json:"done"`
	Priority *Priority `json:"priority"`
}

// Change makes change to the todo todoID and returns the todo. A new title is trimmed and must
// follow Add's rule, or the error wraps ErrTitle; a priority that isn't High, Normal or Low gets
// ErrPriority, and a todo that doesn't exist ErrNoTodo. Either way, nothing changes.
func (s *Store) Change(ctx context.Context, todoID int64, change Change) (Todo, error) {
	if change.Title != nil {
		title, err := cleanTitle(*change.Title)
		if err != nil {
			return Todo{}, err
		}

		change.Title = new(title)
	}

	if change.Priority != nil && !change.Priority.valid() {
		return Todo{}, ErrPriority
	}

	var changed Todo

	err := s.pool.QueryRow(ctx, changeSQL, todoID, change.Title, change.Done, change.Priority).
		Scan(&changed.ID, &changed.Title, &changed.Done, &changed.Priority)
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
