package signin_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/services/gateway/internal/signin"
)

// The client, user and host of the flow under test.
const (
	clientID = "client"
	secret   = "client-secret"
	email    = "user@example.com"
	host     = "machine.tailnet.ts.net"
)

func TestTheLoginGoesToGoogle(t *testing.T) {
	flow := newFlow(t, signin.Google())

	target := login(t, flow)
	if got := target.Scheme + "://" + target.Host + target.Path; got != google {
		t.Errorf("the login went to %s, want %s", got, google)
	}

	query := target.Query()
	for name, want := range map[string]string{
		"client_id":             clientID,
		"redirect_uri":          "https://machine.tailnet.ts.net/auth/google/callback",
		"response_type":         "code",
		"scope":                 "openid email",
		"code_challenge_method": "S256",
		"prompt":                "select_account",
	} {
		if got := query.Get(name); got != want {
			t.Errorf("%s: got %q, want %q", name, got, want)
		}
	}

	again := login(t, flow).Query()
	for _, name := range []string{"state", "nonce", "code_challenge"} {
		if query.Get(name) == "" || query.Get(name) == again.Get(name) {
			t.Errorf("%s: got %q, then %q; want a new one for each login",
				name, query.Get(name), again.Get(name))
		}
	}
}

// google is where Google signs the user in.
const google = "https://accounts.google.com/o/oauth2/v2/auth"

// testFlow is a sign-in flow's routes, with its log and its sessions.
type testFlow struct {
	http.Handler

	logs     *bytes.Buffer
	sessions *fakeSessions
}

// newFlow returns a sign-in flow with provider's endpoints.
func newFlow(t *testing.T, provider oidc.ProviderConfig) *testFlow {
	t.Helper()

	flow := &testFlow{logs: new(bytes.Buffer), sessions: new(fakeSessions)}

	router := chi.NewRouter()
	signin.New(signin.Settings{
		Provider:     provider,
		ClientID:     clientID,
		ClientSecret: secret,
		Email:        email,
		Host:         host,
	}, flow.sessions, slog.New(slog.NewTextHandler(flow.logs, nil))).Register(router)
	flow.Handler = router

	return flow
}

// login starts a login at flow and returns where it sends the browser.
func login(t *testing.T, flow http.Handler) *url.URL {
	t.Helper()

	rec := httptest.NewRecorder()
	flow.ServeHTTP(rec, httptest.NewRequestWithContext(
		t.Context(), http.MethodGet, "/auth/google/login", nil,
	))

	if rec.Code != http.StatusFound {
		t.Fatalf("the login: got %d, want 302", rec.Code)
	}

	target, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}

	return target
}

// callback brings Google's answer, query, back to flow.
func callback(t *testing.T, flow http.Handler, query url.Values) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	flow.ServeHTTP(rec, httptest.NewRequestWithContext(
		t.Context(), http.MethodGet, "/auth/google/callback?"+query.Encode(), nil,
	))

	return rec
}

// refused brings Google's answer, query, back to flow, and fails t unless flow refuses it: a 403
// without a cookie, no session started, and reason in the log.
func refused(t *testing.T, flow *testFlow, query url.Values, reason string) {
	t.Helper()

	before := len(flow.sessions.started())
	rec := callback(t, flow, query)

	if rec.Code != http.StatusForbidden || rec.Body.String() != "Forbidden\n" {
		t.Errorf("got %d %q, want 403 Forbidden", rec.Code, rec.Body)
	}

	cookies, started := rec.Header().Values("Set-Cookie"), len(flow.sessions.started())-before
	if len(cookies) != 0 || started != 0 {
		t.Errorf("got the cookies %q and %d sessions, want none", cookies, started)
	}

	logs, line := flow.logs.String(), `level=WARN msg="refused a sign-in" reason=`
	if !strings.Contains(logs, line) || !strings.Contains(logs, reason) {
		t.Errorf("got the log %q, want %s… with %q", logs, line, reason)
	}
}
