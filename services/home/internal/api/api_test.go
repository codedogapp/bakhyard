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

type source struct {
	App   string `json:"app"`
	State string `json:"state"`
}

type home struct {
	Name    string   `json:"name"`
	Items   []item   `json:"items"`
	Sources []source `json:"sources"`
}

// gatherer is what Register asks for what needs the user.
type gatherer = func(context.Context) ([]screen.Item, []screen.SourceStatus)

// userName is who the tests serve the Home for.
const userName = "Ada"

// at is the tests' time: 2026-10-18 09:00 UTC.
func at() time.Time {
	return time.Date(2026, time.October, 18, 9, 0, 0, 0, time.UTC)
}

func TestHomeGet(t *testing.T) {
	items := []screen.Item{
		{ID: "pulls/web#1", Source: screen.SourceBitbucket, Due: at().Add(time.Hour)},
		{ID: "todos/7", Source: screen.SourceBakhyard},
		{ID: "todos/WEB-9", Source: screen.SourceJira, Due: at().Add(-time.Hour)},
	}
	sources := []screen.SourceStatus{
		{App: "todos", State: screen.StateOK},
		{App: "pulls", State: screen.StateUnavailable},
	}
	rec := serve(t, http.MethodGet, func(context.Context) ([]screen.Item, []screen.SourceStatus) {
		return items, sources
	})

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
		Sources: []source{{App: "todos", State: "ok"}, {App: "pulls", State: "unavailable"}},
	}

	if got.Name != want.Name || !slices.Equal(got.Items, want.Items) ||
		!slices.Equal(got.Sources, want.Sources) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestHomeWithNothingToDo(t *testing.T) {
	rec := serve(t, http.MethodGet, nothing)

	var got map[string]jsontext.Value
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if items := string(got["items"]); items != "[]" {
		t.Errorf("items = %s, want []", items)
	}

	if sources := string(got["sources"]); sources != "[]" {
		t.Errorf("sources = %s, want []", sources)
	}

	if week, found := got["week"]; found {
		t.Errorf("week = %s, want none", week)
	}

	if state, found := got["weekState"]; found {
		t.Errorf("weekState = %s, want none: home has no time service", state)
	}
}

func TestHomeMarksLateOnceTheItemsAreIn(t *testing.T) {
	clock := at()
	slow := func(context.Context) ([]screen.Item, []screen.SourceStatus) {
		clock = clock.Add(time.Second)

		return []screen.Item{{ID: "todos/1", Due: at().Add(time.Millisecond)}}, nil
	}

	rec := serveWith(t, http.MethodGet, api.Peers{Items: slow, Week: noTime},
		func() time.Time { return clock })

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

	var items, week error

	peers := api.Peers{
		Items: func(ctx context.Context) ([]screen.Item, []screen.SourceStatus) {
			items = ctx.Err()

			return nil, nil
		},
		Week: func(ctx context.Context) (*screen.Week, screen.State) {
			week = ctx.Err()

			return nil, ""
		},
	}

	router := chi.NewRouter()
	api.Register(router, userName, at, peers)
	router.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequestWithContext(ctx, http.MethodGet, "/home", nil))

	if !errors.Is(items, context.Canceled) || !errors.Is(week, context.Canceled) {
		t.Errorf("the items and the week were asked for with contexts that ended with %v and "+
			"%v, want %v", items, week, context.Canceled)
	}
}

func TestHomeHead(t *testing.T) {
	rec := serve(t, http.MethodHead, nothing)

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
			rec := serve(t, method, nothing)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
			}

			if got := rec.Header().Values("Allow"); !slices.Equal(got, []string{http.MethodGet}) {
				t.Errorf("Allow = %q, want %q", got, []string{http.MethodGet})
			}
		})
	}
}

// nothing lists no items and no sources.
func nothing(context.Context) ([]screen.Item, []screen.SourceStatus) {
	return nil, nil
}

// noTime answers the week as home does without a time service: no week and no state.
func noTime(context.Context) (*screen.Week, screen.State) {
	return nil, ""
}

// serve answers a request to /home with method, with what gather gathers, at the tests' time,
// as home without a time service.
func serve(t *testing.T, method string, gather gatherer) *httptest.ResponseRecorder {
	t.Helper()

	return serveWith(t, method, api.Peers{Items: gather, Week: noTime}, at)
}

// serveWith answers a request to /home with method, with what peers answer, at the clock now.
func serveWith(
	t *testing.T, method string, peers api.Peers, now func() time.Time,
) *httptest.ResponseRecorder {
	t.Helper()

	router := chi.NewRouter()
	router.Use(middleware.GetHead)
	api.Register(router, userName, now, peers)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, "/home", nil))

	return rec
}
