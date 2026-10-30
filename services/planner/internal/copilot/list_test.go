package copilot

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

// listed is the CLI's list of its sessions: a plan that the CLI hasn't named yet, then a plan of
// an earlier run, which it has named, updated since.
const listed = `{"sessions": [
	{"sessionId": "plan-2", "startTime": "2026-01-03T03:04:05.5Z",
		"modifiedTime": "2026-01-03T03:05:00Z", "isRemote": false, "context": {"cwd": "/work"}},
	{"sessionId": "plan-1", "startTime": "2026-01-02T03:04:05Z",
		"modifiedTime": "2026-01-04T04:00:00Z", "summary": "Plan a trip", "isRemote": false,
		"context": {"cwd": "/work"}}
]}`

func TestPlans(t *testing.T) {
	fake := newFakeCLI("", "")

	plans, err := started(t, fake).Plans(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	want := []Entry{
		{
			ID: "plan-1", Title: "Plan a trip",
			StartedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
			UpdatedAt: time.Date(2026, 1, 4, 4, 0, 0, 0, time.UTC),
		},
		{
			ID:        "plan-2",
			StartedAt: time.Date(2026, 1, 3, 3, 4, 5, 5e8, time.UTC),
			UpdatedAt: time.Date(2026, 1, 3, 3, 5, 0, 0, time.UTC),
		},
	}
	if !reflect.DeepEqual(plans, want) {
		t.Errorf("Plans = %+v, want %+v", plans, want)
	}

	var filter map[string]any
	if !fake.asks(t, "session.list", &filter) || len(filter) != 0 {
		t.Errorf("session.list = %v, want every session, without a filter", filter)
	}
}

func TestPlansFails(t *testing.T) {
	plans, err := started(t, newFakeCLI("", "session.list")).Plans(t.Context())
	if err == nil || !strings.HasPrefix(err.Error(), "list the sessions: ") || plans != nil {
		t.Errorf("Plans = %v, %v, want an error from listing the sessions", plans, err)
	}
}
