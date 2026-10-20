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

// apiError is the JSON body of the gateway's own errors (plan decision 41).
type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Register forwards /api/<name> on router to target as /<name>, for every standard method;
// target answers for its own methods. When target can't be reached, the client gets a 503
// with a JSON error and logger gets the cause.
func Register(router chi.Router, name string, target *url.URL, logger *slog.Logger) {
	router.Handle(apiPrefix+"/"+name, http.StripPrefix(apiPrefix, forwarder(name, target, logger)))
}

// RegisterFallback forwards every path outside /api on router to target unchanged, as Register
// does. The paths under /api that no service claims keep their 404, so a mistyped API path
// never gets the app's page.
func RegisterFallback(router chi.Router, name string, target *url.URL, logger *slog.Logger) {
	router.Handle("/*", forwarder(name, target, logger))
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
