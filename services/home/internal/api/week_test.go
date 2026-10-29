package api_test

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"
	"time"

	"github.com/codedogapp/bakhyard/services/home/internal/api"
	"github.com/codedogapp/bakhyard/services/home/internal/screen"
)

func TestHomeGetsTheWeek(t *testing.T) {
	// A missing member reads as "".
	tests := map[string]struct {
		week                *screen.Week
		state               screen.State
		wantWeek, wantState string
	}{
		"a week": {
			week: &screen.Week{LoggedHours: 31.5, TargetHours: 40}, state: screen.StateOK,
			wantWeek: `{"loggedHours":31.5,"targetHours":40}`, wantState: `"ok"`,
		},
		"a week with nothing logged": {
			week: &screen.Week{TargetHours: 40}, state: screen.StateOK,
			wantWeek: `{"loggedHours":0,"targetHours":40}`, wantState: `"ok"`,
		},
		"no week":          {state: screen.StateOK, wantState: `"ok"`},
		"time unavailable": {state: screen.StateUnavailable, wantState: `"unavailable"`},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := members(t, serveWith(t, http.MethodGet,
				api.Peers{Items: nothing, Week: weekAnswer(test.week, test.state)}, at))

			week, state := string(got["week"]), string(got["weekState"])
			if week != test.wantWeek || state != test.wantState {
				t.Errorf("week = %s and weekState = %s, want %s and %s", week, state,
					test.wantWeek, test.wantState)
			}
		})
	}
}

func TestHomeAsksForTheItemsAndTheWeekAtOnce(t *testing.T) {
	// In a bubble the fake clock moves only once every goroutine waits, so sleeps take no time.
	synctest.Test(t, func(t *testing.T) {
		items := func(context.Context) ([]screen.Item, []screen.SourceStatus) {
			time.Sleep(time.Second)

			return nil, nil
		}
		week := func(context.Context) (*screen.Week, screen.State) {
			time.Sleep(2 * time.Second)

			return &screen.Week{LoggedHours: 8, TargetHours: 40}, screen.StateOK
		}

		start := time.Now()
		got := members(t, serveWith(t, http.MethodGet, api.Peers{Items: items, Week: week}, at))

		if waited, want := time.Since(start), 2*time.Second; waited != want {
			t.Errorf("the Home took %v, want %v: the slower ask alone", waited, want)
		}

		if week, want := string(got["week"]), `{"loggedHours":8,"targetHours":40}`; week != want {
			t.Errorf("week = %s, want %s", week, want)
		}
	})
}

// weekAnswer returns a week ask that answers week and state.
func weekAnswer(
	week *screen.Week, state screen.State,
) func(context.Context) (*screen.Week, screen.State) {
	return func(context.Context) (*screen.Week, screen.State) { return week, state }
}

// members returns the Home's JSON members, by name.
func members(t *testing.T, rec *httptest.ResponseRecorder) map[string]jsontext.Value {
	t.Helper()

	var got map[string]jsontext.Value
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	return got
}
