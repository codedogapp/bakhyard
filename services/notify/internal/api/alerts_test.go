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

type fakeKeeper struct {
	err  error
	kept [][]alert.Alert
}

func (f *fakeKeeper) Keep(_ context.Context, alerts []alert.Alert) error {
	f.kept = append(f.kept, alerts)

	return f.err
}

func TestAlertsKeepTheAlertsAndPushTheGroup(t *testing.T) {
	sender, keeper := &fakeSender{}, &fakeKeeper{}
	router, _ := serve(t, api.Backends{Devices: fakeDevices{}, Sender: sender, Alerts: keeper})

	do(router, http.MethodPost, "/internal/alerts", webhook)

	if len(keeper.kept) != 1 || len(keeper.kept[0]) != 1 {
		t.Fatalf("kept %+v, want the one alert", keeper.kept)
	}

	started := time.Date(2026, 10, 30, 12, 0, 0, 123456789, time.UTC)
	if got := keeper.kept[0][0]; got.Fingerprint != "c4f6a1b2d3e4f5a6" ||
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
		Devices: fakeDevices{}, Sender: sender, Alerts: &fakeKeeper{err: errors.New("down")},
	})

	rec := do(router, http.MethodPost, "/internal/alerts", webhook)
	if rec.Code != http.StatusInternalServerError || len(sender.sent) != 0 ||
		!strings.Contains(logs.String(), "send the alerts") {
		t.Errorf("got %d, sent %+v and logged %q, want a logged 500 and no push",
			rec.Code, sender.sent, logs)
	}
}

func TestAlertsSayWhatAGroupNeeds(t *testing.T) {
	sender, keeper := &fakeSender{}, &fakeKeeper{}
	router, _ := serve(t, api.Backends{Devices: fakeDevices{}, Sender: sender, Alerts: keeper})

	rec := do(router, http.MethodPost, "/internal/alerts", `{"status":"pending","alerts":[{}]}`)
	if rec.Code != http.StatusBadRequest || rec.Body.String() != alert.ErrInvalid.Error()+"\n" ||
		len(sender.sent) != 0 || len(keeper.kept) != 0 {
		t.Errorf("got %d %q, sent %+v and kept %+v, want a 400 that says what's missing",
			rec.Code, rec.Body, sender.sent, keeper.kept)
	}
}
