package session_test

import (
	"errors"
	"testing"
	"time"

	"github.com/codedogapp/bakhyard/services/nudge/internal/change"
	"github.com/codedogapp/bakhyard/services/nudge/internal/session"
)

func projects(list []session.Session) []string {
	names := make([]string, 0, len(list))
	for _, s := range list {
		names = append(names, s.Project)
	}

	return names
}

func TestListShowsRunningSessionsNewestFirstAndEndingTwiceIsHarmless(t *testing.T) {
	registry := session.NewRegistry(&change.Hub{})

	_, endFirst := registry.Start("first")
	registry.Start("second")

	if got := projects(registry.List()); len(got) != 2 || got[0] != "second" || got[1] != "first" {
		t.Errorf("projects = %v, want [second first]", got)
	}

	endFirst()
	endFirst()

	if got := projects(registry.List()); len(got) != 1 || got[0] != "second" {
		t.Errorf("projects after the first ended = %v, want [second]", got)
	}
}

func TestReportReplacesTheNoteAndPublishesAChange(t *testing.T) {
	hub := &change.Hub{}
	changes, end := hub.Subscribe()

	defer end()

	registry := session.NewRegistry(hub)
	started, _ := registry.Start("nudge")

	<-changes

	before := time.Now()

	for _, note := range []string{"  reading the code \n", "running the tests"} {
		if err := registry.Report(started.ID, note); err != nil {
			t.Fatalf("Report(%q) = %v", note, err)
		}
	}

	<-changes

	got := registry.List()[0]
	if got.Note != "running the tests" || got.NotedAt.Before(before) {
		t.Errorf("got %+v, want the last note, noted after %v", got, before)
	}
}

func TestReportRefusesAnEmptyNoteAndAnEndedSession(t *testing.T) {
	registry := session.NewRegistry(&change.Hub{})
	started, end := registry.Start("nudge")

	if err := registry.Report(started.ID, " \n"); !errors.Is(err, session.ErrNoNote) {
		t.Errorf("an empty note: got %v", err)
	}

	end()

	if err := registry.Report(started.ID, "x"); !errors.Is(err, session.ErrNotFound) {
		t.Errorf("an ended session: got %v", err)
	}
}
