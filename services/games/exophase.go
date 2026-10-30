package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/codedogapp/bakhyard/services/games/internal/api"
	"github.com/codedogapp/bakhyard/services/games/internal/covers"
	"github.com/codedogapp/bakhyard/services/games/internal/exophase"
	"github.com/codedogapp/bakhyard/services/games/internal/refresh"
)

// exophasePlayerVar is the setting of the Exophase profile whose games the service lists
// (docs/setup.md). Exophase is off without it.
const exophasePlayerVar = "BAKHYARD_EXOPHASE_PLAYER"

// recentWindow is how far back GET /games/recent lists the games I played.
const recentWindow = 14 * 24 * time.Hour

// exophaseClient returns the client of the Exophase profile in the settings, which asks the API
// at address, or nil when the setting isn't set. The profile's id must be digits only.
func exophaseClient(getenv func(string) string, address *url.URL) (*exophase.Client, error) {
	player := getenv(exophasePlayerVar)
	if player == "" {
		return nil, nil
	}

	if strings.ContainsFunc(player, func(char rune) bool { return char < '0' || char > '9' }) {
		return nil, fmt.Errorf("%s: %q is not an Exophase profile id, which is digits only",
			exophasePlayerVar, player)
	}

	return exophase.New(address, player), nil
}

// poll returns what keeps the games I played in the last 14 days, which it lists with client
// whenever a request finds them stale, until ctx ends, with their covers from cache, or nil when
// Exophase is off and client is nil.
func poll(
	ctx context.Context,
	client *exophase.Client,
	cache *covers.Cache,
	logger *slog.Logger,
) *refresh.OnRequest[api.Recent] {
	if client == nil {
		return nil
	}

	recent := refresh.New("Exophase", func(ctx context.Context) (api.Recent, error) {
		games, err := client.PlayedSince(ctx, time.Now().Add(-recentWindow))
		if err != nil {
			return api.Recent{}, err
		}

		return api.Recent{Games: covered(ctx, games, cache, logger)}, nil
	}, logger)
	go recent.Run(ctx)

	return recent
}
