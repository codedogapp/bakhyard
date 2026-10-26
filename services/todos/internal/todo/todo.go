// Package todo keeps the user's todos (docs/plan.md §8.2 todos-jira, decision 102).
package todo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// listSQL selects every todo, done or not, in the list's order: a new todo on top, and a moved one
// where the move put it (decisions 104 and 118). No two todos share a position, but the ID would
// still put the newer first.
const listSQL = `SELECT id, title, done, priority FROM todos ORDER BY position DESC, id DESC`

// addSQL stores a todo with the title $1 and returns it; the database makes it Normal and puts it
// on top of the list.
const addSQL = `INSERT INTO todos (title) VALUES ($1) RETURNING id, title, done, priority`

// Todo is one of the user's todos.
type Todo struct {
	ID       int64    `json:"id"`
	Title    string   `json:"title"`
	Done     bool     `json:"done"`
	Priority Priority `json:"priority"`
}

// Store keeps the todos in todos' database.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore returns the store that keeps the todos in pool's database.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// List returns every todo in the list's order, and an empty list when there are none.
func (s *Store) List(ctx context.Context) ([]Todo, error) {
	// pgx allows ignoring Query's error: CollectRows returns it.
	rows, _ := s.pool.Query(ctx, listSQL)

	todos, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Todo])
	if err != nil {
		return nil, fmt.Errorf("read the todos: %w", err)
	}

	return todos, nil
}

// Add trims title, stores it as a new todo, which is Normal and on top of the list, and returns
// the todo. A title that is empty once trimmed, has more than 500 characters or has a control
// character gets an error that wraps ErrTitle, and nothing is stored.
func (s *Store) Add(ctx context.Context, title string) (Todo, error) {
	title, err := cleanTitle(title)
	if err != nil {
		return Todo{}, err
	}

	var added Todo

	err = s.pool.QueryRow(ctx, addSQL, title).
		Scan(&added.ID, &added.Title, &added.Done, &added.Priority)
	if err != nil {
		return Todo{}, fmt.Errorf("add the todo: %w", err)
	}

	return added, nil
}
