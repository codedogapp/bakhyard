package session_test

import (
	"crypto/sha256"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/codedogapp/bakhyard/internal/database"
	"github.com/codedogapp/bakhyard/internal/databasetest"
	"github.com/codedogapp/bakhyard/services/gateway/internal/migrations"
	"github.com/codedogapp/bakhyard/services/gateway/internal/session"
)

func TestStartGivesEachSessionANewRandomToken(t *testing.T) {
	store, _ := newStore(t)
	// rand.Text's base32 alphabet, and at least its 128 bits.
	format := regexp.MustCompile(`^[A-Z2-7]{26,}$`)

	first, second := start(t, store), start(t, store)
	for _, token := range []string{first, second} {
		if !format.MatchString(token) {
			t.Errorf("got the token %q, want %s", token, format)
		}
	}

	if first == second {
		t.Errorf("got the token %q twice, want a new one", first)
	}
}

func TestStartStoresTheTokensHashForAWeek(t *testing.T) {
	store, pool := newStore(t)
	token := start(t, store)

	got := stored(t, pool)
	if len(got) != 1 {
		t.Fatalf("got %d sessions, want 1", len(got))
	}

	hash := sha256.Sum256([]byte(token))
	if string(got[0].Hash) != string(hash[:]) || got[0].Lifetime != 7*24*time.Hour {
		t.Errorf("got the hash %x for %s, want %x for a week", got[0].Hash, got[0].Lifetime, hash)
	}
}

func TestStartDeletesTheExpiredSessions(t *testing.T) {
	store, pool := newStore(t)
	expired, live := sha256.Sum256([]byte("expired")), sha256.Sum256([]byte("live"))

	_, err := pool.Exec(t.Context(), `INSERT INTO sessions (token_hash, expires_at)
		VALUES ($1, now() - interval '1 second'), ($2, now() + interval '1 hour')`,
		expired[:], live[:])
	if err != nil {
		t.Fatal(err)
	}

	start(t, store)

	sessions := stored(t, pool)
	for _, row := range sessions {
		if string(row.Hash) == string(expired[:]) {
			t.Error("the expired session is still there")
		}
	}

	if len(sessions) != 2 {
		t.Errorf("got %d sessions, want the live one and the new one", len(sessions))
	}
}

func TestStartFailsWithoutTheDatabase(t *testing.T) {
	store, pool := newStore(t)
	pool.Close()

	_, err := store.Start(t.Context())
	if err == nil || !strings.HasPrefix(err.Error(), "store a session: ") {
		t.Errorf("got %v, want an error that starts with store a session", err)
	}
}

// newStore returns a store in a migrated schema of its own, and the pool that reaches it.
func newStore(t *testing.T) (*session.Store, *pgxpool.Pool) {
	t.Helper()

	schema := databasetest.NewSchema(t)

	pool, err := database.Open(t.Context(), schema.URL, schema.Password, migrations.FS())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(pool.Close)

	return session.NewStore(pool), pool
}

// start starts a session in store and returns its token.
func start(t *testing.T, store *session.Store) string {
	t.Helper()

	token, err := store.Start(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	return token
}

// row is a stored session.
type row struct {
	Hash     []byte
	Lifetime time.Duration
}

// stored returns the sessions in pool's database.
func stored(t *testing.T, pool *pgxpool.Pool) []row {
	t.Helper()

	rows, err := pool.Query(t.Context(),
		"SELECT token_hash AS hash, expires_at - created_at AS lifetime FROM sessions")
	if err != nil {
		t.Fatal(err)
	}

	sessions, err := pgx.CollectRows(rows, pgx.RowToStructByName[row])
	if err != nil {
		t.Fatal(err)
	}

	return sessions
}
