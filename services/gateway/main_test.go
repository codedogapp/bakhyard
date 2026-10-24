package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/services/gateway/internal/database"
	"github.com/codedogapp/bakhyard/services/gateway/internal/databasetest"
	"github.com/codedogapp/bakhyard/services/gateway/internal/session"
)

func TestTheGatewayAnswersItsHealthCheckItself(t *testing.T) {
	web := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("web got %s %s, want nothing", r.Method, r.URL.Path)
	}))
	t.Cleanup(web.Close)

	router, _ := gateway(t, web.URL)

	// No login and the loopback's Host, as process-compose's probe sends them.
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		req := httptest.NewRequestWithContext(t.Context(), method, "/healthz", nil)
		req.Host = "127.0.0.1:8080"

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK || rec.Body.String() != `{"status":"ok"}` {
			t.Errorf("%s /healthz: got %d %q, want the health check", method, rec.Code, rec.Body)
		}
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(
		rec,
		httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/healthz", nil),
	)

	if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != http.MethodGet {
		t.Errorf("POST /healthz: got %d, Allow %q; want 405, Allow GET",
			rec.Code, rec.Header().Get("Allow"))
	}
}

func TestTheGatewayForwardsHEADAsHEAD(t *testing.T) {
	got := make(chan string, 1)

	web := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got <- r.Method + " " + r.URL.Path
	}))
	t.Cleanup(web.Close)

	router, _ := gateway(t, web.URL)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, signedIn(t, http.MethodHead, "/notes"))

	select {
	case request := <-got:
		if request != "HEAD /notes" {
			t.Errorf("web got %q, want HEAD /notes", request)
		}
	default:
		t.Error("web got nothing, want HEAD /notes")
	}
}

func TestTheGatewayLetsTheUserIn(t *testing.T) {
	web := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("web got %s %s, want nothing", r.Method, r.URL.Path)
	}))
	t.Cleanup(web.Close)

	router, token := gateway(t, web.URL)

	// Nothing listens where the test gateway's home and notes are, so a 503 means the request
	// reached their forwarder. An /api path that no service claims keeps its 404 behind the lock.
	for target, want := range map[string]int{
		"/api/home":                 http.StatusServiceUnavailable,
		"/api/notes":                http.StatusServiceUnavailable,
		"/api/notes/files/Inbox.md": http.StatusServiceUnavailable,
		"/api/nothing":              http.StatusNotFound,
		"/api":                      http.StatusNotFound,
	} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, withSession(signedIn(t, http.MethodGet, target), token))

		if rec.Code != want {
			t.Errorf("GET %s: got %d, want %d", target, rec.Code, want)
		}
	}
}

func TestTheGatewayRefusesEveryoneElse(t *testing.T) {
	web := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("web got %s %s, want nothing", r.Method, r.URL.Path)
	}))
	t.Cleanup(web.Close)

	router, token := gateway(t, web.URL)

	// The Tailscale lock comes first: a session doesn't open it.
	for _, target := range []string{
		"/", "/notes", "/api/home", "/api/notes", "/api/notes/files/Inbox.md", "/api/nothing",
		"/auth/google/login", "/auth/google/callback",
	} {
		for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPost} {
			for _, req := range []*http.Request{
				httptest.NewRequestWithContext(t.Context(), method, target, nil),
				withSession(httptest.NewRequestWithContext(t.Context(), method, target, nil), token),
			} {
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)

				if rec.Code != http.StatusForbidden {
					t.Errorf("%s %s without a login: got %d, want 403", method, target, rec.Code)
				}
			}
		}
	}
}

func TestTheGatewayNeedsItsSettings(t *testing.T) {
	for _, name := range []string{
		"BAKHYARD_TAILSCALE_LOGIN", "BAKHYARD_HOST",
		"BAKHYARD_GOOGLE_CLIENT_ID", "BAKHYARD_GOOGLE_EMAIL", "BAKHYARD_DATABASE",
	} {
		vars, items := gatewayEnv(t, "http://127.0.0.1:2")
		delete(vars, name)

		_, err := routes(func(key string) string { return vars[key] }, keychainWith(items))
		if err == nil || err.Error() != name+" is not set" {
			t.Errorf("without %s: got %v, want the error %s is not set", name, err, name)
		}
	}
}

// The login and host that the test gateway lets in.
const (
	testLogin = "user@example.com"
	testHost  = "machine.tailnet.ts.net"
)

// signedIn returns a request as tailscale serve sends it for the test gateway's user.
func signedIn(t *testing.T, method, target string) *http.Request {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), method, target, nil)
	req.Host = testHost
	req.Header.Set("Tailscale-User-Login", testLogin)

	return req
}

// gateway returns the gateway's routes, with home and notes on a port where nothing listens, web
// at webURL and the database in a schema of its own, and the token of a session in that database.
func gateway(t *testing.T, webURL string) (http.Handler, string) {
	t.Helper()

	vars, items := gatewayEnv(t, webURL)

	return gatewayFrom(t, vars, items)
}

// gatewayFrom returns the gateway's routes for the environment vars and the Keychain items, and
// the token of a session in its database.
func gatewayFrom(t *testing.T, vars, items map[string]string) (http.Handler, string) {
	t.Helper()

	router, err := routes(func(key string) string { return vars[key] }, keychainWith(items))
	if err != nil {
		t.Fatal(err)
	}

	pool, err := database.Open(t.Context(), vars[databaseVar], items[databaseAccount])
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(pool.Close)

	token, err := session.NewStore(pool).Start(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	return router, token
}

// withSession returns req with token in its session cookie.
func withSession(req *http.Request, token string) *http.Request {
	req.AddCookie(session.Cookie(token))

	return req
}

// gatewayEnv returns the test gateway's environment and Keychain items, with home and notes on a
// port where nothing listens, web at webURL and the database in a schema of its own.
func gatewayEnv(t *testing.T, webURL string) (vars, items map[string]string) {
	t.Helper()

	schema := databasetest.NewSchema(t)
	vars = map[string]string{
		"BAKHYARD_PEERS": "home=tcp://127.0.0.1:1,notes=tcp://127.0.0.1:1,web=tcp://" +
			strings.TrimPrefix(webURL, "http://"),
		"BAKHYARD_TAILSCALE_LOGIN":  testLogin,
		"BAKHYARD_HOST":             testHost,
		"BAKHYARD_GOOGLE_CLIENT_ID": "client",
		"BAKHYARD_GOOGLE_EMAIL":     testLogin,
		"BAKHYARD_DATABASE":         schema.URL,
	}
	items = map[string]string{
		"google-client-secret": testSecret,
		"postgres-gateway":     schema.Password,
	}

	return vars, items
}
