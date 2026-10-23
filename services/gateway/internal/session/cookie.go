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
	return cookie(token, int(lifetime/time.Second))
}

// expiredCookie returns the cookie that makes the browser drop the session's cookie. It has
// Cookie's attributes: the __Host- prefix makes the browser ignore it without Secure and Path=/.
func expiredCookie() *http.Cookie {
	return cookie("", -1)
}

// cookie returns the session's cookie with value, which the browser keeps for maxAge seconds,
// or drops at once when maxAge is negative.
func cookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     cookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}
