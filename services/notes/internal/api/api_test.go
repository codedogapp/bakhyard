package api_test

import (
	"log/slog"
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

func TestTheListLinksToTheVaultsHistory(t *testing.T) {
	const history = "https://gitea.example/me/vault/commits/branch/main"

	for _, test := range []struct {
		name  string
		store api.Store
		want  string
	}{
		{"read-only", nil, `{"files":["Inbox.md"],"history":"` + history + `"}`},
		{
			"editable", storeWith("Inbox.md"),
			`{"files":["Inbox.md"],"editable":true,"history":"` + history + `"}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			router := chi.NewRouter()
			vault := api.Vault{
				Folder: vaultWith(t, "Inbox.md").Name(), Store: test.store, History: history,
			}

			if err := api.Register(router, vault, slog.New(slog.DiscardHandler)); err != nil {
				t.Fatal(err)
			}

			for _, req := range []*http.Request{
				httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/notes", nil),
				httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/notes/sync", nil),
			} {
				rec := serve(router, req)

				if rec.Code != http.StatusOK || rec.Body.String() != test.want {
					t.Errorf("%s %s: got %d %q, want 200 %q", req.Method, req.URL.Path, rec.Code,
						rec.Body.String(), test.want)
				}
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
