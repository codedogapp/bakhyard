// Package todo keeps the user's todos (docs/plan.md §8.2 todos-jira, decision 102).
package todo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// listSQL selects every todo, newest first (decision 104).
const listSQL = `SELECT id, title FROM todos ORDER BY id DESC`

// Todo is one of the user's todos.
type Todo struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

// Store keeps the todos in todos' database.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore returns the store that keeps the todos in pool's database.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// List returns every todo, newest first, and an empty list when there are none.
func (s *Store) List(ctx context.Context) ([]Todo, error) {
	// pgx allows ignoring Query's error: CollectRows returns it.
	rows, _ := s.pool.Query(ctx, listSQL)

	todos, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Todo])
	if err != nil {
		return nil, fmt.Errorf("read the todos: %w", err)
	}

	return todos, nil
}
