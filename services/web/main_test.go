package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Built or not, the embedded app never answers /healthz in the health check's place.
func TestWebAnswersItsHealthCheck(t *testing.T) {
	router, err := routes(func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(
		rec,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/healthz", nil),
	)

	if rec.Code != http.StatusOK || rec.Body.String() != `{"status":"ok"}` {
		t.Errorf("got %d %q, want 200 and the health check's body", rec.Code, rec.Body.String())
	}
}
