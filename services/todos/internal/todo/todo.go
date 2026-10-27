// Package todo keeps the user's todos, and the Jira tickets that they dismiss from the Todos page's
// inbox (docs/plan.md §8.2 todos-jira, decisions 102 and 130).
package todo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// listSQL selects every todo, done or not, in the list's order: a new todo on top, and a moved one
// where the move put it (decisions 104 and 118). No two todos share a position, but the ID would
// still put the newer first. A todo without a ticket gets an empty one.
const listSQL = `SELECT id, title, done, priority, COALESCE(jira_key, '') FROM todos
ORDER BY position DESC, id DESC`

// addSQL stores a todo with the title $1 and the ticket $2, none when empty, and returns it; the
// database makes it Normal and puts it on top of the list.
const addSQL = `INSERT INTO todos (title, jira_key) VALUES ($1, NULLIF($2, ''))
RETURNING id, title, done, priority, COALESCE(jira_key, '')`

// Todo is one of the user's todos. Ticket is the key of the Jira ticket that it names, or empty
// for none, which the JSON leaves out (decision 125).
type Todo struct {
	ID       int64    `json:"id"`
	Title    string   `json:"title"`
	Done     bool     `json:"done"`
	Priority Priority `json:"priority"`
	Ticket   string   `json:"ticket,omitempty"`
}

// Addition is a todo to add: its title, and the key of the Jira ticket that it names, or empty
// for none (decision 128).
type Addition struct {
	Title  string `json:"title"`
	Ticket string `json:"ticket"`
}

// Store keeps the todos, and the dismissed tickets, in todos' database.
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

// Add trims addition's title, stores it as a new todo, which is Normal and on top of the list and
// names addition's ticket, and returns the todo. A title that is empty once trimmed, has more than
// 500 characters or has a control character gets an error that wraps ErrTitle; a good title with a
// ticket that is neither empty nor a Jira key gets ErrTicket. Either way, nothing is stored.
func (s *Store) Add(ctx context.Context, addition Addition) (Todo, error) {
	title, err := cleanTitle(addition.Title)
	if err != nil {
		return Todo{}, err
	}

	if !validTicket(addition.Ticket) {
		return Todo{}, ErrTicket
	}

	var added Todo

	err = s.pool.QueryRow(ctx, addSQL, title, addition.Ticket).
		Scan(&added.ID, &added.Title, &added.Done, &added.Priority, &added.Ticket)
	if err != nil {
		return Todo{}, fmt.Errorf("add the todo: %w", err)
	}

	return added, nil
}
