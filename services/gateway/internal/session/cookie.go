package session

import (
	"net/http"
	"time"
)

// cookieName names the cookie that carries a session's token. Its __Host- prefix makes the
// browser take it only from a secure page, for the whole host and no other.
const cookieName = "__Host-session"

// Cookie returns the cookie that gives the browser a session's token for as long as the session
// lasts. Scripts can't read it, and other sites' pages can't send it, except by a link.
func Cookie(token string) *http.Cookie {
	return &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(lifetime / time.Second),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}
