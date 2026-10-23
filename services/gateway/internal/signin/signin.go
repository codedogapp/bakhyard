// Package signin signs the user in with Google, the gateway's second lock (docs/plan.md §3 Auth):
// the OpenID Connect authorization code flow with PKCE, which lets in only the user's own
// verified email, with the Google account pinned to it.
package signin

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-chi/chi/v5"
	"golang.org/x/oauth2"

	"github.com/codedogapp/bakhyard/services/gateway/internal/session"
)

const (
	// basePath holds the flow's routes; Google sends the browser back to its callback.
	basePath      = "/auth/google"
	loginRoute    = "/login"
	callbackRoute = "/callback"
	// googleTimeout bounds every call to Google.
	googleTimeout = 10 * time.Second
)

// Settings say which Google client signs the user in, and who the user is.
type Settings struct {
	// Provider holds Google's endpoints: Google(), or a fake's in tests.
	Provider     oidc.ProviderConfig
	ClientID     string
	ClientSecret string
	// Email is the one email allowed in. Its first sign-in pins its Google account to it, and
	// every later sign-in needs that account.
	Email string
	// Host is the gateway's public host, where Google sends the browser back to.
	Host string
}

// SessionStarter starts the user's sessions: session.Store, or a fake in tests.
type SessionStarter interface {
	// Start starts a session and returns its token.
	Start(ctx context.Context) (string, error)
}

// Pinner pins the user's Google account to their email: pin.Store, or a fake in tests.
type Pinner interface {
	// Pin pins the account sub to email unless an account is pinned to it already, and reports
	// whether sub is the account pinned to email.
	Pin(ctx context.Context, email, sub string) (bool, error)
}

// Flow signs the user in with Google.
type Flow struct {
	oauth    oauth2.Config
	verifier *oidc.IDTokenVerifier
	client   *http.Client
	email    string
	logins   *logins
	sessions SessionStarter
	pins     Pinner
	logger   *slog.Logger
}

// New returns the sign-in flow for settings, which checks each sign-in's account against the
// one pinned in pins and starts a session in sessions. It reaches Google only when someone signs
// in.
func New(settings Settings, sessions SessionStarter, pins Pinner, logger *slog.Logger) *Flow {
	client := &http.Client{Timeout: googleTimeout}
	provider := settings.Provider.NewProvider(oidc.ClientContext(context.Background(), client))

	return &Flow{
		oauth: oauth2.Config{
			ClientID:     settings.ClientID,
			ClientSecret: settings.ClientSecret,
			Endpoint:     provider.Endpoint(),
			RedirectURL:  "https://" + settings.Host + basePath + callbackRoute,
			Scopes:       []string{oidc.ScopeOpenID, "email"},
		},
		verifier: provider.Verifier(&oidc.Config{ClientID: settings.ClientID}),
		client:   client,
		email:    settings.Email,
		logins:   newLogins(),
		sessions: sessions,
		pins:     pins,
		logger:   logger,
	}
}

// Register adds the flow's routes to router, under /auth/google. They're a sub-router, so that the
// middleware of router's group also sees the methods they don't serve: chi answers those with a
// 405 that skips a group's middleware.
func (f *Flow) Register(router chi.Router) {
	router.Route(basePath, func(google chi.Router) {
		google.Get(loginRoute, f.login)
		google.Get(callbackRoute, f.callback)
	})
}

// login sends the browser to Google and keeps the login until Google sends it back.
func (f *Flow) login(w http.ResponseWriter, r *http.Request) {
	state, nonce, verifier := rand.Text(), rand.Text(), oauth2.GenerateVerifier()
	f.logins.add(state, pending{nonce: nonce, verifier: verifier})

	target := f.oauth.AuthCodeURL(state,
		oidc.Nonce(nonce),
		oauth2.S256ChallengeOption(verifier),
		oauth2.SetAuthURLParam("prompt", "select_account"),
	)
	http.Redirect(w, r, target, http.StatusFound)
}

// callback checks Google's answer and the account pinned to its email, and starts a session for
// the user only, in a cookie.
func (f *Flow) callback(w http.ResponseWriter, r *http.Request) {
	user, err := f.check(r)
	if err != nil {
		f.refuse(w, r, err)

		return
	}

	pinned, err := f.pins.Pin(r.Context(), user.email, user.sub)
	if err != nil {
		f.fail(w, r, user.email, err)

		return
	}

	if !pinned {
		f.refuse(w, r, fmt.Errorf("another account with the email %s", user.email))

		return
	}

	token, err := f.sessions.Start(r.Context())
	if err != nil {
		f.fail(w, r, user.email, err)

		return
	}

	http.SetCookie(w, session.Cookie(token))
	f.logger.InfoContext(r.Context(), "signed in", "email", user.email)
	http.Redirect(w, r, "/", http.StatusFound)
}

// refuse answers a sign-in that isn't the user's with a 403, and logs why.
func (f *Flow) refuse(w http.ResponseWriter, r *http.Request, reason error) {
	f.logger.WarnContext(r.Context(), "refused a sign-in", "reason", reason)
	http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
}

// fail answers the user's sign-in with a 500 when it can't finish, and logs the error.
func (f *Flow) fail(w http.ResponseWriter, r *http.Request, email string, err error) {
	f.logger.ErrorContext(r.Context(), "failed a sign-in", "email", email, "err", err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}
