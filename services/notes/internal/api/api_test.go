package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
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
			router, _ := newRouter(t, vaultWith(t, test.files...).Name())
			rec := get(t, router, "/notes", nil)

			if rec.Code != http.StatusOK || rec.Body.String() != test.want ||
				rec.Header().Get("Content-Type") != "application/json" {
				t.Errorf("got %d %q %q, want 200 application/json %q", rec.Code,
					rec.Header().Get("Content-Type"), rec.Body.String(), test.want)
			}
		})
	}
}

func TestEachPathTakesOnlyItsMethod(t *testing.T) {
	router, _ := newRouter(t, vaultWith(t, "Inbox.md").Name())

	for _, test := range []struct {
		target  string
		allow   string
		methods []string
	}{
		{"/notes", http.MethodGet, []string{http.MethodPost, http.MethodPut, http.MethodDelete}},
		{"/notes/files/Inbox.md", http.MethodGet, []string{
			http.MethodPost, http.MethodPut, http.MethodDelete,
		}},
		{"/notes/sync", http.MethodPost, []string{http.MethodGet, http.MethodPut, http.MethodDelete}},
	} {
		for _, method := range test.methods {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec,
				httptest.NewRequestWithContext(t.Context(), method, test.target, nil))

			if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != test.allow {
				t.Errorf("%s %s: got %d, Allow %q; want 405, Allow %s",
					method, test.target, rec.Code, rec.Header().Get("Allow"), test.allow)
			}
		}
	}
}
