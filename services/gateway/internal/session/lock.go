package session

import (
	"context"
	"log/slog"
	"net/http"
)

// Renewer renews the user's sessions: Store, or a fake in tests.
type Renewer interface {
	// Renew makes the session whose token is token last longer, and reports whether it was live.
	Renew(ctx context.Context, token string) (bool, error)
}

// Require returns middleware that passes a request on only when it carries a live session, which
// it renews, in the browser too. Any other request gets a 401, on which the app shows its sign-in
// screen.
func Require(sessions Renewer, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return &lock{sessions: sessions, logger: logger, next: next}
	}
}

// lock is Require's middleware in front of next.
type lock struct {
	sessions Renewer
	logger   *slog.Logger
	next     http.Handler
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
		l.logger.InfoContext(r.Context(), "no session", "method", r.Method, "path", r.URL.Path)
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)

		return
	}

	http.SetCookie(w, Cookie(token))
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

// tokenOf returns the session token in r's cookie, or "" when it has none.
func tokenOf(r *http.Request) string {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return ""
	}

	return cookie.Value
}
