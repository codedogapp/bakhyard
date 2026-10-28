package subscription

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// addSQL stores the subscription $1, $2 and $3, or replaces the keys of the one with its endpoint.
	addSQL = `INSERT INTO push_subscriptions (endpoint, p256dh, auth) VALUES ($1, $2, $3)
ON CONFLICT (endpoint) DO UPDATE SET p256dh = EXCLUDED.p256dh, auth = EXCLUDED.auth`
	listSQL   = `SELECT endpoint, p256dh, auth FROM push_subscriptions ORDER BY created_at, endpoint`
	removeSQL = `DELETE FROM push_subscriptions WHERE endpoint = $1`
)

// Store keeps the subscriptions in notify's database.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore returns the store that keeps the subscriptions in pool's database.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Add stores sub, or puts it in the place of the subscription with its endpoint. A subscription
// that fails Check gets its error, wrapping ErrInvalid, and nothing is stored.
func (s *Store) Add(ctx context.Context, sub Subscription) error {
	if err := sub.Check(); err != nil {
		return err
	}

	if _, err := s.pool.Exec(
		ctx,
		addSQL,
		sub.Endpoint,
		sub.Keys.P256dh,
		sub.Keys.Auth,
	); err != nil {
		return fmt.Errorf("store the subscription: %w", err)
	}

	return nil
}

// List returns every subscription, the oldest first.
func (s *Store) List(ctx context.Context) ([]Subscription, error) {
	// pgx allows ignoring Query's error: CollectRows returns it.
	rows, _ := s.pool.Query(ctx, listSQL)

	subs, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Subscription, error) {
		var sub Subscription

		err := row.Scan(&sub.Endpoint, &sub.Keys.P256dh, &sub.Keys.Auth)

		return sub, err
	})
	if err != nil {
		return nil, fmt.Errorf("read the subscriptions: %w", err)
	}

	return subs, nil
}

// Remove forgets the subscription with endpoint; one that isn't there is no error.
func (s *Store) Remove(ctx context.Context, endpoint string) error {
	if _, err := s.pool.Exec(ctx, removeSQL, endpoint); err != nil {
		return fmt.Errorf("forget the subscription: %w", err)
	}

	return nil
}
