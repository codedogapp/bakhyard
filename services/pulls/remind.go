package main

import (
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/codedogapp/bakhyard/internal/env"
	"github.com/codedogapp/bakhyard/internal/notifier"
	"github.com/codedogapp/bakhyard/services/pulls/internal/bitbucket"
	"github.com/codedogapp/bakhyard/services/pulls/internal/remind"
)

// notifyTimeout bounds one request to notify.
const notifyTimeout = 10 * time.Second

// notifyClient returns the client of notify, the peer that pushes notifications to the user's
// devices, or nil when BAKHYARD_PEERS lists no notify.
func notifyClient(getenv func(string) string) (*notifier.Client, error) {
	address, found, err := env.LookupPeer(getenv, "notify")
	if err != nil || !found {
		return nil, err
	}

	return &notifier.Client{
		Notify: &url.URL{Scheme: "http", Host: address},
		HTTP:   &http.Client{Timeout: notifyTimeout},
	}, nil
}

// newReminder returns the reminder of the user's pull requests that mine keeps and that wait for
// approvals, which client reads and notify pushes in the host's time zone, or nil when Bitbucket
// or notify is off.
func newReminder(
	client *bitbucket.Client,
	mine *bitbucket.Poller,
	notify *notifier.Client,
) *remind.Reminder {
	if client == nil || notify == nil {
		return nil
	}

	return &remind.Reminder{
		Mine: mine, Approvals: client.Approvals, Pusher: notify,
		Logger: slog.Default(), Location: time.Local,
	}
}
