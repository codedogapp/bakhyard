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

// apiError is the body of the gateway's own errors: the Error schema of the API specs in api/.
type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Register forwards /api/<name> on router to target as /<name>, for every standard method;
// target answers for its own methods. When target can't be reached, the client gets a 503
// with a JSON error and logger gets the cause.
func Register(router chi.Router, name string, target *url.URL, logger *slog.Logger) {
	forward := &httputil.ReverseProxy{
		Rewrite:      func(pr *httputil.ProxyRequest) { pr.SetURL(target) },
		ErrorHandler: unavailable(name, logger),
	}

	router.Handle(apiPrefix+"/"+name, http.StripPrefix(apiPrefix, forward))
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
