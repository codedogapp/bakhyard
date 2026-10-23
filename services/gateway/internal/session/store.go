// Package session keeps the user's sessions (docs/plan.md §3 Auth): each sign-in starts one, the
// browser carries its token in a cookie, and the gateway's second lock lets in only the requests
// that carry a live one.
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

// renewSQL moves a live session's expiry to a lifetime from now.
const renewSQL = `
UPDATE sessions SET expires_at = now() + $2::interval
WHERE token_hash = $1 AND expires_at > now()`

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

// Renew makes the session whose token is token last another week from now, and reports whether it
// was live: a token that is unknown or expired has no session to renew.
func (s *Store) Renew(ctx context.Context, token string) (bool, error) {
	hash := sha256.Sum256([]byte(token))

	tag, err := s.pool.Exec(ctx, renewSQL, hash[:], lifetime)
	if err != nil {
		return false, fmt.Errorf("renew a session: %w", err)
	}

	return tag.RowsAffected() == 1, nil
}
