package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/codedogapp/bakhyard/internal/env"
	attentionv1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1"
	"github.com/codedogapp/bakhyard/internal/secrets"
	"github.com/codedogapp/bakhyard/internal/tokenexpiry"
	"github.com/codedogapp/bakhyard/services/github/internal/github"
)

// githubAPIVar is the setting of GitHub's REST API, https://api.github.com (docs/setup.md).
// GitHub is off when it isn't set.
const githubAPIVar = "BAKHYARD_GITHUB_API"

// githubAccount is the Keychain account of the GitHub token, and the name of its file (decision
// 202).
const githubAccount = "github-token"

// githubClient returns the client of GitHub's API in the settings, with the token that readSecret
// reads, or nil when the API isn't set.
func githubClient(
	getenv func(string) string,
	readSecret secrets.Reader,
) (*github.Client, error) {
	if getenv(githubAPIVar) == "" {
		return nil, nil
	}

	address, err := env.SecureURL(getenv, githubAPIVar)
	if err != nil {
		return nil, err
	}

	token, err := readSecret(context.Background(), githubAccount)
	if err != nil {
		return nil, fmt.Errorf("read the GitHub token: %w", err)
	}

	return github.New(github.Settings{API: address, Token: token}), nil
}

// poll starts searching GitHub with client for the user's open pull requests, for those that wait
// for the user's review and for the open issues assigned to the user, until ctx ends. It returns
// the pollers that keep what the searches find, or nil ones when GitHub is off and client is nil.
func poll(
	ctx context.Context,
	client *github.Client,
	logger *slog.Logger,
) (mine, reviews, issues *github.Poller) {
	if client == nil {
		return nil, nil, nil
	}

	mine = github.NewPoller("GitHub", client.Mine, logger)
	go mine.Run(ctx)

	reviews = github.NewPoller("GitHub reviews", client.Reviews, logger)
	go reviews.Run(ctx)

	issues = github.NewPoller("GitHub issues", client.Issues, logger)
	go issues.Run(ctx)

	return mine, reviews, issues
}

// watchToken starts asking GitHub with client when the user's token expires, until ctx ends. It
// returns the poller that keeps the answer, or nil when GitHub is off and client is nil.
func watchToken(
	ctx context.Context,
	client *github.Client,
	logger *slog.Logger,
) *github.ExpiryPoller {
	if client == nil {
		return nil
	}

	expiry := github.NewExpiryPoller(client.Expiry, logger)
	go expiry.Run(ctx)

	return expiry
}

// githubToken returns the GitHub token, which expires when the latest ask of expiry to succeed
// says, once its first ask has ended (decision 503). It never expires before an ask has
// succeeded, or when GitHub is off and expiry is nil.
func githubToken(expiry *github.ExpiryPoller) tokenexpiry.Token {
	expires := func(ctx context.Context) (time.Time, error) {
		if expiry == nil {
			return time.Time{}, nil
		}

		latest, err := expiry.Latest(ctx)

		return latest.Found.Expires, err
	}

	return tokenexpiry.Token{
		Name: "GitHub", Source: attentionv1.Source_SOURCE_GITHUB, Expires: expires,
	}
}
