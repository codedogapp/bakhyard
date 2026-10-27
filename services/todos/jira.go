package main

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/codedogapp/bakhyard/internal/env"
	"github.com/codedogapp/bakhyard/services/todos/internal/jira"
	"github.com/codedogapp/bakhyard/services/todos/internal/todo"
)

// The settings of the Jira site whose tickets the service lists (docs/setup.md). Jira is off
// when none is set.
const (
	jiraSiteVar  = "BAKHYARD_JIRA_SITE"
	jiraAPIVar   = "BAKHYARD_JIRA_API"
	jiraEmailVar = "BAKHYARD_JIRA_EMAIL"
)

// jiraAccount names the login Keychain item, under secretService, that holds the Jira API token.
const jiraAccount = "jira-token"

// jiraClient returns the client of the Jira site in the settings, with the API token from the
// Keychain, or nil when none of the Jira settings is set.
func jiraClient(getenv func(string) string, password passwordReader) (*jira.Client, error) {
	settings := []string{jiraSiteVar, jiraAPIVar, jiraEmailVar}
	if !slices.ContainsFunc(settings, func(name string) bool { return getenv(name) != "" }) {
		return nil, nil
	}

	site, err := env.SecureURL(getenv, jiraSiteVar)
	if err != nil {
		return nil, err
	}

	address, err := env.SecureURL(getenv, jiraAPIVar)
	if err != nil {
		return nil, err
	}

	email, err := env.Required(getenv, jiraEmailVar)
	if err != nil {
		return nil, err
	}

	token, err := password(context.Background(), secretService, jiraAccount)
	if err != nil {
		return nil, fmt.Errorf("read the Jira token: %w", err)
	}

	return jira.New(jira.Settings{Site: site, API: address, Email: email, Token: token}), nil
}

// poll starts polling Jira with client for the user's open tickets and the tickets that store's
// todos name, until ctx ends, and returns the poller that keeps what the polls find, or nil
// when Jira is off and client is nil.
func poll(
	ctx context.Context, client *jira.Client, store *todo.Store, logger *slog.Logger,
) *jira.Poller {
	if client == nil {
		return nil
	}

	poller := jira.NewPoller(client, store.Linked, logger)
	go poller.Run(ctx)

	return poller
}
