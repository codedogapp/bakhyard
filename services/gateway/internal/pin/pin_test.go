package pin_test

import (
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/codedogapp/bakhyard/services/gateway/internal/database"
	"github.com/codedogapp/bakhyard/services/gateway/internal/databasetest"
	"github.com/codedogapp/bakhyard/services/gateway/internal/pin"
)

// email is the allowed email under test.
const email = "user@example.com"

func TestPinPinsTheFirstAccount(t *testing.T) {
	store, pool := newStore(t)

	if !pinned(t, store, email, "first") {
		t.Error("got another account pinned, want the first")
	}

	got := stored(t, pool)
	if len(got) != 1 || got[0].Email != email || got[0].Sub != "first" {
		t.Errorf("got the pins %+v, want %s pinned to first", got, email)
	}
}

func TestPinKeepsThePinnedAccount(t *testing.T) {
	store, pool := newStore(t)
	pinned(t, store, email, "first")
	before := stored(t, pool)

	for _, step := range []struct {
		sub  string
		want bool
	}{{"first", true}, {"second", false}, {"first", true}} {
		if got := pinned(t, store, email, step.sub); got != step.want {
			t.Errorf("%s: got %t, want %t", step.sub, got, step.want)
		}
	}

	if after := stored(t, pool); !slices.EqualFunc(after, before, sameRow) {
		t.Errorf("the pins went from %+v to %+v, want them as they were", before, after)
	}
}

func TestPinPinsEachEmailOnItsOwn(t *testing.T) {
	store, pool := newStore(t)

	for _, step := range []struct {
		email, sub string
		want       bool
	}{
		{"a@example.com", "first", true},
		{"b@example.com", "second", true},
		{"a@example.com", "second", false},
		{"b@example.com", "first", false},
	} {
		if got := pinned(t, store, step.email, step.sub); got != step.want {
			t.Errorf("%s to %s: got %t, want %t", step.sub, step.email, got, step.want)
		}
	}

	got := stored(t, pool)
	if len(got) != 2 || got[0].Sub != "first" || got[1].Sub != "second" {
		t.Errorf("got the pins %+v, want a to first and b to second", got)
	}
}

func TestPinAgreesOnOneAccountAtOnce(t *testing.T) {
	store, pool := newStore(t)

	const accounts = 8

	// Each account writes only its own entry.
	won := make([]bool, accounts)

	var group sync.WaitGroup
	for account := range accounts {
		group.Go(func() {
			ok, err := store.Pin(t.Context(), email, strconv.Itoa(account))
			if err != nil {
				t.Error(err)
			}

			won[account] = ok
		})
	}

	group.Wait()

	winner := slices.Index(won, true)
	if winner < 0 || slices.Contains(won[winner+1:], true) {
		t.Fatalf("got the accounts that won %v, want exactly one", won)
	}

	if got := stored(t, pool); len(got) != 1 || got[0].Sub != strconv.Itoa(winner) {
		t.Errorf("got the pins %+v, want %s pinned to %d", got, email, winner)
	}
}

func TestPinFailsWithoutTheDatabase(t *testing.T) {
	store, pool := newStore(t)
	pool.Close()

	_, err := store.Pin(t.Context(), email, "first")
	if err == nil || !strings.HasPrefix(err.Error(), "pin the account: ") {
		t.Errorf("got %v, want an error that starts with pin the account", err)
	}
}

// newStore returns a store in a migrated schema of its own, and the pool that reaches it.
func newStore(t *testing.T) (*pin.Store, *pgxpool.Pool) {
	t.Helper()

	schema := databasetest.NewSchema(t)

	pool, err := database.Open(t.Context(), schema.URL, schema.Password)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(pool.Close)

	return pin.NewStore(pool), pool
}

// pinned pins sub to email in store, and reports whether sub is the account pinned to it.
func pinned(t *testing.T, store *pin.Store, email, sub string) bool {
	t.Helper()

	ok, err := store.Pin(t.Context(), email, sub)
	if err != nil {
		t.Fatal(err)
	}

	return ok
}

// row is a stored pin.
type row struct {
	Email    string    `db:"email"`
	Sub      string    `db:"sub"`
	PinnedAt time.Time `db:"pinned_at"`
}

func sameRow(a, b row) bool {
	return a.Email == b.Email && a.Sub == b.Sub && a.PinnedAt.Equal(b.PinnedAt)
}

// stored returns the pins in pool's database, by email.
func stored(t *testing.T, pool *pgxpool.Pool) []row {
	t.Helper()

	rows, err := pool.Query(t.Context(), "SELECT email, sub, pinned_at FROM pins ORDER BY email")
	if err != nil {
		t.Fatal(err)
	}

	pins, err := pgx.CollectRows(rows, pgx.RowToStructByName[row])
	if err != nil {
		t.Fatal(err)
	}

	return pins
}
