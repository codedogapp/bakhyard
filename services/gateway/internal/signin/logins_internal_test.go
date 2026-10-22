package signin

import (
	"testing"
	"testing/synctest"
	"time"
)

func TestALoginIsTakenOnce(t *testing.T) {
	logins := newLogins()
	logins.add("state", pending{nonce: "nonce", verifier: "verifier"})

	got, ok := logins.take("state")
	if !ok || got.nonce != "nonce" || got.verifier != "verifier" {
		t.Errorf("first take: got %+v, %t; want the login", got, ok)
	}

	if _, ok := logins.take("state"); ok {
		t.Error("second take: got the login, want nothing")
	}
}

func TestALoginExpiresAfterTenMinutes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		logins := newLogins()
		logins.add("early", pending{})
		logins.add("late", pending{})

		time.Sleep(loginTTL - time.Nanosecond)

		if _, ok := logins.take("early"); !ok {
			t.Error("just before ten minutes: got nothing, want the login")
		}

		time.Sleep(time.Nanosecond)

		if _, ok := logins.take("late"); ok {
			t.Error("at ten minutes: got the login, want nothing")
		}
	})
}

func TestANewLoginDropsTheExpiredOnes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		logins := newLogins()
		logins.add("old", pending{})
		time.Sleep(loginTTL / 2)
		logins.add("recent", pending{})
		time.Sleep(loginTTL / 2)
		logins.add("new", pending{})

		// Only old expired: it went in loginTTL ago.
		if _, kept := logins.byState["old"]; kept || len(logins.byState) != 2 {
			t.Errorf("got %d logins, old kept %t; want recent and new only",
				len(logins.byState), kept)
		}
	})
}
