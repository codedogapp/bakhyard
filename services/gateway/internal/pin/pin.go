// Package pin pins each allowed email to the Google account that first signs in with it
// (docs/plan.md §3 Auth, decision 87), so that no other account gets in with that email later.
package pin

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// pinSQL pins an account to an email unless one is pinned to it already, and returns the account
// pinned to it. On a conflict, the update keeps the pinned account and lets RETURNING give it.
const pinSQL = `
INSERT INTO pins (email, sub) VALUES ($1, $2)
ON CONFLICT (email) DO UPDATE SET sub = pins.sub
RETURNING sub`

// Store keeps the pins in the gateway's database.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore returns the store that keeps the pins in pool's database.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Pin pins the account sub to email unless an account is pinned to it already, and reports
// whether sub is the account pinned to email. Concurrent first pins of an email agree on one
// account.
func (s *Store) Pin(ctx context.Context, email, sub string) (bool, error) {
	var pinned string

	err := s.pool.QueryRow(ctx, pinSQL, email, sub).Scan(&pinned)
	if err != nil {
		return false, fmt.Errorf("pin the account: %w", err)
	}

	return pinned == sub, nil
}
