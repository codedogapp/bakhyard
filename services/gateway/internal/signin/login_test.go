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
	flow, _ := newFlow(t, signin.Google())

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

// newFlow returns the routes of a sign-in flow with provider's endpoints, and its log.
func newFlow(t *testing.T, provider oidc.ProviderConfig) (http.Handler, *bytes.Buffer) {
	t.Helper()

	var logs bytes.Buffer

	router := chi.NewRouter()
	signin.New(signin.Settings{
		Provider:     provider,
		ClientID:     clientID,
		ClientSecret: secret,
		Email:        email,
		Host:         host,
	}, slog.New(slog.NewTextHandler(&logs, nil))).Register(router)

	return router, &logs
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

// refused fails t unless rec is a 403 and logs give reason for it.
func refused(t *testing.T, rec *httptest.ResponseRecorder, logs *bytes.Buffer, reason string) {
	t.Helper()

	if rec.Code != http.StatusForbidden || rec.Body.String() != "Forbidden\n" {
		t.Errorf("got %d %q, want 403 Forbidden", rec.Code, rec.Body)
	}

	line := `level=WARN msg="refused a sign-in" reason=`
	if !strings.Contains(logs.String(), line) || !strings.Contains(logs.String(), reason) {
		t.Errorf("got the log %q, want %s… with %q", logs, line, reason)
	}
}
