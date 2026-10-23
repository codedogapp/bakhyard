package session

import "net/http"

// Hide is middleware that passes each request on without the session cookie, so no service behind
// the gateway sees the token. The other cookies stay.
func Hide(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, withoutCookie(r))
	})
}

// withoutCookie returns a copy of r without the session cookie; r itself is left as it is.
func withoutCookie(r *http.Request) *http.Request {
	out := r.Clone(r.Context())
	out.Header.Del("Cookie")

	for _, cookie := range r.Cookies() {
		if cookie.Name != cookieName {
			out.AddCookie(cookie)
		}
	}

	return out
}
