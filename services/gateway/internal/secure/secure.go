// Package secure adds the security headers of docs/plan.md §4.4 to every response the gateway
// sends, the services' included.
package secure

import "net/http"

// policy is the app's Content-Security-Policy. form-action doesn't fall back to default-src, so
// the policy names it.
const policy = "default-src 'self'; style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data: https:; font-src 'self' data:; connect-src 'self'; " +
	"form-action 'self'; frame-ancestors 'none'; base-uri 'none'"

// Headers is middleware that adds the security headers to each response that next sends. A
// response that sets one of them itself keeps its own value, such as the vault files' stricter
// policy (decision 92).
func Headers(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&writer{ResponseWriter: w}, r)
		// For a handler that wrote nothing: the server sends its 200 after this.
		add(w.Header())
	})
}

// writer adds the headers just before they go out: on each WriteHeader, since a 1xx doesn't send
// the final ones and ReverseProxy clears the map after it, and on a Write or a flush, which send
// a 200's headers when none went out yet. Adding them once they're out does nothing.
type writer struct {
	http.ResponseWriter
}

func (w *writer) WriteHeader(code int) {
	add(w.Header())
	w.ResponseWriter.WriteHeader(code)
}

func (w *writer) Write(body []byte) (int, error) {
	add(w.Header())

	return w.ResponseWriter.Write(body)
}

// FlushError is the flush that http.ResponseController calls, as ReverseProxy does.
func (w *writer) FlushError() error {
	add(w.Header())

	return http.NewResponseController(w.ResponseWriter).Flush()
}

// Unwrap lets http.ResponseController reach the server's writer, to flush or take over the
// connection.
func (w *writer) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// add sets each security header that header lacks.
func add(header http.Header) {
	addMissing(header, "Content-Security-Policy", policy)
	addMissing(header, "X-Content-Type-Options", "nosniff")
	// Other sites never learn the app's address, from the links in notes for one.
	addMissing(header, "Referrer-Policy", "same-origin")
}

func addMissing(header http.Header, key, value string) {
	if len(header.Values(key)) == 0 {
		header.Set(key, value)
	}
}
