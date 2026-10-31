package main

import (
	"context"
	"log/slog"

	attentionv1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1"
	"github.com/codedogapp/bakhyard/internal/jira"
	"github.com/codedogapp/bakhyard/internal/tokenexpiry"
	"github.com/codedogapp/bakhyard/services/todos/internal/tickets"
	"github.com/codedogapp/bakhyard/services/todos/internal/todo"
)

// jiraExpiresVar holds the date on which the Jira token expires, YYYY-MM-DD, for the Home's
// reminder to renew it (docs/setup.md). Without it, there's no reminder.
const jiraExpiresVar = "BAKHYARD_JIRA_TOKEN_EXPIRES"

// jiraToken returns the Jira token, which expires on the date in BAKHYARD_JIRA_TOKEN_EXPIRES, or
// never, as far as todos knows, when that isn't set.
func jiraToken(getenv func(string) string) (tokenexpiry.Token, error) {
	expires, err := tokenexpiry.FromEnv(getenv, jiraExpiresVar)
	if err != nil {
		return tokenexpiry.Token{}, err
	}

	return tokenexpiry.Token{
		Name: "Jira", Source: attentionv1.Source_SOURCE_JIRA, Expires: expires,
	}, nil
}

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
