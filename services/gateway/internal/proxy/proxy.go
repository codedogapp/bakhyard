// Package proxy forwards the gateway's public API to the services behind it.
package proxy

import (
	"encoding/json/v2"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/go-chi/chi/v5"
)

// apiPrefix starts every public API path; the services serve their paths without it.
const apiPrefix = "/api"

// agentPrefix starts the paths that agents use; the service serves them with it.
const agentPrefix = "/agent"

// apiError is the JSON body of the gateway's own errors (plan decision 41).
type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Register forwards /api/<name> and every path under it on router to target, as /<name>…, for
// every standard method; target answers for its own paths and methods (decisions 41 and 92). When
// target can't be reached, the client gets a 503 with a JSON error and logger gets the cause.
func Register(router chi.Router, name string, target *url.URL, logger *slog.Logger) {
	forward := http.StripPrefix(apiPrefix, forwarder(name, target, logger))
	router.Handle(apiPrefix+"/"+name, forward)
	router.Handle(apiPrefix+"/"+name+"/*", forward)
}

// RegisterAgent forwards /agent/<path> on router to target unchanged, for every standard method:
// the paths that agents on other machines use to ask and report. target answers for its own paths
// and methods. Whoever calls it decides who gets in.
func RegisterAgent(router chi.Router, name string, target *url.URL, logger *slog.Logger) {
	router.Handle(agentPrefix+"/*", forwarder(name, target, logger))
}

// RegisterFallback forwards every path that router has no route for to target unchanged, as
// Register does. It is router's NotFound handler rather than a /* route, which chi would also pick
// for a method that one of router's own routes doesn't serve; so that route keeps its 405. With
// RegisterUnclaimed, no path under /api gets there.
func RegisterFallback(router chi.Router, name string, target *url.URL, logger *slog.Logger) {
	router.NotFound(forwarder(name, target, logger).ServeHTTP)
}

// RegisterUnclaimed answers 404 on router for /api and the paths under it that no service
// claims, so a mistyped API path never gets the app's page.
func RegisterUnclaimed(router chi.Router) {
	router.Handle(apiPrefix, http.NotFoundHandler())
	router.Handle(apiPrefix+"/*", http.NotFoundHandler())
}

func forwarder(name string, target *url.URL, logger *slog.Logger) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Rewrite:      func(pr *httputil.ProxyRequest) { pr.SetURL(target) },
		ErrorHandler: unavailable(name, logger),
	}
}

func unavailable(name string, logger *slog.Logger) func(http.ResponseWriter, *http.Request, error) {
	body := apiError{Code: "unavailable", Message: name + " is unavailable"}

	return func(w http.ResponseWriter, r *http.Request, err error) {
		logger.WarnContext(r.Context(), "upstream unavailable", "upstream", name, "err", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		// The body always encodes, so an error means the client has gone.
		_ = json.MarshalWrite(w, body)
	}
}
