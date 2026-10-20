package supervisor_test

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// bufferedChecks is more than any test counts, so a health server never waits for a test.
const bufferedChecks = 100

// healthServer answers health checks in a service's place: 503 Service Unavailable until it is
// healthy, then 200 OK.
type healthServer struct {
	listen  string
	healthy atomic.Bool
	checked chan struct{} // gets a value for each check, once its answer is decided
}

func newHealthServer(t *testing.T, healthy bool) *healthServer {
	t.Helper()

	health := &healthServer{checked: make(chan struct{}, bufferedChecks)}
	health.healthy.Store(healthy)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if !health.healthy.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		select {
		case health.checked <- struct{}{}:
		default:
		}
	}))
	t.Cleanup(server.Close)

	health.listen = "tcp://" + server.Listener.Addr().String()

	return health
}
