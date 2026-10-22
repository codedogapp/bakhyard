package session_test

import (
	"testing"

	"github.com/codedogapp/bakhyard/services/gateway/internal/session"
)

func TestTheCookieKeepsTheTokenFromScriptsAndOtherSites(t *testing.T) {
	cookie := session.Cookie("TOKEN")

	want := "__Host-session=TOKEN; Path=/; Max-Age=604800; HttpOnly; Secure; SameSite=Lax"
	if got := cookie.String(); got != want {
		t.Errorf("got the cookie %q, want %q", got, want)
	}

	err := cookie.Valid()
	if err != nil {
		t.Errorf("got an invalid cookie: %v", err)
	}
}
