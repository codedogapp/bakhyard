package session

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
)

// apiPrefix starts the paths of the gateway's public API (package proxy). The app fetches only
// there, so any other GET or HEAD loads a page.
const apiPrefix = "/api"

// Renewer renews the user's sessions: Store, or a fake in tests.
type Renewer interface {
	// Renew makes the session whose token is token last longer, and reports whether it was live.
	Renew(ctx context.Context, token string) (bool, error)
}

// Require returns middleware that passes a request on only when it carries a live session, which
// it renews, in the browser too. A page load without one goes to loginPath to sign in; any other
// request gets a 401, on which the app reloads. The request passed on carries no session cookie,
// so no service behind the gateway sees the token.
func Require(
	sessions Renewer, loginPath string, logger *slog.Logger,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return &lock{sessions: sessions, loginPath: loginPath, logger: logger, next: next}
	}
}

// lock is Require's middleware in front of next.
type lock struct {
	sessions  Renewer
	loginPath string
	logger    *slog.Logger
	next      http.Handler
}

func (l *lock) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := tokenOf(r)

	live, err := l.renew(r.Context(), token)
	if err != nil {
		l.logger.ErrorContext(r.Context(), "failed to check a session",
			"path", r.URL.Path, "err", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)

		return
	}

	if !live {
		l.refuse(w, r)

		return
	}

	http.SetCookie(w, Cookie(token))
	dropCookie(r)
	l.next.ServeHTTP(w, r)
}

// renew renews the session whose token is token, and reports whether it was live. Without a
// token there's no session, and nothing to ask the store.
func (l *lock) renew(ctx context.Context, token string) (bool, error) {
	if token == "" {
		return false, nil
	}

	return l.sessions.Renew(ctx, token)
}

// refuse sends a page load to sign in and answers any other request with a 401.
func (l *lock) refuse(w http.ResponseWriter, r *http.Request) {
	l.logger.InfoContext(r.Context(), "no session", "method", r.Method, "path", r.URL.Path)

	if pageLoad(r) {
		http.Redirect(w, r, l.loginPath, http.StatusFound)

		return
	}

	http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
}

// pageLoad reports whether r loads a page: a GET or HEAD outside the API.
func pageLoad(r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}

	return r.URL.Path != apiPrefix && !strings.HasPrefix(r.URL.Path, apiPrefix+"/")
}

// tokenOf returns the session token in r's cookie, or "" when it has none.
func tokenOf(r *http.Request) string {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return ""
	}

	return cookie.Value
}

// dropCookie removes the session cookie from r and keeps the others.
func dropCookie(r *http.Request) {
	cookies := r.Cookies()
	r.Header.Del("Cookie")

	for _, cookie := range cookies {
		if cookie.Name != cookieName {
			r.AddCookie(cookie)
		}
	}
}
