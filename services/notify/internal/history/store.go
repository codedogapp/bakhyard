// Package history keeps the alerts that Alertmanager posts, a row for each time one fired, for the
// page that follows them up (decision 606).
package history

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/codedogapp/bakhyard/services/notify/internal/alert"
)

const (
	// kept is how long an alert stays after it resolved.
	kept = 30 * 24 * time.Hour
	// most is how many alerts List returns at most.
	most = 100
)

const (
	// keepSQL stores the firing of the alert $1 that started at $2, or updates its name, summary
	// and description. $6 is when it resolved, NULL while it fires; once resolved, it stays so.
	keepSQL = `INSERT INTO alerts (fingerprint, started_at, name, summary, description, resolved_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (fingerprint, started_at) DO UPDATE SET name = EXCLUDED.name,
    summary = EXCLUDED.summary, description = EXCLUDED.description,
    resolved_at = COALESCE(alerts.resolved_at, EXCLUDED.resolved_at)`
	// pruneSQL forgets the alerts that resolved more than $1 seconds ago.
	pruneSQL = `DELETE FROM alerts WHERE resolved_at < now() - make_interval(secs => $1)`
	// listSQL reads the alerts that fire or resolved in the last $1 seconds, at most $2 of them.
	listSQL = `SELECT name, summary, description, started_at, resolved_at FROM alerts
WHERE resolved_at IS NULL OR resolved_at >= now() - make_interval(secs => $1)
ORDER BY resolved_at IS NULL DESC, started_at DESC, fingerprint
LIMIT $2`
)

// Entry is a time that an alert fired: the alert's name, summary and description, when it
// started, and when it resolved, zero while it fires.
type Entry struct {
	Name        string
	Summary     string
	Description string
	StartedAt   time.Time
	ResolvedAt  time.Time
}

// Store keeps the alerts in notify's database.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore returns the store that keeps the alerts in pool's database.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Keep stores alerts and then forgets those that resolved more than 30 days ago, all at once. An
// alert that fires again later gets a row of its own. Keep skips the alerts that it can't tell
// apart, without a fingerprint or a start, those whose status is neither firing nor resolved, and
// the resolved ones without an end.
func (s *Store) Keep(ctx context.Context, alerts []alert.Alert) error {
	err := pgx.BeginFunc(ctx, s.pool, func(txn pgx.Tx) error {
		for _, found := range alerts {
			resolved, ok := resolvedAt(found)
			if !ok {
				continue
			}

			if _, err := txn.Exec(ctx, keepSQL, found.Fingerprint, found.StartsAt,
				found.Labels["alertname"], found.Annotations["summary"],
				found.Annotations["description"], resolved); err != nil {
				return err
			}
		}

		_, err := txn.Exec(ctx, pruneSQL, kept.Seconds())

		return err
	})
	if err != nil {
		return fmt.Errorf("keep the alerts: %w", err)
	}

	return nil
}

// List returns the alerts that fire, the latest to start first, and then those that resolved in
// the last 30 days, likewise; 100 at most.
func (s *Store) List(ctx context.Context) ([]Entry, error) {
	// pgx allows ignoring Query's error: CollectRows returns it.
	rows, _ := s.pool.Query(ctx, listSQL, kept.Seconds(), most)

	entries, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Entry, error) {
		var (
			entry    Entry
			resolved *time.Time
		)

		err := row.Scan(&entry.Name, &entry.Summary, &entry.Description, &entry.StartedAt,
			&resolved)
		if resolved != nil {
			entry.ResolvedAt = *resolved
		}

		return entry, err
	})
	if err != nil {
		return nil, fmt.Errorf("read the alerts: %w", err)
	}

	return entries, nil
}

// resolvedAt returns when found resolved, nil while it fires, and false when it can't be kept.
func resolvedAt(found alert.Alert) (*time.Time, bool) {
	if found.Fingerprint == "" || found.StartsAt.IsZero() {
		return nil, false
	}

	switch found.Status {
	case alert.Firing:
		return nil, true
	case alert.Resolved:
		return new(found.EndsAt), !found.EndsAt.IsZero()
	}

	return nil, false
}
