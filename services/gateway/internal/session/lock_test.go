package session_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/services/gateway/internal/session"
)

// liveToken is the token of the only live session in the fake store.
const liveToken = "LIVE"

func TestTheLockLetsALiveSessionInAndRenewsItsCookie(t *testing.T) {
	sessions := &fakeSessions{}
	handler, _, reached := locked(sessions)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, request(t, http.MethodGet, "/api/home", liveToken))

	if rec.Code != http.StatusOK || len(*reached) != 1 {
		t.Fatalf("got %d, %d requests passed on; want 200 and the request", rec.Code, len(*reached))
	}

	want := session.Cookie(liveToken).String()
	if got := rec.Header().Get("Set-Cookie"); got != want {
		t.Errorf("got the cookie %q, want %q", got, want)
	}

	if len(sessions.asked) != 1 || sessions.asked[0] != liveToken {
		t.Errorf("the store renewed %q, want only %s", sessions.asked, liveToken)
	}
}

func TestTheLockAnswersARequestWithoutASessionWith401(t *testing.T) {
	for _, req := range []*http.Request{
		request(t, http.MethodGet, "/api/home", ""),
		request(t, http.MethodGet, "/api", "UNKNOWN"),
		request(t, http.MethodHead, "/api/home", "UNKNOWN"),
		request(t, http.MethodPost, "/api/home", ""),
		// A page load too: the lock no longer sends one to sign in (decision 86).
		request(t, http.MethodGet, "/", ""),
	} {
		handler, logs, reached := locked(&fakeSessions{})

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized || len(*reached) != 0 {
			t.Errorf("%s %s: got %d, want 401", req.Method, req.URL.Path, rec.Code)
		}

		if rec.Header().Get("Set-Cookie") != "" || rec.Header().Get("Location") != "" {
			t.Errorf("%s %s: got a cookie or a redirect, want neither", req.Method, req.URL.Path)
		}

		if !strings.Contains(logs.String(), `"msg":"no session"`) {
			t.Errorf("%s %s: got the logs %s, want no session", req.Method, req.URL.Path, logs)
		}
	}
}

func TestTheLockAsksTheStoreOnlyAboutAToken(t *testing.T) {
	sessions := &fakeSessions{}
	handler, _, _ := locked(sessions)

	handler.ServeHTTP(httptest.NewRecorder(), request(t, http.MethodGet, "/api/home", ""))

	if len(sessions.asked) != 0 {
		t.Errorf("the store renewed %q, want nothing", sessions.asked)
	}
}

func TestTheLockFailsWhenTheStoreFails(t *testing.T) {
	handler, logs, reached := locked(&fakeSessions{err: errors.New("the database is down")})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, request(t, http.MethodGet, "/api/home", liveToken))

	if rec.Code != http.StatusInternalServerError || len(*reached) != 0 {
		t.Errorf("got %d, want 500", rec.Code)
	}

	if !strings.Contains(logs.String(), "the database is down") {
		t.Errorf("got the logs %s, want the store's error", logs)
	}

	if strings.Contains(logs.String(), liveToken) {
		t.Errorf("got the logs %s, want no token in them", logs)
	}
}

// fakeSessions is a store that has only the session of liveToken, or fails with err.
type fakeSessions struct {
	err   error
	asked []string
}

func (f *fakeSessions) Renew(_ context.Context, token string) (bool, error) {
	f.asked = append(f.asked, token)

	return token == liveToken, f.err
}

// locked returns the lock in front of a handler that answers 200, the lock's logs, and the
// requests that handler got.
func locked(sessions *fakeSessions) (http.Handler, *bytes.Buffer, *[]*http.Request) {
	logs := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logs, nil))
	reached := &[]*http.Request{}

	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		*reached = append(*reached, r)
	})

	return session.Require(sessions, logger)(next), logs, reached
}

// request returns a request for target, with token in the session cookie unless it's empty.
func request(t *testing.T, method, target, token string) *http.Request {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), method, target, nil)
	if token != "" {
		req.AddCookie(session.Cookie(token))
	}

	return req
}
