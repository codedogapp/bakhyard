package main

import (
	"net/http"
	"net/url"
	"time"

	"github.com/codedogapp/bakhyard/internal/notifier"
	"github.com/codedogapp/bakhyard/internal/requestid"
)

// notifyTimeout bounds one request to notify.
const notifyTimeout = 10 * time.Second

// notifyClient returns the client of notify, the peer at address that pushes the agents'
// questions to the user's devices. Each push carries the ID of the request whose handling
// sends it.
func notifyClient(address string) notifier.Client {
	return notifier.Client{
		Notify: &url.URL{Scheme: "http", Host: address},
		HTTP: &http.Client{
			Timeout:   notifyTimeout,
			Transport: requestid.Transport(http.DefaultTransport),
		},
	}
}
