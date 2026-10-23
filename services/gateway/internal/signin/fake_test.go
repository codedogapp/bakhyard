package signin_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json/v2"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/coreos/go-oidc/v3/oidc/oidctest"
	"golang.org/x/oauth2"
)

// The fake's key ID and access token.
const (
	keyID       = "test-key"
	accessToken = "the-access-token"
)

// fakeGoogle stands in for Google's token and key endpoints. It trades each code that the user
// approved for an ID token, once.
type fakeGoogle struct {
	url string
	key *ecdsa.PrivateKey

	mu     sync.Mutex
	grants map[string]grant
	// secrets holds what the flow must never log: the codes, the PKCE verifiers and the tokens.
	secrets []string
}

// grant is a login that the user approved at Google.
type grant struct {
	challenge string
	// claims are the ID token's claims; nil leaves the ID token out of the answer.
	claims map[string]any
	signer *ecdsa.PrivateKey
}

func newFakeGoogle(t *testing.T) *fakeGoogle {
	t.Helper()

	google := &fakeGoogle{key: newKey(t), grants: make(map[string]grant), secrets: []string{secret}}

	mux := http.NewServeMux()
	mux.Handle("GET /keys", &oidctest.Server{PublicKeys: []oidctest.PublicKey{
		{PublicKey: google.key.Public(), KeyID: keyID, Algorithm: oidc.ES256},
	}})
	mux.HandleFunc("POST /token", google.token)

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	google.url = server.URL

	return google
}

func newKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	return key
}

// provider returns the fake's endpoints.
func (g *fakeGoogle) provider() oidc.ProviderConfig {
	return oidc.ProviderConfig{
		IssuerURL:  g.url,
		AuthURL:    g.url + "/auth",
		TokenURL:   g.url + "/token",
		JWKSURL:    g.url + "/keys",
		Algorithms: []string{oidc.ES256},
	}
}

// approve returns a new code for the login that sent the browser to redirect. The code gets a
// valid ID token for the user, as change leaves it.
func (g *fakeGoogle) approve(redirect *url.URL, change func(*grant)) string {
	query := redirect.Query()
	approved := grant{
		challenge: query.Get("code_challenge"),
		claims: map[string]any{
			"iss": g.url, "aud": clientID, "sub": subject,
			"iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
			"nonce": query.Get("nonce"), "email": email, "email_verified": true,
		},
		signer: g.key,
	}
	change(&approved)

	code := "4/" + rand.Text()

	g.mu.Lock()
	defer g.mu.Unlock()

	g.grants[code] = approved
	g.secrets = append(g.secrets, code)

	return code
}

// token is the fake's token endpoint. It answers with the tokens for an approved code if the
// client, the redirect URL and the PKCE verifier match, as Google does.
func (g *fakeGoogle) token(w http.ResponseWriter, r *http.Request) {
	code, verifier := r.FormValue("code"), r.FormValue("code_verifier")

	g.mu.Lock()
	defer g.mu.Unlock()

	approved, ok := g.grants[code]
	delete(g.grants, code)
	g.secrets = append(g.secrets, verifier)

	user, password, _ := r.BasicAuth()
	if !ok || user != clientID || password != secret ||
		r.FormValue("grant_type") != "authorization_code" ||
		r.FormValue("redirect_uri") != "https://"+host+"/auth/google/callback" ||
		oauth2.S256ChallengeFromVerifier(verifier) != approved.challenge {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))

		return
	}

	answer := map[string]any{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   3600,
	}
	g.secrets = append(g.secrets, accessToken)

	if approved.claims != nil {
		claims, err := json.Marshal(approved.claims)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		idToken := oidctest.SignIDToken(approved.signer, keyID, oidc.ES256, string(claims))
		answer["id_token"] = idToken
		g.secrets = append(g.secrets, idToken)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.MarshalWrite(w, answer)
}

// noLeaks fails t if logs have the client secret, a code, a PKCE verifier or a token.
func (g *fakeGoogle) noLeaks(t *testing.T, logs string) {
	t.Helper()

	g.mu.Lock()
	defer g.mu.Unlock()

	for _, value := range g.secrets {
		if value != "" && strings.Contains(logs, value) {
			t.Errorf("the log has %q:\n%s", value, logs)
		}
	}
}

// fakeSessions stands in for the session store. It starts each session with a new token, unless
// err is set.
type fakeSessions struct {
	mu     sync.Mutex
	tokens []string
	err    error
}

func (s *fakeSessions) Start(context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return "", s.err
	}

	token := rand.Text()
	s.tokens = append(s.tokens, token)

	return token, nil
}

// started returns the tokens of the sessions started so far.
func (s *fakeSessions) started() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return slices.Clone(s.tokens)
}

// fakePins stands in for the pin store. It pins the first account of each email, unless err is
// set.
type fakePins struct {
	mu       sync.Mutex
	accounts map[string]string
	err      error
}

func (p *fakePins) Pin(_ context.Context, email, sub string) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.err != nil {
		return false, p.err
	}

	pinned, ok := p.accounts[email]
	if !ok {
		p.accounts[email], pinned = sub, sub
	}

	return pinned == sub, nil
}

// pinned returns the accounts pinned so far, by email.
func (p *fakePins) pinned() map[string]string {
	p.mu.Lock()
	defer p.mu.Unlock()

	return maps.Clone(p.accounts)
}
