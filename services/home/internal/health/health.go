package health

import (
	"io"
	"net/http"
)

const body = `{"status":"ok"}`

// Register serves GET and HEAD /healthz on mux; other methods get 405.
func Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", healthz)
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// A failed write means the client has gone; there is no one left to tell.
	_, _ = io.WriteString(w, body)
}
