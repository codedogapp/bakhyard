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

// item holds the response fields under test. Due is a string, so a missing "due" reads as "".
type item struct {
	ID   string `json:"id"`
	Due  string `json:"due"`
	Late bool   `json:"late"`
}

type week struct {
	LoggedHours float64 `json:"loggedHours"`
	TargetHours float64 `json:"targetHours"`
}

type home struct {
	Name  string `json:"name"`
	Items []item `json:"items"`
	Week  week   `json:"week"`
}

// userName is who the tests serve the Home for.
const userName = "Ada"

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
		Name: userName,
		Items: []item{
			{ID: "todos/PAY-1398", Due: "2026-10-17T09:00:00Z", Late: true},
			{ID: "pulls/PAY-1423", Due: "2026-10-18T07:00:00Z", Late: true},
			{ID: "time/today", Due: "2026-10-18T12:00:00Z"},
			{ID: "time/token", Due: "2026-10-23T09:00:00Z"},
			{ID: "planner/home-lab-backups"},
			{ID: "github/dotfiles-14"},
		},
		Week: week{LoggedHours: 31.5, TargetHours: 40},
	}

	if got.Name != want.Name {
		t.Errorf("name = %q, want %q", got.Name, want.Name)
	}

	if !slices.Equal(got.Items, want.Items) {
		t.Errorf("items = %+v, want %+v", got.Items, want.Items)
	}

	if got.Week != want.Week {
		t.Errorf("week = %+v, want %+v", got.Week, want.Week)
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
	api.Register(router, userName, now)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, "/home", nil))

	return rec
}
