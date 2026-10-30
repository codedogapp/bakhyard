package api_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/codedogapp/bakhyard/services/notify/internal/alert"
	"github.com/codedogapp/bakhyard/services/notify/internal/api"
	"github.com/codedogapp/bakhyard/services/notify/internal/history"
	"github.com/codedogapp/bakhyard/services/notify/internal/push"
)

// webhook is a firing group of one alert, as Alertmanager posts it, with members that notify
// doesn't read.
const webhook = `{
	"version": "4",
	"groupKey": "{}:{alertname=\"DiskFull\"}",
	"truncatedAlerts": 0,
	"status": "firing",
	"receiver": "bakhyard",
	"groupLabels": {"alertname": "DiskFull"},
	"commonLabels": {"alertname": "DiskFull", "severity": "warning"},
	"commonAnnotations": {"summary": "A filesystem is nearly full"},
	"externalURL": "http://alertmanager:9093",
	"notification_reason": "first notification",
	"alerts": [{
		"status": "firing",
		"labels": {"alertname": "DiskFull", "mountpoint": "/"},
		"annotations": {"summary": "A filesystem is nearly full", "description": "/ is 91% full"},
		"startsAt": "2026-10-30T12:00:00.123456789Z",
		"endsAt": "0001-01-01T00:00:00Z",
		"generatorURL": "http://vmalert:8880/vmalert/alert?group_id=1&alert_id=2",
		"fingerprint": "c4f6a1b2d3e4f5a6"
	}]
}`

type fakeHistory struct {
	err     error
	kept    [][]alert.Alert
	entries []history.Entry
}

func (f *fakeHistory) Keep(_ context.Context, alerts []alert.Alert) error {
	f.kept = append(f.kept, alerts)

	return f.err
}

func (f *fakeHistory) List(context.Context) ([]history.Entry, error) {
	return f.entries, f.err
}

func TestAlertsKeepTheAlertsAndPushTheGroup(t *testing.T) {
	sender, alerts := &fakeSender{}, &fakeHistory{}
	router, _ := serve(t, api.Backends{Devices: fakeDevices{}, Sender: sender, Alerts: alerts})

	do(router, http.MethodPost, "/internal/alerts", webhook)

	if len(alerts.kept) != 1 || len(alerts.kept[0]) != 1 {
		t.Fatalf("kept %+v, want the one alert", alerts.kept)
	}

	started := time.Date(2026, 10, 30, 12, 0, 0, 123456789, time.UTC)
	if got := alerts.kept[0][0]; got.Fingerprint != "c4f6a1b2d3e4f5a6" ||
		!got.StartsAt.Equal(started) || !got.EndsAt.IsZero() ||
		got.Labels["alertname"] != "DiskFull" {
		t.Errorf("kept %+v, want DiskFull's alert, started at %v, not ended", got, started)
	}

	want := push.Message{Title: "A filesystem is nearly full", Body: "/ is 91% full", URL: "/"}
	if len(sender.sent) != 1 || sender.sent[0] != want {
		t.Errorf("sent %+v, want %+v", sender.sent, want)
	}
}

func TestAlertsThatCantBeKeptAreNotPushed(t *testing.T) {
	sender := &fakeSender{}
	router, logs := serve(t, api.Backends{
		Devices: fakeDevices{}, Sender: sender, Alerts: &fakeHistory{err: errors.New("down")},
	})

	rec := do(router, http.MethodPost, "/internal/alerts", webhook)
	if rec.Code != http.StatusInternalServerError || len(sender.sent) != 0 ||
		!strings.Contains(logs.String(), "send the alerts") {
		t.Errorf("got %d, sent %+v and logged %q, want a logged 500 and no push",
			rec.Code, sender.sent, logs)
	}
}

func TestAlertsSayWhatAGroupNeeds(t *testing.T) {
	sender, alerts := &fakeSender{}, &fakeHistory{}
	router, _ := serve(t, api.Backends{Devices: fakeDevices{}, Sender: sender, Alerts: alerts})

	rec := do(router, http.MethodPost, "/internal/alerts", `{"status":"pending","alerts":[{}]}`)
	if rec.Code != http.StatusBadRequest || rec.Body.String() != alert.ErrInvalid.Error()+"\n" ||
		len(sender.sent) != 0 || len(alerts.kept) != 0 {
		t.Errorf("got %d %q, sent %+v and kept %+v, want a 400 that says what's missing",
			rec.Code, rec.Body, sender.sent, alerts.kept)
	}
}

func TestAlertsListAnswersTheKeptAlerts(t *testing.T) {
	alerts := &fakeHistory{entries: []history.Entry{
		{
			Name:        "OnBattery",
			Summary:     "The laptop runs on its battery",
			Description: "Plug it in",
			StartedAt:   time.Date(2026, 10, 30, 13, 0, 0, 0, time.UTC),
		},
		{
			Name:        "DiskFull",
			Summary:     "A filesystem is nearly full",
			Description: "/ is 91% full",
			StartedAt:   time.Date(2026, 10, 30, 12, 0, 0, 0, time.FixedZone("", 3600)),
			ResolvedAt:  time.Date(2026, 10, 30, 12, 30, 0, 123456000, time.UTC),
		},
	}}
	router, _ := serve(
		t,
		api.Backends{Devices: fakeDevices{}, Sender: &fakeSender{}, Alerts: alerts},
	)

	rec := do(router, http.MethodGet, "/notify/alerts", "")

	want := `{"alerts":[{"name":"OnBattery","summary":"The laptop runs on its battery",` +
		`"description":"Plug it in","startedAt":"2026-10-30T13:00:00Z"},` +
		`{"name":"DiskFull","summary":"A filesystem is nearly full",` +
		`"description":"/ is 91% full","startedAt":"2026-10-30T12:00:00+01:00",` +
		`"resolvedAt":"2026-10-30T12:30:00.123456Z"}]}`
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/json" ||
		rec.Body.String() != want {
		t.Errorf("got %d %q %s, want 200 and %s", rec.Code, rec.Header().Get("Content-Type"),
			rec.Body, want)
	}
}

func TestAlertsListAnswersByItsOutcome(t *testing.T) {
	for _, test := range []struct {
		name, method string
		alerts       *fakeHistory
		want         int
		body, logged string
	}{
		{"none", http.MethodGet, &fakeHistory{}, http.StatusOK, `{"alerts":[]}`, ""},
		{
			"a failure", http.MethodGet, &fakeHistory{err: errors.New("down")},
			http.StatusInternalServerError, "Internal Server Error\n", "list the alerts",
		},
		{"another method", http.MethodPut, &fakeHistory{}, http.StatusMethodNotAllowed, "", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			router, logs := serve(t, api.Backends{
				Devices: fakeDevices{}, Sender: &fakeSender{}, Alerts: test.alerts,
			})

			rec := do(router, test.method, "/notify/alerts", "")
			if rec.Code != test.want || rec.Body.String() != test.body ||
				!strings.Contains(logs.String(), test.logged) {
				t.Errorf("got %d %q with the log %q, want %d %q", rec.Code, rec.Body, logs,
					test.want, test.body)
			}
		})
	}
}
