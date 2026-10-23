package session

import (
	"context"
	"log/slog"
	"net/http"
)

// Ender ends the user's sessions: Store, or a fake in tests.
type Ender interface {
	// End ends the session whose token is token, if there is one.
	End(ctx context.Context, token string) error
}

// SignOut returns the handler that signs the user out on this device: it ends the session in the
// request's cookie, makes the browser drop the cookie and sends it to the app, which then shows
// its sign-in screen. The sessions of other devices stay.
func SignOut(sessions Ender, logger *slog.Logger) http.Handler {
	return &signOut{sessions: sessions, logger: logger}
}

// signOut is SignOut's handler.
type signOut struct {
	sessions Ender
	logger   *slog.Logger
}

func (s *signOut) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := s.end(r.Context(), tokenOf(r))
	if err != nil {
		// The cookie stays, so that signing out again can end the session.
		s.logger.ErrorContext(r.Context(), "failed a sign-out", "err", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)

		return
	}

	http.SetCookie(w, expiredCookie())
	s.logger.InfoContext(r.Context(), "signed out")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// end ends the session whose token is token. Without a token there's no session, and nothing to
// ask the store.
func (s *signOut) end(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}

	return s.sessions.End(ctx, token)
}
