package api_test

import (
	"bufio"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/codedogapp/bakhyard/services/planner/internal/api"
	"github.com/codedogapp/bakhyard/services/planner/internal/copilot"
)

// follower is a page that follows the events of the plan "plan-1".
type follower struct {
	lines *bufio.Scanner
	// leave ends the page's request, and stop the service.
	leave, stop context.CancelFunc
	// served closes once the handler has returned.
	served chan struct{}
}

// follow serves the API that a Door serves from planner, with a heartbeat every heartbeat, and
// follows the plan "plan-1" as a page does.
func follow(t *testing.T, planner api.Planner, heartbeat time.Duration) follower {
	t.Helper()

	stopping, stop := context.WithCancel(t.Context())
	router := chi.NewRouter()
	door := api.Door{Planner: planner, Logger: slog.New(slog.DiscardHandler), Heartbeat: heartbeat}
	door.Register(stopping, router)

	served := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(served)

		router.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)

	page, leave := context.WithCancel(t.Context())
	req, _ := http.NewRequestWithContext(page, http.MethodGet,
		server.URL+"/planner/plans/plan-1/events", nil)

	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = resp.Body.Close() })

	sent := resp.Header.Get("Content-Type")
	if resp.StatusCode != http.StatusOK || sent != "text/event-stream" {
		t.Fatalf("got %d (%s), want 200 as an event stream", resp.StatusCode, sent)
	}

	return follower{lines: bufio.NewScanner(resp.Body), leave: leave, stop: stop, served: served}
}

// next returns the next event that the page gets, as its lines, skipping the comments.
func (f follower) next(t *testing.T) string {
	t.Helper()

	var event []string

	for f.lines.Scan() {
		line := f.lines.Text()
		if line == "" && len(event) > 0 {
			return strings.Join(event, "\n")
		}

		if line != "" && !strings.HasPrefix(line, ":") {
			event = append(event, line)
		}
	}

	t.Fatalf("the stream ended before the next event (%v)", f.lines.Err())

	return ""
}

// ends checks that the stream ends with no more events, and the handler with it.
func (f follower) ends(t *testing.T) {
	t.Helper()

	for f.lines.Scan() {
		if line := f.lines.Text(); line != "" && !strings.HasPrefix(line, ":") {
			t.Errorf("got %q after the end", line)
		}
	}

	if err := f.lines.Err(); err != nil {
		t.Errorf("the stream broke: %v", err)
	}

	<-f.served
}

func TestFollowingAPlanStreamsItsConversation(t *testing.T) {
	asked := copilot.Message{ID: "1", From: copilot.User, Text: "Plan a trip"}
	updates := []copilot.Update{
		// It came while the planner read the conversation, which has it already.
		{Change: copilot.ChangeMessage, Message: asked},
		{
			Change:  copilot.ChangeDelta,
			Message: copilot.Message{ID: "m2", From: copilot.Assistant, Text: "Where"},
		},
		{
			Change:  copilot.ChangeMessage,
			Message: copilot.Message{ID: "m2", From: copilot.Assistant, Text: "Where to?"},
		},
		{Change: copilot.ChangePlan, Plan: "# Plan\n"},
	}
	fake := &standIn{
		talk:    copilot.Conversation{Messages: []copilot.Message{asked}},
		updates: make(chan copilot.Update, len(updates)),
	}

	for _, update := range updates {
		fake.updates <- update
	}

	close(fake.updates)

	page := follow(t, fake, time.Hour)

	for _, want := range []string{
		"event: conversation\n" +
			`data: {"messages":[{"id":"1","from":"user","text":"Plan a trip"}],` +
			`"writing":[],"plan":""}`,
		"event: delta\n" + `data: {"id":"m2","from":"assistant","text":"Where"}`,
		"event: message\n" + `data: {"id":"m2","from":"assistant","text":"Where to?"}`,
		"event: plan\n" + `data: {"plan":"# Plan\n"}`,
	} {
		if got := page.next(t); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}

	// The planner has dropped the page, which connects again.
	page.ends(t)
}

func TestFollowingAPlanBeatsWhileItIsQuiet(t *testing.T) {
	page := follow(t, &standIn{updates: make(chan copilot.Update)}, time.Millisecond)

	beat := false
	for !beat && page.lines.Scan() {
		beat = page.lines.Text() == ": still here"
	}

	if !beat {
		t.Errorf("the stream ended without a heartbeat (%v)", page.lines.Err())
	}
}

func TestFollowingAPlanEndsWhenThePageLeaves(t *testing.T) {
	fake := &standIn{updates: make(chan copilot.Update)}
	page := follow(t, fake, time.Hour)

	page.next(t)
	page.leave()
	<-page.served

	if fake.followed.Err() == nil {
		t.Error("the planner still follows the plan for the page")
	}
}

func TestFollowingAPlanEndsWhenTheServiceStops(t *testing.T) {
	page := follow(t, &standIn{updates: make(chan copilot.Update)}, time.Hour)

	page.next(t)
	page.stop()
	page.ends(t)
}

func TestFollowingAPlanByHeadAnswersTheHeadersAlone(t *testing.T) {
	// The service answers HEAD with the GET handler.
	router := chi.NewRouter()
	router.Use(middleware.GetHead)
	api.Door{Planner: &standIn{updates: make(chan copilot.Update)}}.Register(t.Context(), router)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodHead,
		"/planner/plans/plan-1/events", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	sent := rec.Result().Header.Get("Content-Type")
	if rec.Code != http.StatusOK || sent != "text/event-stream" {
		t.Errorf("got %d (%s), want 200 as an event stream", rec.Code, sent)
	}
}

func TestFollowingAnUnknownPlanIsNotFound(t *testing.T) {
	rec := newPlannerAPI(&standIn{}).serve(t, http.MethodGet, "/planner/plans/plan-2/events", "")
	if rec.Code != http.StatusNotFound || rec.Body.String() != "no such plan\n" {
		t.Errorf("got %d %q, want 404 no such plan", rec.Code, rec.Body)
	}
}

func TestFollowingAPlanFailsWhenCopilotFails(t *testing.T) {
	planner := newPlannerAPI(&standIn{failure: errors.New("no connection")})

	rec := planner.serve(t, http.MethodGet, "/planner/plans/plan-1/events", "")
	if rec.Code != http.StatusBadGateway || rec.Body.String() != "Bad Gateway\n" {
		t.Errorf("got %d %q, want 502 Bad Gateway", rec.Code, rec.Body)
	}

	logs := planner.logs.String()
	if !strings.Contains(logs, `msg="follow a plan" err="no connection"`) {
		t.Errorf("logged %q, want the cause", logs)
	}
}
