package session_test

import (
	"crypto/sha256"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRenewKeepsALiveSessionForAnotherWeek(t *testing.T) {
	store, pool := newStore(t)
	token := start(t, store)
	hash := sha256.Sum256([]byte(token))
	expireIn(t, pool, hash[:], "1 hour")

	live, err := store.Renew(t.Context(), token)
	if err != nil || !live {
		t.Fatalf("got %t, %v; want a live session", live, err)
	}

	var renewed bool

	err = pool.QueryRow(t.Context(), renewedSQL, hash[:]).Scan(&renewed)
	if err != nil || !renewed {
		t.Errorf("got %t, %v; want the session to expire a week from now", renewed, err)
	}
}

// renewedSQL tells whether the session with the hash $1 expires a week from now, within a minute.
const renewedSQL = `
SELECT expires_at > now() + interval '7 days' - interval '1 minute'
	AND expires_at <= now() + interval '7 days'
FROM sessions WHERE token_hash = $1`

func TestRenewFindsNoSessionForAnUnknownToken(t *testing.T) {
	store, _ := newStore(t)
	start(t, store)

	live, err := store.Renew(t.Context(), "UNKNOWN")
	if err != nil || live {
		t.Errorf("got %t, %v; want no session", live, err)
	}
}

func TestRenewLeavesAnExpiredSessionExpired(t *testing.T) {
	store, pool := newStore(t)
	token := start(t, store)
	hash := sha256.Sum256([]byte(token))
	expireIn(t, pool, hash[:], "1 second ago")

	live, err := store.Renew(t.Context(), token)
	if err != nil || live {
		t.Fatalf("got %t, %v; want no session", live, err)
	}

	var expired bool

	err = pool.QueryRow(t.Context(),
		"SELECT expires_at <= now() FROM sessions WHERE token_hash = $1", hash[:]).Scan(&expired)
	if err != nil || !expired {
		t.Errorf("got %t, %v; want the session still expired", expired, err)
	}
}

func TestRenewFailsWithoutTheDatabase(t *testing.T) {
	store, pool := newStore(t)
	pool.Close()

	_, err := store.Renew(t.Context(), "TOKEN")
	if err == nil || !strings.HasPrefix(err.Error(), "renew a session: ") {
		t.Errorf("got %v, want an error that starts with renew a session", err)
	}
}

// expireIn moves the expiry of the session with hash to now plus interval, a Postgres interval.
func expireIn(t *testing.T, pool *pgxpool.Pool, hash []byte, interval string) {
	t.Helper()

	_, err := pool.Exec(t.Context(),
		"UPDATE sessions SET expires_at = now() + $2::interval WHERE token_hash = $1",
		hash, interval)
	if err != nil {
		t.Fatal(err)
	}
}
