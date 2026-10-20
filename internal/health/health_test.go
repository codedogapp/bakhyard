package health_test

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/codedogapp/bakhyard/internal/health"
)

func TestHealthzGet(t *testing.T) {
	rec := serve(t, http.MethodGet)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}

	if got := rec.Body.String(); got != `{"status":"ok"}` {
		t.Errorf("body = %q, want %q", got, `{"status":"ok"}`)
	}
}

func TestHealthzHead(t *testing.T) {
	rec := serve(t, http.MethodHead)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}
}

func TestHealthzOtherMethods(t *testing.T) {
	methods := []string{
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodOptions,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			rec := serve(t, method)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
			}

			if got := rec.Header().Values("Allow"); !slices.Equal(got, []string{http.MethodGet}) {
				t.Errorf("Allow = %q, want %q", got, []string{http.MethodGet})
			}
		})
	}
}

func serve(t *testing.T, method string) *httptest.ResponseRecorder {
	t.Helper()

	router := chi.NewRouter()
	router.Use(middleware.GetHead)
	health.Register(router)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, "/healthz", nil))

	return rec
}
