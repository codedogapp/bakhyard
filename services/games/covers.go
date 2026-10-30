package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/codedogapp/bakhyard/internal/secrets"
	"github.com/codedogapp/bakhyard/services/games/internal/api"
	"github.com/codedogapp/bakhyard/services/games/internal/covers"
	"github.com/codedogapp/bakhyard/services/games/internal/exophase"
	"github.com/codedogapp/bakhyard/services/games/internal/igdb"
	"github.com/codedogapp/bakhyard/services/games/internal/twitch"
)

// twitchClientIDVar is the setting of the client ID of the Twitch app with which games asks IGDB
// for the games' covers (docs/setup.md). The games come without covers when it isn't set.
const twitchClientIDVar = "BAKHYARD_TWITCH_CLIENT_ID"

// twitchAccount is the Keychain account of the Twitch app's client secret, and the name of its
// file (decision 300).
const twitchAccount = "twitch-client-secret"

// coverCache returns the cache of the covers that IGDB finds for the Twitch app in the settings,
// which gets its tokens from Twitch with the secret that readSecret reads, or nil when the app
// isn't set.
func coverCache(
	ctx context.Context, getenv func(string) string, readSecret secrets.Reader, addresses apis,
) (*covers.Cache, error) {
	clientID := getenv(twitchClientIDVar)
	if clientID == "" {
		return nil, nil
	}

	secret, err := readSecret(ctx, twitchAccount)
	if err != nil {
		return nil, fmt.Errorf("read the Twitch client secret: %w", err)
	}

	tokens := twitch.New(addresses.twitch, clientID, secret)

	return covers.New(igdb.New(addresses.igdb, clientID, tokens)), nil
}

// covered returns games, each with its cover from cache, or with none when cache is nil or finds
// none. It logs to logger why cache found no more covers, unless ctx has ended.
func covered(
	ctx context.Context, games []exophase.Game, cache *covers.Cache, logger *slog.Logger,
) []api.Game {
	answered := make([]api.Game, len(games))
	titles := make([]string, len(games))

	for i, game := range games {
		answered[i].Game = game
		titles[i] = game.Title
	}

	if cache == nil {
		return answered
	}

	found, err := cache.Find(ctx, titles)
	if err != nil && ctx.Err() == nil {
		logger.ErrorContext(ctx, "find covers", "err", err)
	}

	for i, cover := range found {
		answered[i].Cover = cover
	}

	return answered
}
