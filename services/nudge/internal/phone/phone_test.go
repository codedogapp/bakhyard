package phone_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/codedogapp/bakhyard/internal/database"
	"github.com/codedogapp/bakhyard/internal/databasetest"
	"github.com/codedogapp/bakhyard/services/nudge/internal/change"
	"github.com/codedogapp/bakhyard/services/nudge/internal/migrations"
	"github.com/codedogapp/bakhyard/services/nudge/internal/phone"
	"github.com/codedogapp/bakhyard/services/nudge/internal/question"
	"github.com/codedogapp/bakhyard/services/nudge/internal/session"
)

const id1 = "11111111-1111-4111-8111-111111111111"

type page struct {
	server    *httptest.Server
	questions *question.Store
	sessions  *session.Registry
	pool      *pgxpool.Pool
	logs      *bytes.Buffer
	stop      context.CancelFunc
}

func newPage(t *testing.T) *page {
	t.Helper()

	schema := databasetest.NewSchema(t)

	pool, err := database.Open(t.Context(), schema.URL, schema.Password, migrations.FS())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(pool.Close)

	hub := &change.Hub{}
	stopping, stop := context.WithCancel(t.Context())
	site := &page{
		questions: question.NewStore(pool, hub),
		sessions:  session.NewRegistry(hub),
		pool:      pool,
		logs:      &bytes.Buffer{},
		stop:      stop,
	}

	router := chi.NewRouter()
	// As the service does, answer HEAD with the GET handler.
	router.Use(middleware.GetHead)
	phone.Door{
		Questions: site.questions,
		Sessions:  site.sessions,
		Changes:   hub,
		Logger:    slog.New(slog.NewTextHandler(site.logs, nil)),
		Heartbeat: time.Millisecond,
	}.Register(stopping, router)

	site.server = httptest.NewServer(router)
	t.Cleanup(site.server.Close)

	return site
}

func (site *page) do(t *testing.T, method, path, body string) (int, string) {
	t.Helper()

	req, err := http.NewRequestWithContext(
		t.Context(),
		method,
		site.server.URL+path,
		strings.NewReader(body),
	)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := site.server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)

	return resp.StatusCode, string(raw)
}

func (site *page) askMerge(t *testing.T, options ...string) {
	t.Helper()

	_, err := site.questions.Ask(t.Context(), question.Question{
		ID: id1, Project: "atlas", Text: "Merge?", Options: options,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestStateListsPendingQuestionsAndRunningSessions(t *testing.T) {
	site := newPage(t)

	status, body := site.do(t, http.MethodGet, "/nudge/state", "")
	if status != http.StatusOK || body != `{"questions":[],"sessions":[]}` {
		t.Errorf("empty: %d %s", status, body)
	}

	site.askMerge(t, "Yes", "No")
	started, _ := site.sessions.Start("atlas")

	_, body = site.do(t, http.MethodGet, "/nudge/state", "")
	for _, want := range []string{
		`"id":"` + id1 + `"`, `"text":"Merge?"`, `"options":["Yes","No"]`,
		`"id":"` + started.ID + `"`, `"project":"atlas"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the state lacks %s: %s", want, body)
		}
	}
}

func TestStateFailsWithoutTheDatabase(t *testing.T) {
	site := newPage(t)
	site.pool.Close()

	status, _ := site.do(t, http.MethodGet, "/nudge/state", "")
	if status != http.StatusInternalServerError ||
		!strings.Contains(site.logs.String(), "list the questions") {
		t.Errorf("got %d, log %s", status, site.logs)
	}
}

func TestAnswerReachesTheAgent(t *testing.T) {
	site := newPage(t)
	site.askMerge(t, "Yes", "No")

	status, body := site.do(
		t,
		http.MethodPost,
		"/nudge/questions/"+id1+"/answer",
		`{"option":"Yes","text":"go"}`,
	)
	if status != http.StatusNoContent {
		t.Fatalf("answer: %d %s", status, body)
	}

	got, err := site.questions.Wait(t.Context(), id1)
	if err != nil || got != (question.Answer{Option: "Yes", Text: "go"}) {
		t.Errorf("the agent got %+v, %v", got, err)
	}
}

func TestAnAnswerThatCannotBeTakenIsRefused(t *testing.T) {
	site := newPage(t)
	site.askMerge(t, "Yes")

	tests := []struct {
		name, path, body string
		want             int
	}{
		{
			"an unknown question",
			"/nudge/questions/" + "22222222-2222-4222-8222-222222222222/answer",
			`{"text":"x"}`,
			http.StatusNotFound,
		},
		{
			"an id that is no UUID",
			"/nudge/questions/nope/answer",
			`{"text":"x"}`,
			http.StatusNotFound,
		},
		{"an empty answer", "/nudge/questions/" + id1 + "/answer", `{}`, http.StatusBadRequest},
		{
			"an unknown option",
			"/nudge/questions/" + id1 + "/answer",
			`{"option":"Maybe"}`,
			http.StatusBadRequest,
		},
		{"a bad body", "/nudge/questions/" + id1 + "/answer", `nope`, http.StatusBadRequest},
	}

	for _, test := range tests {
		if status, body := site.do(t, http.MethodPost, test.path, test.body); status != test.want {
			t.Errorf("%s: %d %s, want %d", test.name, status, body, test.want)
		}
	}
}

func TestAnAnswerFailsWithoutTheDatabase(t *testing.T) {
	site := newPage(t)
	site.askMerge(t)
	site.pool.Close()

	status, _ := site.do(t, http.MethodPost, "/nudge/questions/"+id1+"/answer", `{"text":"x"}`)
	if status != http.StatusInternalServerError ||
		!strings.Contains(site.logs.String(), "answer the question") {
		t.Errorf("got %d, log %s", status, site.logs)
	}
}
