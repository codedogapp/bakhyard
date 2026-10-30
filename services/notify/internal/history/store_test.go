package history_test

import (
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/codedogapp/bakhyard/internal/database"
	"github.com/codedogapp/bakhyard/internal/databasetest"
	"github.com/codedogapp/bakhyard/services/notify/internal/alert"
	"github.com/codedogapp/bakhyard/services/notify/internal/history"
	"github.com/codedogapp/bakhyard/services/notify/internal/migrations"
)

// newStore returns a store in a migrated schema of its own, and the pool that reaches it.
func newStore(t *testing.T) (*history.Store, *pgxpool.Pool) {
	t.Helper()

	schema := databasetest.NewSchema(t)

	pool, err := database.Open(t.Context(), schema.URL, schema.Password, migrations.FS())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(pool.Close)

	return history.NewStore(pool), pool
}

// firing returns the alert name, as Alertmanager posts it while it fires since started.
func firing(name string, started time.Time) alert.Alert {
	return alert.Alert{
		Status: alert.Firing,
		Labels: map[string]string{"alertname": name, "severity": "warning"},
		Annotations: map[string]string{
			"summary":     name + " fires",
			"description": "Look at " + name,
		},
		StartsAt:    started,
		Fingerprint: "fp-" + name,
	}
}

// resolved returns found as Alertmanager posts it once it resolved at ended.
func resolved(found alert.Alert, ended time.Time) alert.Alert {
	found.Status = alert.Resolved
	found.EndsAt = ended

	return found
}

// entry returns the entry that the store keeps for found.
func entry(found alert.Alert) history.Entry {
	return history.Entry{
		Fingerprint: found.Fingerprint,
		Name:        found.Labels["alertname"],
		Summary:     found.Annotations["summary"],
		Description: found.Annotations["description"],
		StartedAt:   found.StartsAt,
		ResolvedAt:  found.EndsAt,
	}
}

func keep(t *testing.T, store *history.Store, alerts ...alert.Alert) {
	t.Helper()

	if err := store.Keep(t.Context(), alerts); err != nil {
		t.Fatal(err)
	}
}

// expect fails t unless the store lists want, at the same instants.
func expect(t *testing.T, store *history.Store, want ...history.Entry) {
	t.Helper()

	got, err := store.List(t.Context())
	if err != nil || !slices.EqualFunc(got, want, func(a, b history.Entry) bool {
		return a.Fingerprint == b.Fingerprint && a.Name == b.Name && a.Summary == b.Summary &&
			a.Description == b.Description && a.StartedAt.Equal(b.StartedAt) &&
			a.ResolvedAt.Equal(b.ResolvedAt)
	}) {
		t.Errorf("listed %+v (%v), want %+v", got, err, want)
	}
}

func TestListShowsTheFiringAlertsFirstAndThenTheResolvedOnesLatestFirst(t *testing.T) {
	store, _ := newStore(t)
	now := time.Now().Truncate(time.Second)
	disk := firing("DiskFull", now.Add(-3*time.Hour))
	battery := firing("BatteryLow", now.Add(-time.Hour))
	heat := resolved(firing("Hot", now.Add(-2*time.Hour)), now.Add(-90*time.Minute))
	aging := resolved(firing("Old", now.Add(-29*24*time.Hour)), now.Add(-29*24*time.Hour))

	keep(t, store, disk, heat)
	keep(t, store, aging, battery)

	expect(t, store, entry(battery), entry(disk), entry(heat), entry(aging))
}

func TestKeepFollowsAnAlertFromFiringToResolvedAndAgain(t *testing.T) {
	store, _ := newStore(t)
	now := time.Now().Truncate(time.Second)
	first := firing("BatteryLow", now.Add(-time.Hour))

	keep(t, store, first)
	expect(t, store, entry(first))

	worse := firing("BatteryLow", first.StartsAt)
	worse.Annotations = map[string]string{"summary": "Battery at 12%", "description": "Plug it"}
	keep(t, store, worse)
	expect(t, store, entry(worse))

	ended := resolved(worse, now.Add(-time.Minute))
	keep(t, store, ended)
	keep(t, store, worse, resolved(worse, now))
	expect(t, store, entry(ended))

	again := firing("BatteryLow", now)
	keep(t, store, again)
	expect(t, store, entry(again), entry(ended))
}

func TestKeepSkipsTheAlertsItCantTellApart(t *testing.T) {
	store, _ := newStore(t)
	now := time.Now().Truncate(time.Second)
	kept := firing("Kept", now)
	noFingerprint := firing("NoFingerprint", now)
	noFingerprint.Fingerprint = ""
	pending := firing("Pending", now)
	pending.Status = "pending"
	noEnd := firing("NoEnd", now.Add(-time.Minute))

	keep(t, store, noEnd)
	keep(t, store, firing("NoStart", time.Time{}), noFingerprint, pending,
		resolved(noEnd, time.Time{}), kept)

	expect(t, store, entry(kept), entry(noEnd))
}

func TestListReturnsTheLatest100(t *testing.T) {
	store, _ := newStore(t)
	now := time.Now().Truncate(time.Second)

	var alerts []alert.Alert

	want := make([]history.Entry, 0, 100)

	for i := range 101 {
		found := firing("Alert"+strconv.Itoa(i), now.Add(-time.Duration(i)*time.Minute))
		alerts = append(alerts, found)

		if i < 100 {
			want = append(want, entry(found))
		}
	}

	keep(t, store, alerts...)

	expect(t, store, want...)
}
