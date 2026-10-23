package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTheGatewayNeedsASessionForEverythingButSigningIn(t *testing.T) {
	web := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("web got %s %s, want nothing", r.Method, r.URL.Path)
	}))
	t.Cleanup(web.Close)

	router, _ := gateway(t, web.URL)

	// A page load goes to sign in; anything else gets a 401, and reaches no service.
	for _, test := range []struct {
		method, target string
		want           int
	}{
		{http.MethodGet, "/", http.StatusFound},
		{http.MethodHead, "/notes", http.StatusFound},
		{http.MethodGet, "/auth/other", http.StatusFound},
		{http.MethodGet, "/api/home", http.StatusUnauthorized},
		{http.MethodGet, "/api/nothing", http.StatusUnauthorized},
		{http.MethodPost, "/", http.StatusUnauthorized},
	} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, signedIn(t, test.method, test.target))

		location := rec.Header().Get("Location")

		wrongPlace := test.want == http.StatusFound && location != "/auth/google/login"
		if rec.Code != test.want || wrongPlace {
			t.Errorf("%s %s without a session: got %d to %q, want %d",
				test.method, test.target, rec.Code, location, test.want)
		}
	}
}

func TestTheGatewayRenewsTheSessionAndKeepsItFromTheServices(t *testing.T) {
	cookies := make(chan string, 1)

	web := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		cookies <- r.Header.Get("Cookie")
	}))
	t.Cleanup(web.Close)

	router, token := gateway(t, web.URL)

	req := withSession(signedIn(t, http.MethodGet, "/notes"), token)
	req.Header.Add("Cookie", "theme=dark")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	want := "__Host-session=" + token +
		"; Path=/; Max-Age=604800; HttpOnly; Secure; SameSite=Lax"
	if rec.Code != http.StatusOK || rec.Header().Get("Set-Cookie") != want {
		t.Errorf("got %d with the cookie %q, want 200 with %q",
			rec.Code, rec.Header().Get("Set-Cookie"), want)
	}

	select {
	case got := <-cookies:
		if got != "theme=dark" {
			t.Errorf("web got the cookies %q, want only theme=dark", got)
		}
	default:
		t.Error("web got nothing, want GET /notes")
	}
}
