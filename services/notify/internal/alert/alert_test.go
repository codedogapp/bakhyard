package alert_test

import (
	"errors"
	"testing"

	"github.com/codedogapp/bakhyard/services/notify/internal/alert"
	"github.com/codedogapp/bakhyard/services/notify/internal/push"
)

// diskFull is a firing alert with a summary and a description, as each rule of the laptop's
// has, about the filesystem at mountpoint.
func diskFull(status alert.Status, mountpoint string) alert.Alert {
	return alert.Alert{
		Status: status,
		Labels: map[string]string{"alertname": "DiskFull", "mountpoint": mountpoint},
		Annotations: map[string]string{
			"summary":     "A filesystem is nearly full",
			"description": mountpoint + " is 91% full",
		},
	}
}

// group is a group of alerts with the summary and name that the alerts share.
func group(status alert.Status, alerts ...alert.Alert) alert.Group {
	return alert.Group{
		Status:            status,
		CommonLabels:      map[string]string{"alertname": "DiskFull"},
		CommonAnnotations: map[string]string{"summary": "A filesystem is nearly full"},
		Alerts:            alerts,
	}
}

func TestMessageWordsThePush(t *testing.T) {
	const (
		summary = "A filesystem is nearly full"
		page    = "/alerts"
	)

	nameless := group(alert.Firing, diskFull(alert.Firing, "/"))
	nameless.CommonLabels, nameless.CommonAnnotations = nil, nil
	truncated := group(alert.Firing, diskFull(alert.Firing, "/"))
	truncated.TruncatedAlerts = 3

	tests := []struct {
		name  string
		group alert.Group
		want  push.Message
	}{
		{
			"firing",
			group(alert.Firing, diskFull(alert.Firing, "/"), diskFull(alert.Firing, "/data")),
			push.Message{Title: summary, Body: "/ is 91% full\n/data is 91% full", URL: page},
		},
		{
			"resolved",
			group(alert.Resolved, diskFull(alert.Resolved, "/"), diskFull(alert.Resolved, "/data")),
			push.Message{Title: "Resolved: " + summary, URL: page},
		},
		{
			"a resolved alert left out of a firing group's body",
			group(alert.Firing, diskFull(alert.Resolved, "/"), diskFull(alert.Firing, "/data")),
			push.Message{Title: summary, Body: "/data is 91% full", URL: page},
		},
		{
			"the alerts that Alertmanager left out",
			truncated,
			push.Message{Title: summary, Body: "/ is 91% full\nand 3 more", URL: page},
		},
		{
			"the name when the alerts share no summary",
			alert.Group{
				Status:       alert.Resolved,
				CommonLabels: map[string]string{"alertname": "DiskFull"},
				Alerts:       []alert.Alert{diskFull(alert.Resolved, "/")},
			},
			push.Message{Title: "Resolved: DiskFull", URL: page},
		},
		{
			"a title when the alerts share neither",
			nameless,
			push.Message{Title: "Alerts", Body: "/ is 91% full", URL: page},
		},
		{
			"an alert's summary or name when it has no description",
			group(
				alert.Firing,
				alert.Alert{
					Status:      alert.Firing,
					Labels:      map[string]string{"alertname": "DiskFull"},
					Annotations: map[string]string{"summary": "/ is nearly full"},
				},
				alert.Alert{
					Status: alert.Firing,
					Labels: map[string]string{"alertname": "DiskFull"},
				},
			),
			push.Message{Title: summary, Body: "/ is nearly full\nDiskFull", URL: page},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.group.Message()
			if err != nil || got != test.want {
				t.Errorf("got %+v, %v, want %+v", got, err, test.want)
			}
		})
	}
}

func TestMessageRefusesWhatIsntAGroupOfAlerts(t *testing.T) {
	for name, group := range map[string]alert.Group{
		"nothing":        {},
		"no status":      group("", diskFull(alert.Firing, "/")),
		"another status": group("pending", diskFull(alert.Firing, "/")),
		"no alert":       group(alert.Firing),
	} {
		t.Run(name, func(t *testing.T) {
			if got, err := group.Message(); !errors.Is(err, alert.ErrInvalid) {
				t.Errorf("got %+v, %v, want %v", got, err, alert.ErrInvalid)
			}
		})
	}
}
