package jira

import (
	"context"
	"fmt"
	"slices"

	"github.com/codedogapp/bakhyard/internal/env"
	"github.com/codedogapp/bakhyard/internal/secrets"
)

// The settings of the Jira site (docs/setup.md). Jira is off when none is set.
const (
	siteVar  = "BAKHYARD_JIRA_SITE"
	apiVar   = "BAKHYARD_JIRA_API"
	emailVar = "BAKHYARD_JIRA_EMAIL"
)

// tokenAccount is the Keychain account of the Jira API token, and the name of its file.
const tokenAccount = "jira-token"

// FromEnv returns the client of the Jira site in the settings that getenv reads, with the API
// token that readSecret reads, or nil when none of the Jira settings is set.
func FromEnv(getenv func(string) string, readSecret secrets.Reader) (*Client, error) {
	settings := []string{siteVar, apiVar, emailVar}
	if !slices.ContainsFunc(settings, func(name string) bool { return getenv(name) != "" }) {
		return nil, nil
	}

	site, err := env.SecureURL(getenv, siteVar)
	if err != nil {
		return nil, err
	}

	address, err := env.SecureURL(getenv, apiVar)
	if err != nil {
		return nil, err
	}

	email, err := env.Required(getenv, emailVar)
	if err != nil {
		return nil, err
	}

	token, err := readSecret(context.Background(), tokenAccount)
	if err != nil {
		return nil, fmt.Errorf("read the Jira token: %w", err)
	}

	return New(Settings{Site: site, API: address, Email: email, Token: token}), nil
}
