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
			router, _ := newRouter(t, vaultWith(t), test.files)
			rec := get(t, router, "/notes", nil)

			if rec.Code != http.StatusOK || rec.Body.String() != test.want ||
				rec.Header().Get("Content-Type") != "application/json" {
				t.Errorf("got %d %q %q, want 200 application/json %q", rec.Code,
					rec.Header().Get("Content-Type"), rec.Body.String(), test.want)
			}
		})
	}
}

func TestTheAPIIsReadOnly(t *testing.T) {
	router, _ := newRouter(t, vaultWith(t, "Inbox.md"), []string{"Inbox.md"})

	for _, target := range []string{"/notes", "/notes/files/Inbox.md"} {
		for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, target, nil))

			if rec.Code != http.StatusMethodNotAllowed ||
				rec.Header().Get("Allow") != http.MethodGet {
				t.Errorf("%s %s: got %d, Allow %q; want 405, Allow GET",
					method, target, rec.Code, rec.Header().Get("Allow"))
			}
		}
	}
}
