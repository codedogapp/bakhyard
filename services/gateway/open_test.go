package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTheOpenGatewayLetsEveryRequestThrough(t *testing.T) {
	router := openGateway(t)

	for _, test := range []struct{ method, target, want string }{
		{http.MethodGet, "/api/home", "home got GET /home"},
		{http.MethodGet, "/api/notes/files/Inbox.md", "notes got GET /notes/files/Inbox.md"},
		{http.MethodGet, "/api/pulls/mine", "pulls got GET /pulls/mine"},
		{http.MethodPost, "/api/todos", "todos got POST /todos"},
		{http.MethodGet, "/", "web got GET /"},
		{http.MethodHead, "/todos", "web got HEAD /todos"},
		// No route signs the user in or out, so web gets these as it gets any other path.
		{http.MethodGet, "/auth/google/login", "web got GET /auth/google/login"},
		{http.MethodGet, "/auth/google/callback", "web got GET /auth/google/callback"},
		{http.MethodPost, "/auth/logout", "web got POST /auth/logout"},
	} {
		// No Tailscale login, a host other than the published one, and no session.
		rec := httptest.NewRecorder()
		router.ServeHTTP(
			rec,
			httptest.NewRequestWithContext(t.Context(), test.method, test.target, nil),
		)

		if got := rec.Header().Get("Got"); got != test.want {
			t.Errorf(
				"%s %s: got %d %q, want %q",
				test.method,
				test.target,
				rec.Code,
				got,
				test.want,
			)
		}
	}
}

func TestTheOpenGatewayKeepsEverythingButTheLocks(t *testing.T) {
	router := openGateway(t)

	withCookies := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/todos", nil)
	withCookies.Header.Set("Cookie", "__Host-session=token; theme=dark")

	crossSite := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/todos", nil)
	crossSite.Header.Set("Sec-Fetch-Site", "cross-site")

	for name, test := range map[string]struct {
		req        *http.Request
		code       int
		got        string
		gotCookies string
	}{
		"the health check": {
			httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/healthz", nil),
			http.StatusOK, "", "",
		},
		"an unknown API": {
			httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/nothing", nil),
			http.StatusNotFound, "", "",
		},
		"the session cookie":  {withCookies, http.StatusOK, "todos got GET /todos", "theme=dark"},
		"another site's POST": {crossSite, http.StatusForbidden, "", ""},
	} {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, test.req)

			if rec.Code != test.code || rec.Header().Get("Got") != test.got ||
				rec.Header().Get("Got-Cookies") != test.gotCookies {
				t.Errorf("got %d, %q with the cookies %q; want %d, %q with %q",
					rec.Code, rec.Header().Get("Got"), rec.Header().Get("Got-Cookies"),
					test.code, test.got, test.gotCookies)
			}

			if policy := rec.Header().Get("Content-Security-Policy"); policy != appPolicy {
				t.Errorf("got the policy %q, want the app's", policy)
			}
		})
	}
}

func TestTheOpenGatewayNeedsItsPeers(t *testing.T) {
	_, err := openRoutes(func(string) string { return "" })

	for _, name := range []string{"home", "notes", "pulls", "todos", "web"} {
		if err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("without BAKHYARD_PEERS: got %v, want an error that names %s", err, name)
		}
	}
}

func TestOnlyTheEndToEndBuildIsOpen(t *testing.T) {
	// The open gateway needs nothing but its peers; the real one needs its sign-in settings too.
	_, err := served(secretsWith(nil))(t.Context(), func(key string) string {
		if key == "BAKHYARD_PEERS" {
			return "home=tcp://127.0.0.1:1,notes=tcp://127.0.0.1:1,pulls=tcp://127.0.0.1:1" +
				",todos=tcp://127.0.0.1:1,web=tcp://127.0.0.1:1"
		}

		return ""
	})
	if err == nil || err.Error() != "BAKHYARD_TAILSCALE_LOGIN is not set" {
		t.Errorf("built without the e2e tag: got %v, want the real gateway's routes", err)
	}
}

// openGateway returns the routes of the gateway built for the end-to-end tests, with nothing but
// BAKHYARD_PEERS set. Each of its peers answers with a Got header that names the peer and the
// request, and a Got-Cookies header with the cookies it got.
func openGateway(t *testing.T) http.Handler {
	t.Helper()

	var peers []string

	for _, name := range []string{"home", "notes", "pulls", "todos", "web"} {
		peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Got", name+" got "+r.Method+" "+r.URL.Path)
			w.Header().Set("Got-Cookies", r.Header.Get("Cookie"))
		}))
		t.Cleanup(peer.Close)

		peers = append(peers, name+"=tcp://"+strings.TrimPrefix(peer.URL, "http://"))
	}

	router, err := openRoutes(func(key string) string {
		if key == "BAKHYARD_PEERS" {
			return strings.Join(peers, ",")
		}

		return ""
	})
	if err != nil {
		t.Fatal(err)
	}

	return router
}
