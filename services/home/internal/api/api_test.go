package api_test

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/codedogapp/bakhyard/services/home/internal/api"
	"github.com/codedogapp/bakhyard/services/home/internal/screen"
)

// item holds the response fields under test. Due is a string, so a missing "due" reads as "".
type item struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Due    string `json:"due"`
	Late   bool   `json:"late"`
}

type home struct {
	Name  string `json:"name"`
	Items []item `json:"items"`
}

// userName is who the tests serve the Home for.
const userName = "Ada"

// at is the tests' time: 2026-10-18 09:00 UTC.
func at() time.Time {
	return time.Date(2026, time.October, 18, 9, 0, 0, 0, time.UTC)
}

func TestHomeGet(t *testing.T) {
	rec := serve(t, http.MethodGet, listing(
		screen.Item{ID: "pulls/web#1", Source: screen.SourceBitbucket, Due: at().Add(time.Hour)},
		screen.Item{ID: "todos/7", Source: screen.SourceBakhyard},
		screen.Item{ID: "todos/WEB-9", Source: screen.SourceJira, Due: at().Add(-time.Hour)},
	))

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
			{ID: "todos/WEB-9", Source: "Jira", Due: "2026-10-18T08:00:00Z", Late: true},
			{ID: "pulls/web#1", Source: "Bitbucket", Due: "2026-10-18T10:00:00Z"},
			{ID: "todos/7", Source: "Bakhyard"},
		},
	}

	if got.Name != want.Name || !slices.Equal(got.Items, want.Items) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestHomeWithNothingToDo(t *testing.T) {
	rec := serve(t, http.MethodGet, listing())

	var got map[string]jsontext.Value
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if items := string(got["items"]); items != "[]" {
		t.Errorf("items = %s, want []", items)
	}

	if week, found := got["week"]; found {
		t.Errorf("week = %s, want none", week)
	}
}

func TestHomeMarksLateOnceTheItemsAreIn(t *testing.T) {
	clock := at()
	slow := func(context.Context) []screen.Item {
		clock = clock.Add(time.Second)

		return []screen.Item{{ID: "todos/1", Due: at().Add(time.Millisecond)}}
	}

	rec := serveWith(t, http.MethodGet, slow, func() time.Time { return clock })

	var got home
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if len(got.Items) != 1 || !got.Items[0].Late {
		t.Errorf("items = %+v, want todos/1 late", got.Items)
	}
}

func TestHomeAsksWithTheRequestsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	var asked error

	items := func(ctx context.Context) []screen.Item {
		asked = ctx.Err()

		return nil
	}

	router := chi.NewRouter()
	api.Register(router, userName, at, items)
	router.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequestWithContext(ctx, http.MethodGet, "/home", nil))

	if !errors.Is(asked, context.Canceled) {
		t.Errorf("the items were asked for with a context that ended with %v, want %v",
			asked, context.Canceled)
	}
}

func TestHomeHead(t *testing.T) {
	rec := serve(t, http.MethodHead, listing())

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
			rec := serve(t, method, listing())

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
			}

			if got := rec.Header().Values("Allow"); !slices.Equal(got, []string{http.MethodGet}) {
				t.Errorf("Allow = %q, want %q", got, []string{http.MethodGet})
			}
		})
	}
}

// listing returns an items function that lists items.
func listing(items ...screen.Item) func(context.Context) []screen.Item {
	return func(context.Context) []screen.Item { return items }
}

// serve answers a request to /home with method, with the items that items lists, at the tests'
// time.
func serve(
	t *testing.T, method string, items func(context.Context) []screen.Item,
) *httptest.ResponseRecorder {
	t.Helper()

	return serveWith(t, method, items, at)
}

// serveWith is serve with the clock now.
func serveWith(
	t *testing.T, method string, items func(context.Context) []screen.Item, now func() time.Time,
) *httptest.ResponseRecorder {
	t.Helper()

	router := chi.NewRouter()
	router.Use(middleware.GetHead)
	api.Register(router, userName, now, items)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, "/home", nil))

	return rec
}
