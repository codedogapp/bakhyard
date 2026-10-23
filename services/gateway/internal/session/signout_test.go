package session_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/services/gateway/internal/session"
)

// dropped is the cookie that makes the browser drop the session's cookie.
const dropped = "__Host-session=; Path=/; Max-Age=0; HttpOnly; Secure; SameSite=Lax"

func TestSignOutEndsTheSessionAndDropsItsCookie(t *testing.T) {
	sessions := &fakeEnder{}
	handler, logs := signingOut(sessions)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, request(t, http.MethodPost, "/auth/logout", liveToken))

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/" {
		t.Fatalf("got %d to %q, want 303 to /", rec.Code, rec.Header().Get("Location"))
	}

	if got := rec.Header().Get("Set-Cookie"); got != dropped {
		t.Errorf("got the cookie %q, want %q", got, dropped)
	}

	if !slices.Equal(sessions.ended, []string{liveToken}) {
		t.Errorf("the store ended %q, want only %s", sessions.ended, liveToken)
	}

	if !strings.Contains(logs.String(), `"msg":"signed out"`) {
		t.Errorf("logs = %q, want signed out", logs.String())
	}
}

func TestSignOutWithoutASessionStillDropsTheCookie(t *testing.T) {
	// Without a cookie there's nothing to ask the store; an unknown or expired token ends nothing.
	for _, test := range []struct {
		token string
		ended []string
	}{
		{"", nil},
		{"UNKNOWN", []string{"UNKNOWN"}},
	} {
		sessions := &fakeEnder{}
		handler, _ := signingOut(sessions)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, request(t, http.MethodPost, "/auth/logout", test.token))

		if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/" ||
			rec.Header().Get("Set-Cookie") != dropped {
			t.Errorf("token %q: got %d to %q with the cookie %q; want 303 to / with %q",
				test.token, rec.Code, rec.Header().Get("Location"),
				rec.Header().Get("Set-Cookie"), dropped)
		}

		if !slices.Equal(sessions.ended, test.ended) {
			t.Errorf("token %q: ended %q, want %q", test.token, sessions.ended, test.ended)
		}
	}
}

func TestSignOutFailsWhenTheStoreFails(t *testing.T) {
	const token = "SECRET-TOKEN"

	handler, logs := signingOut(&fakeEnder{err: errors.New("the database is down")})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, request(t, http.MethodPost, "/auth/logout", token))

	if rec.Code != http.StatusInternalServerError ||
		!strings.HasPrefix(rec.Header().Get("Content-Type"), "text/plain") {
		t.Fatalf("got %d, %q; want a plain-text 500", rec.Code, rec.Header().Get("Content-Type"))
	}

	// The cookie stays, so that signing out again can end the session.
	if rec.Header().Get("Set-Cookie") != "" || rec.Header().Get("Location") != "" {
		t.Errorf("got the cookie %q and the location %q, want neither",
			rec.Header().Get("Set-Cookie"), rec.Header().Get("Location"))
	}

	for _, want := range []string{`"msg":"failed a sign-out"`, "the database is down"} {
		if !strings.Contains(logs.String(), want) {
			t.Errorf("logs = %q, want them to contain %q", logs.String(), want)
		}
	}

	if strings.Contains(logs.String(), token) {
		t.Errorf("logs = %q, want them without the token", logs.String())
	}
}

// fakeEnder is a store that ends any session, or fails with err.
type fakeEnder struct {
	err   error
	ended []string
}

func (f *fakeEnder) End(_ context.Context, token string) error {
	f.ended = append(f.ended, token)

	return f.err
}

// signingOut returns the sign-out handler on sessions, and its logs.
func signingOut(sessions *fakeEnder) (http.Handler, *bytes.Buffer) {
	logs := &bytes.Buffer{}

	return session.SignOut(sessions, slog.New(slog.NewJSONHandler(logs, nil))), logs
}
