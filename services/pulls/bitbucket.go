package main

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/codedogapp/bakhyard/internal/env"
	"github.com/codedogapp/bakhyard/services/pulls/internal/bitbucket"
)

// The settings of the Bitbucket workspaces whose pull requests the service lists
// (docs/setup.md). Bitbucket is off when none is set.
const (
	bitbucketAPIVar        = "BAKHYARD_BITBUCKET_API"
	bitbucketEmailVar      = "BAKHYARD_BITBUCKET_EMAIL"
	bitbucketWorkspacesVar = "BAKHYARD_BITBUCKET_WORKSPACES"
)

// bitbucketAccount names the login Keychain item, under secretService, that holds the Bitbucket
// API token.
const bitbucketAccount = "bitbucket-token"

// bitbucketClient returns the client of the Bitbucket workspaces in the settings, with the API
// token from the Keychain, or nil when none of the Bitbucket settings is set.
func bitbucketClient(
	getenv func(string) string,
	password passwordReader,
) (*bitbucket.Client, error) {
	settings := []string{bitbucketAPIVar, bitbucketEmailVar, bitbucketWorkspacesVar}
	if !slices.ContainsFunc(settings, func(name string) bool { return getenv(name) != "" }) {
		return nil, nil
	}

	address, err := env.SecureURL(getenv, bitbucketAPIVar)
	if err != nil {
		return nil, err
	}

	email, err := env.Required(getenv, bitbucketEmailVar)
	if err != nil {
		return nil, err
	}

	slugs, err := workspaces(getenv)
	if err != nil {
		return nil, err
	}

	token, err := password(context.Background(), secretService, bitbucketAccount)
	if err != nil {
		return nil, fmt.Errorf("read the Bitbucket token: %w", err)
	}

	return bitbucket.New(bitbucket.Settings{
		API: address, Email: email, Token: token, Workspaces: slugs,
	}), nil
}

// workspaces returns the workspace slugs in their setting, which separates them with commas,
// sorted and each once. Spaces around a slug don't count, and no slug may be empty, . or ..,
// which the client's path would lose (decision 146).
func workspaces(getenv func(string) string) ([]string, error) {
	value, err := env.Required(getenv, bitbucketWorkspacesVar)
	if err != nil {
		return nil, err
	}

	var slugs []string

	for slug := range strings.SplitSeq(value, ",") {
		slug = strings.TrimSpace(slug)
		if slug == "" || slug == "." || slug == ".." {
			return nil, fmt.Errorf("%s: %q is not a list of workspace slugs",
				bitbucketWorkspacesVar, value)
		}

		slugs = append(slugs, slug)
	}

	slices.Sort(slugs)

	return slices.Compact(slugs), nil
}

// poll starts polling Bitbucket with client for the user's open pull requests, until ctx ends,
// and returns the poller that keeps what the polls find, or nil when Bitbucket is off and client
// is nil.
func poll(ctx context.Context, client *bitbucket.Client, logger *slog.Logger) *bitbucket.Poller {
	if client == nil {
		return nil
	}

	poller := bitbucket.NewPoller(client.Mine, logger)
	go poller.Run(ctx)

	return poller
}
