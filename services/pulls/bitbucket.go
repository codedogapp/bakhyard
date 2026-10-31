package main

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/codedogapp/bakhyard/internal/env"
	attentionv1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1"
	"github.com/codedogapp/bakhyard/internal/secrets"
	"github.com/codedogapp/bakhyard/internal/tokenexpiry"
	"github.com/codedogapp/bakhyard/services/pulls/internal/bitbucket"
)

// The settings of the Bitbucket workspaces whose pull requests the service lists
// (docs/setup.md). Bitbucket is off when none is set.
const (
	bitbucketAPIVar        = "BAKHYARD_BITBUCKET_API"
	bitbucketEmailVar      = "BAKHYARD_BITBUCKET_EMAIL"
	bitbucketWorkspacesVar = "BAKHYARD_BITBUCKET_WORKSPACES"
)

// bitbucketAccount is the Keychain account of the Bitbucket API token, and the name of its file.
const bitbucketAccount = "bitbucket-token"

// bitbucketExpiresVar holds the date on which the Bitbucket token expires, YYYY-MM-DD, for the
// Home's reminder to renew it (docs/setup.md). Without it, there's no reminder.
const bitbucketExpiresVar = "BAKHYARD_BITBUCKET_TOKEN_EXPIRES"

// bitbucketToken returns the Bitbucket token, which expires on the date in
// BAKHYARD_BITBUCKET_TOKEN_EXPIRES, or never, as far as pulls knows, when that isn't set.
func bitbucketToken(getenv func(string) string) (tokenexpiry.Token, error) {
	expires, err := tokenexpiry.FromEnv(getenv, bitbucketExpiresVar)
	if err != nil {
		return tokenexpiry.Token{}, err
	}

	return tokenexpiry.Token{
		Name:    "Bitbucket",
		Source:  attentionv1.Source_SOURCE_BITBUCKET,
		Expires: tokenexpiry.On(expires),
	}, nil
}

// bitbucketClient returns the client of the Bitbucket workspaces in the settings, with the API
// token that readSecret reads, or nil when none of the Bitbucket settings is set.
func bitbucketClient(
	getenv func(string) string,
	readSecret secrets.Reader,
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

	token, err := readSecret(context.Background(), bitbucketAccount)
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

// poll starts polling Bitbucket with client for the user's open pull requests and for those that
// wait for the user's review, until ctx ends, and returns the pollers that keep what the polls
// find, or nil ones when Bitbucket is off and client is nil.
func poll(
	ctx context.Context,
	client *bitbucket.Client,
	logger *slog.Logger,
) (mine, reviews *bitbucket.Poller) {
	if client == nil {
		return nil, nil
	}

	mine = bitbucket.NewPoller(client.Mine, logger)
	go mine.Run(ctx)

	reviews = bitbucket.NewReviewsPoller(client.Reviews, logger)
	go reviews.Run(ctx)

	return mine, reviews
}
