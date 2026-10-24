package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/services/notes/internal/api"
)

func TestTheListNamesEveryFile(t *testing.T) {
	for _, test := range []struct {
		name  string
		files []string
		want  string
	}{
		{"files", []string{"Inbox.md", "img/photo.png"}, `{"files":["Inbox.md","img/photo.png"]}`},
		{"none", nil, `{"files":[]}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			rec := serve(t, test.files, http.MethodGet)

			if rec.Code != http.StatusOK || rec.Body.String() != test.want ||
				rec.Header().Get("Content-Type") != "application/json" {
				t.Errorf("got %d %q %q, want 200 application/json %q", rec.Code,
					rec.Header().Get("Content-Type"), rec.Body.String(), test.want)
			}
		})
	}
}

func TestTheListIsReadOnly(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		rec := serve(t, []string{"Inbox.md"}, method)

		if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != http.MethodGet {
			t.Errorf("%s /notes: got %d, Allow %q; want 405, Allow GET",
				method, rec.Code, rec.Header().Get("Allow"))
		}
	}
}

// serve sends a method request for /notes to a router that lists files, and returns its answer.
func serve(t *testing.T, files []string, method string) *httptest.ResponseRecorder {
	t.Helper()

	router := chi.NewRouter()
	api.Register(router, files)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, "/notes", nil))

	return rec
}
