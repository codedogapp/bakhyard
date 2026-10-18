package api_test

import (
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/codedogapp/bakhyard/services/home/internal/api"
)

// item is an item as clients see it. Due is a string, so a missing "due" reads as "".
type item struct {
	ID     string `json:"id"`
	App    string `json:"app"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Source string `json:"source"`
	Due    string `json:"due"`
	Late   bool   `json:"late"`
}

type week struct {
	LoggedHours float64 `json:"loggedHours"`
	TargetHours float64 `json:"targetHours"`
}

type home struct {
	Items []item `json:"items"`
	Week  week   `json:"week"`
}

func TestHomeGet(t *testing.T) {
	rec := serve(t, http.MethodGet)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}

	var got home
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	want := home{
		Items: []item{
			{
				ID: "todos/PAY-1398", App: "todos", Source: "Jira",
				Title: "PAY-1398 Settlement report export", Detail: "Was due yesterday.",
				Due: "2026-10-17T09:00:00Z", Late: true,
			},
			{
				ID: "pulls/PAY-1423", App: "pulls", Source: "Bitbucket",
				Title: "Nudge reviewers on PAY-1423",
				Due:   "2026-10-18T07:00:00Z", Late: true,
			},
			{
				ID: "time/today", App: "time", Source: "Jira",
				Title: "Log today's hours",
				Due:   "2026-10-18T12:00:00Z",
			},
			{
				ID: "time/token", App: "time", Source: "Jira",
				Title: "Jira token expires in 5 days",
				Due:   "2026-10-23T09:00:00Z",
			},
			{
				ID: "planner/home-lab-backups", App: "planner", Source: "Copilot",
				Title: "Finish plan: home lab backups",
			},
			{
				ID: "github/dotfiles-14", App: "github", Source: "GitHub",
				Title: "Review requested: dotfiles#14",
			},
		},
		Week: week{LoggedHours: 31.5, TargetHours: 40},
	}

	if !slices.Equal(got.Items, want.Items) {
		t.Errorf("items = %+v, want %+v", got.Items, want.Items)
	}

	if got.Week != want.Week {
		t.Errorf("week = %+v, want %+v", got.Week, want.Week)
	}
}

// TestHomeEnums checks that the spec allows every app and source the home API returns.
func TestHomeEnums(t *testing.T) {
	var got api.Home
	if err := json.Unmarshal(serve(t, http.MethodGet).Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	for _, item := range got.Items {
		if !item.App.Valid() {
			t.Errorf("item %q: app %q is not in the spec", item.ID, item.App)
		}

		if !item.Source.Valid() {
			t.Errorf("item %q: source %q is not in the spec", item.ID, item.Source)
		}
	}
}

func TestHomeHead(t *testing.T) {
	rec := serve(t, http.MethodHead)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}
}

func TestHomeOtherMethods(t *testing.T) {
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

// now is the clock the tests serve with: 2026-10-18 09:00 UTC.
func now() time.Time {
	return time.Date(2026, time.October, 18, 9, 0, 0, 0, time.UTC)
}

func serve(t *testing.T, method string) *httptest.ResponseRecorder {
	t.Helper()

	router := chi.NewRouter()
	router.Use(middleware.GetHead)
	api.Register(router, now)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, "/home", nil))

	return rec
}
