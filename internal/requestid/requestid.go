// Package requestid gives each request an ID, which its answer carries and the logs of its
// handling name, so that one request's lines can be found across the services (docs/plan.md
// decisions 500 and 507).
package requestid

import (
	"context"
	"net/http"
	"uuid"
)

// Header carries a request's ID to the service that handles it, and back in its answer.
const Header = "X-Request-Id"

// key is the context key of a request's ID.
type key struct{}

// Middleware gives each request that next handles an ID: its X-Request-Id when that's a UUID,
// in the UUID's canonical form, or else a new UUIDv7. The answer's headers get the ID before
// next runs, and FromContext finds it in the request's context.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := fromHeader(r.Header)
		w.Header().Set(Header, id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), key{}, id)))
	})
}

// FromContext returns the ID of the request that ctx belongs to, if it has one.
func FromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(key{}).(string)

	return id, ok
}

// Pass sets header's X-Request-Id to the ID of the request that ctx belongs to, for a request
// that its handling sends on, or deletes it when ctx has none, so that a client's value never
// passes on unchecked.
func Pass(ctx context.Context, header http.Header) {
	id, ok := FromContext(ctx)
	if ok {
		header.Set(Header, id)

		return
	}

	header.Del(Header)
}

func fromHeader(header http.Header) string {
	id, err := uuid.Parse(header.Get(Header))
	if err != nil {
		return uuid.NewV7().String()
	}

	return id.String()
}
