package main

import (
	"context"
	"log/slog"
	"net/url"
	"time"

	"github.com/codedogapp/bakhyard/internal/env"
	attentionv1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1"
	"github.com/codedogapp/bakhyard/internal/githubapi"
	"github.com/codedogapp/bakhyard/internal/tokenexpiry"
	"github.com/codedogapp/bakhyard/services/planner/internal/copilot"
)

// apiVar is the setting of the REST API of the Copilot account's GitHub host, which planner asks
// when the Copilot token expires (docs/setup.md). It defaults to https://api.<host>, as on
// GitHub.com and GHE.com.
const apiVar = "BAKHYARD_COPILOT_API"

// copilotAPI returns the settings of the REST API of the GitHub host in settings, with its token,
// or zero ones when Copilot is off.
func copilotAPI(getenv func(string) string, settings copilot.Settings) (githubapi.Settings, error) {
	if settings.Host == "" {
		return githubapi.Settings{}, nil
	}

	if getenv(apiVar) == "" {
		address := &url.URL{Scheme: "https", Host: "api." + settings.Host}

		return githubapi.Settings{API: address, Token: settings.Token}, nil
	}

	address, err := env.SecureURL(getenv, apiVar)
	if err != nil {
		return githubapi.Settings{}, err
	}

	return githubapi.Settings{API: address, Token: settings.Token}, nil
}

// copilotToken returns the Copilot token, which expires when the latest ask of the GitHub API in
// settings to succeed says, once its first ask has ended (decision 504). It asks at once and then
// every five minutes, until ctx ends, and logs each ask that fails to logger. It never expires
// before an ask has succeeded, or when Copilot is off and settings are zero.
func copilotToken(
	ctx context.Context,
	settings githubapi.Settings,
	logger *slog.Logger,
) tokenexpiry.Token {
	token := tokenexpiry.Token{
		Name:    "Copilot",
		Source:  attentionv1.Source_SOURCE_COPILOT,
		Expires: tokenexpiry.On(time.Time{}),
	}
	if settings.API != nil {
		token.Expires = tokenexpiry.Watch(ctx, "Copilot token", githubapi.New(settings).Expiry,
			logger)
	}

	return token
}
