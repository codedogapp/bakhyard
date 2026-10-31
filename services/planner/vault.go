package main

import (
	"net/http"
	"net/url"
	"time"

	"github.com/codedogapp/bakhyard/internal/env"
	"github.com/codedogapp/bakhyard/internal/requestid"
	"github.com/codedogapp/bakhyard/services/planner/internal/api"
	"github.com/codedogapp/bakhyard/services/planner/internal/vault"
)

// notesTimeout bounds one request to notes, which bounds each of its calls to Gitea at 10 s: a new
// note takes one, or a few when its name is taken.
const notesTimeout = 30 * time.Second

// vaultClient returns the vault that keeps plans through notes, the peer that serves the Obsidian
// vault, or nil when BAKHYARD_PEERS lists no notes. Each save carries the ID of the request whose
// handling saves.
func vaultClient(getenv func(string) string) (api.Saver, error) {
	address, found, err := env.LookupPeer(getenv, "notes")
	if err != nil || !found {
		return nil, err
	}

	return vault.Client{
		Notes: &url.URL{Scheme: "http", Host: address},
		HTTP: &http.Client{
			Timeout:   notesTimeout,
			Transport: requestid.Transport(http.DefaultTransport),
		},
		Now: time.Now,
	}, nil
}
