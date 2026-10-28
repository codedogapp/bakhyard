package subscription_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/codedogapp/bakhyard/internal/database"
	"github.com/codedogapp/bakhyard/internal/databasetest"
	"github.com/codedogapp/bakhyard/services/notify/internal/migrations"
	"github.com/codedogapp/bakhyard/services/notify/internal/subscription"
)

// The keys of RFC 8291, Appendix A: the user agent's, and the application server's public key as a
// second valid P-256 point.
const (
	p256dh      = "BCVxsr7N_eNgVRqvHtD0zTZsEc6-VV-JvLexhqUzORcxaOzi6-AYWXvTBHm4bjyPjs7Vd8pZGH6SRpkNtoIAiw4"
	auth        = "BTBZMqHH6r4Tts7J_aSIgg"
	otherP256dh = "BP4z9KsN6nGRTbVYI_c7VJSPQTBtkgcy27mlmlMoZIIgDll6e3vCYLocInmYWAmS6TlzAC8wEqKK6PBru3jl7A8"
)

func valid(endpoint string) subscription.Subscription {
	return subscription.Subscription{
		Endpoint: endpoint,
		Keys:     subscription.Keys{P256dh: p256dh, Auth: auth},
	}
}

func TestAddRefusesAnInvalidSubscription(t *testing.T) {
	withKeys := func(key, secret string) subscription.Subscription {
		sub := valid("https://push.example/send/1")
		sub.Keys = subscription.Keys{P256dh: key, Auth: secret}

		return sub
	}

	tests := map[string]subscription.Subscription{
		"no endpoint":      valid(""),
		"an http endpoint": valid("http://push.example/send/1"),
		"no host":          valid("https:///send/1"),
		"a bad URL":        valid("https://push.example/%zz"),
		"p256dh not b64":   withKeys("not base64 !", auth),
		"p256dh not a key": withKeys("AAAA", auth),
		"auth not b64":     withKeys(p256dh, "not base64 !"),
		"auth too short":   withKeys(p256dh, "AAAA"),
	}

	for name, sub := range tests {
		t.Run(name, func(t *testing.T) {
			store, _ := newStore(t)

			err := store.Add(t.Context(), sub)
			if !errors.Is(err, subscription.ErrInvalid) {
				t.Errorf("got %v, want ErrInvalid", err)
			}

			if got, _ := store.List(t.Context()); len(got) != 0 {
				t.Errorf("stored %+v", got)
			}
		})
	}
}

func TestAddKeepsSubscriptionsOldestFirstAndReplacesTheKeysOfOneEndpoint(t *testing.T) {
	store, _ := newStore(t)
	ctx := t.Context()

	for _, endpoint := range []string{"https://push.example/a", "https://push.example/b"} {
		if err := store.Add(ctx, valid(endpoint)); err != nil {
			t.Fatal(err)
		}
	}

	changed := valid("https://push.example/a")
	changed.Keys.P256dh = otherP256dh

	if err := store.Add(ctx, changed); err != nil {
		t.Fatal(err)
	}

	got, err := store.List(ctx)
	want := []subscription.Subscription{changed, valid("https://push.example/b")}

	if err != nil || !slices.Equal(got, want) {
		t.Errorf("got %+v (%v), want %+v", got, err, want)
	}
}

func TestRemoveForgetsOneEndpointAndIgnoresAnotherThatIsNotThere(t *testing.T) {
	store, _ := newStore(t)
	ctx := t.Context()

	if err := store.Add(ctx, valid("https://push.example/a")); err != nil {
		t.Fatal(err)
	}

	for _, endpoint := range []string{"https://push.example/a", "https://push.example/none"} {
		if err := store.Remove(ctx, endpoint); err != nil {
			t.Fatalf("Remove(%q): %v", endpoint, err)
		}
	}

	if got, err := store.List(ctx); err != nil || len(got) != 0 {
		t.Errorf("got %+v (%v), want none", got, err)
	}
}

func TestTheStoreFailsWithoutTheDatabase(t *testing.T) {
	store, pool := newStore(t)
	pool.Close()

	ctx := t.Context()

	if err := store.Add(ctx, valid("https://push.example/a")); err == nil {
		t.Error("Add succeeded")
	}

	if _, err := store.List(ctx); err == nil {
		t.Error("List succeeded")
	}

	if err := store.Remove(ctx, "https://push.example/a"); err == nil {
		t.Error("Remove succeeded")
	}
}

// newStore returns a store in a migrated schema of its own, and the pool that reaches it.
func newStore(t *testing.T) (*subscription.Store, *pgxpool.Pool) {
	t.Helper()

	schema := databasetest.NewSchema(t)

	pool, err := database.Open(t.Context(), schema.URL, schema.Password, migrations.FS())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(pool.Close)

	return subscription.NewStore(pool), pool
}
