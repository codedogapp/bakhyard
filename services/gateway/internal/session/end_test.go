package session_test

import (
	"bytes"
	"crypto/sha256"
	"strings"
	"testing"
)

func TestEndEndsOnlyItsSession(t *testing.T) {
	store, pool := newStore(t)
	token, other := start(t, store), start(t, store)

	err := store.End(t.Context(), token)
	if err != nil {
		t.Fatal(err)
	}

	otherHash := sha256.Sum256([]byte(other))

	sessions := stored(t, pool)
	if len(sessions) != 1 || !bytes.Equal(sessions[0].Hash, otherHash[:]) {
		t.Errorf("got %d sessions, want only the other one", len(sessions))
	}
}

func TestEndEndsAnExpiredSessionToo(t *testing.T) {
	store, pool := newStore(t)
	token := start(t, store)
	hash := sha256.Sum256([]byte(token))
	expireIn(t, pool, hash[:], "1 second ago")

	err := store.End(t.Context(), token)
	if err != nil || len(stored(t, pool)) != 0 {
		t.Errorf("got %v and %d sessions, want none", err, len(stored(t, pool)))
	}
}

func TestEndFindsNoSessionToEndForAnUnknownToken(t *testing.T) {
	store, pool := newStore(t)
	start(t, store)

	err := store.End(t.Context(), "UNKNOWN")
	if err != nil || len(stored(t, pool)) != 1 {
		t.Errorf("got %v and %d sessions, want no error and the session", err, len(stored(t, pool)))
	}
}

func TestEndFailsWithoutTheDatabase(t *testing.T) {
	store, pool := newStore(t)
	pool.Close()

	err := store.End(t.Context(), "TOKEN")
	if err == nil || !strings.HasPrefix(err.Error(), "end a session: ") {
		t.Errorf("got %v, want an error that starts with end a session", err)
	}
}
