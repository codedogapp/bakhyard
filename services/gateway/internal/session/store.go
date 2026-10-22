// Package session keeps the user's sessions (docs/plan.md §3 Auth): each sign-in starts one, and
// the browser carries its token in a cookie.
package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// lifetime is how long a session lasts.
const lifetime = 7 * 24 * time.Hour

// startSQL stores a session and deletes the expired ones, in one statement.
const startSQL = `
WITH expired AS (DELETE FROM sessions WHERE expires_at <= now())
INSERT INTO sessions (token_hash, expires_at) VALUES ($1, now() + $2::interval)`

// Store keeps the sessions in the gateway's database. It stores the SHA-256 of each session's
// token, never the token.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore returns the store that keeps the sessions in pool's database.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Start starts a session that lasts a week and returns its token. It deletes the expired
// sessions on the way.
func (s *Store) Start(ctx context.Context) (string, error) {
	token := rand.Text()
	hash := sha256.Sum256([]byte(token))

	_, err := s.pool.Exec(ctx, startSQL, hash[:], lifetime)
	if err != nil {
		return "", fmt.Errorf("store a session: %w", err)
	}

	return token, nil
}
