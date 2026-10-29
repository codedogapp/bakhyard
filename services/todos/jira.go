package main

import (
	"context"
	"log/slog"

	"github.com/codedogapp/bakhyard/internal/jira"
	"github.com/codedogapp/bakhyard/services/todos/internal/tickets"
	"github.com/codedogapp/bakhyard/services/todos/internal/todo"
)

// poll starts polling Jira with client for the user's open tickets and the tickets that store's
// todos name, until ctx ends, and returns the poller that keeps what the polls find, or nil
// when Jira is off and client is nil.
func poll(
	ctx context.Context, client *jira.Client, store *todo.Store, logger *slog.Logger,
) *tickets.Poller {
	if client == nil {
		return nil
	}

	poller := tickets.NewPoller(client, store.Linked, logger)
	go poller.Run(ctx)

	return poller
}
