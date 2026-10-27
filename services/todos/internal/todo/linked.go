package todo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// linkedSQL selects the keys of the tickets that the todos not yet done name, each once, byte by
// byte in order, whatever the database's collation. SELECT DISTINCT orders only by what it
// selects, so the collation goes on the column it selects.
const linkedSQL = `SELECT DISTINCT jira_key COLLATE "C" AS jira_key FROM todos
WHERE jira_key IS NOT NULL AND NOT done ORDER BY jira_key`

// Linked returns the keys of the Jira tickets that the todos not yet done name, each once, in byte
// order, and an empty list when there are none (decision 133).
func (s *Store) Linked(ctx context.Context) ([]string, error) {
	// pgx allows ignoring Query's error: CollectRows returns it.
	rows, _ := s.pool.Query(ctx, linkedSQL)

	keys, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("read the linked tickets: %w", err)
	}

	return keys, nil
}
