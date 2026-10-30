package history_test

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/codedogapp/bakhyard/services/notify/internal/alert"
)

// count returns how many alerts the database holds.
func count(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()

	var rows int
	if err := pool.QueryRow(t.Context(), "SELECT count(*) FROM alerts").Scan(&rows); err != nil {
		t.Fatal(err)
	}

	return rows
}

func TestAlertsResolvedMoreThan30DaysAgoAreGone(t *testing.T) {
	store, pool := newStore(t)
	now := time.Now().Truncate(time.Second)
	long := firing("DiskFull", now.Add(-40*24*time.Hour))
	heat := resolved(firing("Hot", now.Add(-2*time.Hour)), now.Add(-time.Hour))
	old := resolved(firing("Old", now.Add(-32*24*time.Hour)), now.Add(-31*24*time.Hour))

	keep(t, store, long, heat, old)
	expect(t, store, entry(long), entry(heat))

	if rows := count(t, pool); rows != 2 {
		t.Errorf("%d alerts left, want the one resolved 31 days ago gone", rows)
	}

	_, err := pool.Exec(t.Context(),
		"UPDATE alerts SET resolved_at = now() - interval '31 days' WHERE name = 'Hot'")
	if err != nil {
		t.Fatal(err)
	}

	expect(t, store, entry(long))
	keep(t, store, long)

	if rows := count(t, pool); rows != 1 {
		t.Errorf("%d alerts left, want only the one that fires", rows)
	}
}

func TestKeepKeepsNothingWhenAnAlertFails(t *testing.T) {
	store, _ := newStore(t)
	now := time.Now().Truncate(time.Second)
	broken := firing("Broken", now)
	broken.Annotations = map[string]string{"summary": "a NUL \x00 that text can't hold"}

	if err := store.Keep(t.Context(), []alert.Alert{firing("Fine", now), broken}); err == nil {
		t.Error("Keep succeeded")
	}

	expect(t, store)
}

func TestTheStoreFailsWithoutTheDatabase(t *testing.T) {
	store, pool := newStore(t)
	pool.Close()

	if err := store.Keep(t.Context(), []alert.Alert{firing("DiskFull", time.Now())}); err == nil {
		t.Error("Keep succeeded")
	}

	if _, err := store.List(t.Context()); err == nil {
		t.Error("List succeeded")
	}
}
