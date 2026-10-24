package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The app's policy (docs/plan.md §4.4), and the stricter one of a vault file (decision 92).
const (
	appPolicy = "default-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; " +
		"font-src 'self' data:; connect-src 'self'; form-action 'self'; frame-ancestors 'none'; " +
		"base-uri 'none'"
	vaultPolicy = "default-src 'none'; style-src 'unsafe-inline'; sandbox"
)

func TestEveryResponseCarriesTheSecurityHeaders(t *testing.T) {
	notes := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Security-Policy", vaultPolicy)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		_, _ = io.WriteString(w, "# A note")
	}))
	t.Cleanup(notes.Close)

	web := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "<!doctype html>")
	}))
	t.Cleanup(web.Close)

	vars, items := gatewayEnv(t, web.URL)
	vars["BAKHYARD_PEERS"] = "home=tcp://127.0.0.1:1" +
		",notes=tcp://" + strings.TrimPrefix(notes.URL, "http://") +
		",web=tcp://" + strings.TrimPrefix(web.URL, "http://")
	router, token := gatewayFrom(t, vars, items)

	crossSite := withSession(signedIn(t, http.MethodPost, "/auth/logout"), token)
	crossSite.Header.Set("Sec-Fetch-Site", "cross-site")

	for name, test := range map[string]struct {
		req    *http.Request
		code   int
		policy string
	}{
		"an app file": {signedIn(t, http.MethodGet, "/"), http.StatusOK, appPolicy},
		"a vault file": {
			withSession(signedIn(t, http.MethodGet, "/api/notes/files/A.md"), token),
			http.StatusOK, vaultPolicy,
		},
		"a service that's down": {
			withSession(signedIn(t, http.MethodGet, "/api/home"), token),
			http.StatusServiceUnavailable, appPolicy,
		},
		"an unknown API": {
			withSession(signedIn(t, http.MethodGet, "/api/nothing"), token),
			http.StatusNotFound, appPolicy,
		},
		"the health check": {
			httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/healthz", nil),
			http.StatusOK, appPolicy,
		},
		"a method the route lacks": {
			httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/healthz", nil),
			http.StatusMethodNotAllowed, appPolicy,
		},
		"no login": {
			httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil),
			http.StatusForbidden, appPolicy,
		},
		"no session": {
			signedIn(t, http.MethodGet, "/api/home"), http.StatusUnauthorized, appPolicy,
		},
		"another site's POST": {crossSite, http.StatusForbidden, appPolicy},
		"the way to Google": {
			signedIn(t, http.MethodGet, "/auth/google/login"), http.StatusFound, appPolicy,
		},
	} {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, test.req)

			if rec.Code != test.code {
				t.Errorf("got %d, want %d", rec.Code, test.code)
			}

			for key, want := range map[string]string{
				"Content-Security-Policy": test.policy,
				"X-Content-Type-Options":  "nosniff",
				"Referrer-Policy":         "same-origin",
			} {
				if got := rec.Result().Header.Values(key); len(got) != 1 || got[0] != want {
					t.Errorf("%s is %q, want only %q", key, got, want)
				}
			}
		})
	}
}
