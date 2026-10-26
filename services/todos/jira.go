package main

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"slices"

	"github.com/codedogapp/bakhyard/internal/env"
	"github.com/codedogapp/bakhyard/services/todos/internal/jira"
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

	site, err := jiraURL(getenv, jiraSiteVar)
	if err != nil {
		return nil, err
	}

	address, err := jiraURL(getenv, jiraAPIVar)
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

// jiraURL returns the URL in the setting name. It must be an https URL, so that the token
// doesn't travel in the clear, or an http one to a loopback IP address, for a stand-in Jira.
func jiraURL(getenv func(string) string, name string) (*url.URL, error) {
	value, err := env.Required(getenv, name)
	if err != nil {
		return nil, err
	}

	address, err := url.Parse(value)
	if err != nil || address.Host == "" || !secure(address) {
		return nil, fmt.Errorf("%s: %q is not an https URL", name, value)
	}

	return address, nil
}

// secure reports whether address keeps a token on the way to it from other machines' eyes.
func secure(address *url.URL) bool {
	switch address.Scheme {
	case "https":
		return true
	case "http":
		return net.ParseIP(address.Hostname()).IsLoopback()
	default:
		return false
	}
}
