package todo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/codedogapp/bakhyard/internal/jira"
)

// dismissSQL stores the dismissal of ticket $1, unless it is stored already.
const dismissSQL = `INSERT INTO dismissals (jira_key) VALUES ($1) ON CONFLICT DO NOTHING`

// dismissedSQL selects the dismissed tickets' keys, byte by byte in order, whatever the database's
// collation.
const dismissedSQL = `SELECT jira_key FROM dismissals ORDER BY jira_key COLLATE "C"`

// Dismiss dismisses the Jira ticket key from the inbox for good (decision 130), and reports whether
// it wasn't dismissed already. A key that isn't a Jira key gets ErrTicket, and nothing is stored.
func (s *Store) Dismiss(ctx context.Context, key string) (bool, error) {
	if !jira.IsKey(key) {
		return false, ErrTicket
	}

	tag, err := s.pool.Exec(ctx, dismissSQL, key)
	if err != nil {
		return false, fmt.Errorf("dismiss the ticket: %w", err)
	}

	return tag.RowsAffected() == 1, nil
}

// Dismissed returns the keys of the dismissed tickets in byte order, and an empty list when there
// are none.
func (s *Store) Dismissed(ctx context.Context) ([]string, error) {
	// pgx allows ignoring Query's error: CollectRows returns it.
	rows, _ := s.pool.Query(ctx, dismissedSQL)

	keys, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("read the dismissals: %w", err)
	}

	return keys, nil
}
