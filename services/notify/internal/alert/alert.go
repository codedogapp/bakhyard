// Package alert reads Alertmanager's webhook, a group of alerts, and words the push that tells of
// it (decision 601).
package alert

import (
	"cmp"
	"errors"
	"fmt"
	"strings"

	"github.com/codedogapp/bakhyard/services/notify/internal/push"
)

// Status is whether alerts fire or have resolved.
type Status string

// The statuses that Alertmanager gives a group and each of its alerts.
const (
	Firing   Status = "firing"
	Resolved Status = "resolved"
)

const (
	// fallbackTitle titles a group whose alerts share neither a summary nor a name.
	fallbackTitle = "Alerts"
	// home is the path that a tap on the push opens, since the app has no page for alerts.
	home = "/"
)

// ErrInvalid means that a webhook isn't a group of alerts that fire or have resolved.
var ErrInvalid = errors.New("the webhook needs a status, firing or resolved, and an alert")

// Group is the body of Alertmanager's webhook, version 4, as far as the push needs it: a group of
// alerts, which fires while any of them fires, the labels and annotations that they share, and
// how many alerts Alertmanager left out.
type Group struct {
	Status            Status            `json:"status"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	TruncatedAlerts   int               `json:"truncatedAlerts"`
	Alerts            []Alert           `json:"alerts"`
}

// Alert is one of a group's alerts.
type Alert struct {
	Status      Status            `json:"status"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

// Message words g's push, which opens the Home. Its title is the summary that g's alerts share,
// or else their name, after "Resolved: " once g has resolved. While g fires, its body has a line
// for each alert that fires, the alert's description, or else its summary or name.
func (g Group) Message() (push.Message, error) {
	if len(g.Alerts) == 0 || (g.Status != Firing && g.Status != Resolved) {
		return push.Message{}, ErrInvalid
	}

	title := cmp.Or(g.CommonAnnotations["summary"], g.CommonLabels["alertname"], fallbackTitle)
	if g.Status == Resolved {
		return push.Message{Title: "Resolved: " + title, URL: home}, nil
	}

	return push.Message{Title: title, Body: g.firing(), URL: home}, nil
}

// firing lists g's alerts that fire, a line each, and then how many alerts Alertmanager left out.
func (g Group) firing() string {
	var lines []string

	for _, a := range g.Alerts {
		if a.Status == Firing {
			lines = append(lines, a.line())
		}
	}

	if g.TruncatedAlerts > 0 {
		lines = append(lines, fmt.Sprintf("and %d more", g.TruncatedAlerts))
	}

	return strings.Join(lines, "\n")
}

// line tells of a in a line: its description, or else its summary or name.
func (a Alert) line() string {
	return cmp.Or(a.Annotations["description"], a.Annotations["summary"], a.Labels["alertname"])
}
