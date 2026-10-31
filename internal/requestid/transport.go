package requestid

import "net/http"

// Transport returns a RoundTripper that sends each request on through base with its
// X-Request-Id set by Pass, from the request's context, so that the service it asks takes the ID
// of the request whose handling asks. It sends a copy, since a RoundTripper mustn't change the
// request it's given.
func Transport(base http.RoundTripper) http.RoundTripper {
	return transport{base: base}
}

// transport is the RoundTripper that Transport returns.
type transport struct {
	base http.RoundTripper
}

// RoundTrip sends a copy of req, with its request's ID, through the base RoundTripper.
func (t transport) RoundTrip(req *http.Request) (*http.Response, error) {
	sent := req.Clone(req.Context())
	Pass(sent.Context(), sent.Header)

	return t.base.RoundTrip(sent)
}
