package api_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/codedogapp/bakhyard/services/planner/internal/copilot"
)

func TestListingThePlansAnswersThem(t *testing.T) {
	plans := []copilot.Entry{
		{
			ID:        "plan-2",
			StartedAt: time.Date(2026, 1, 3, 3, 4, 5, 5e8, time.UTC),
			UpdatedAt: time.Date(2026, 1, 3, 3, 5, 0, 0, time.UTC),
		},
		{
			ID: "plan-1", Title: "Plan a trip",
			StartedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
			UpdatedAt: time.Date(2026, 1, 2, 4, 0, 0, 0, time.UTC),
		},
	}

	for _, test := range []struct {
		plans []copilot.Entry
		want  string
	}{
		{plans, `{"plans":[{"id":"plan-2","title":"","started_at":"2026-01-03T03:04:05.5Z",` +
			`"updated_at":"2026-01-03T03:05:00Z"},{"id":"plan-1","title":"Plan a trip",` +
			`"started_at":"2026-01-02T03:04:05Z","updated_at":"2026-01-02T04:00:00Z"}]}`},
		{nil, `{"plans":[]}`},
	} {
		rec := newPlannerAPI(&standIn{plans: test.plans}).serve(t, http.MethodGet,
			"/planner/plans", "")

		sent := rec.Result().Header.Get("Content-Type")
		if rec.Code != http.StatusOK || rec.Body.String() != test.want ||
			sent != "application/json" {
			t.Errorf("got %d %s (%s), want 200 %s as JSON", rec.Code, rec.Body, sent, test.want)
		}
	}
}

func TestListingThePlansFailsWhenCopilotFails(t *testing.T) {
	planner := newPlannerAPI(&standIn{failure: errors.New("no connection")})

	rec := planner.serve(t, http.MethodGet, "/planner/plans", "")
	if rec.Code != http.StatusBadGateway || rec.Body.String() != "Bad Gateway\n" {
		t.Errorf("got %d %q, want 502 Bad Gateway", rec.Code, rec.Body)
	}

	logs := planner.logs.String()
	if !strings.Contains(logs, `msg="list the plans" err="no connection"`) {
		t.Errorf("logged %q, want the cause", logs)
	}
}
