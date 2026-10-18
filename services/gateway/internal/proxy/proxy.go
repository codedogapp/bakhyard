// Package proxy forwards the gateway's public API to the services behind it.
package proxy

import (
	"encoding/json/v2"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// apiPrefix starts every public API path; the services serve their paths without it.
const apiPrefix = "/api"

// apiError has the shape of a Connect error, so clients keep their error handling when the
// contracts arrive.
type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Register forwards /api/<name> on mux to target as /<name>, whatever the method; target
// answers for its own methods. When target can't be reached, the client gets a 503 with a
// JSON error and logger gets the cause.
func Register(mux *http.ServeMux, name string, target *url.URL, logger *slog.Logger) {
	forward := &httputil.ReverseProxy{
		Rewrite:      func(pr *httputil.ProxyRequest) { pr.SetURL(target) },
		ErrorHandler: unavailable(name, logger),
	}

	mux.Handle(apiPrefix+"/"+name, http.StripPrefix(apiPrefix, forward))
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
