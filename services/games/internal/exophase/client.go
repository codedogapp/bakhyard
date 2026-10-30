// Package exophase lists the games that a player played lately, with their playtime and trophies,
// from the games list on Exophase's API: undocumented JSON with no login, which Exophase's own
// site reads (docs/plan.md §2 and §3, decisions 66, 67 and 301). It never asks for a profile's
// page, whose HTML holds personal data.
package exophase

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"
)

const (
	// userAgent names Bakhyard and links its repo. Cloudflare challenges Go's default one.
	userAgent = "bakhyard/0.1 (+https://github.com/codedogapp/bakhyard)"
	// maxPages bounds a listing, so that a server that never sends its last page can't keep the
	// client going.
	maxPages = 4
	// maxAnswer bounds the size of each answer from Exophase.
	maxAnswer = 4 << 20
	// requestTimeout bounds each call to Exophase.
	requestTimeout = 10 * time.Second
	// listing starts the errors of a listing of the player's games.
	listing = "list my Exophase games"
)

// errAnswer marks the error of an answer from Exophase that the client can't read.
var errAnswer = errors.New("read Exophase's answer")

// Client lists a player's games on Exophase.
type Client struct {
	api    *url.URL
	player string
	client *http.Client
}

// New returns the client of the player whose profile id is player, on Exophase's API at api,
// https://api.exophase.com. It reaches Exophase only when it lists.
func New(api *url.URL, player string) *Client {
	return &Client{api: api, player: player, client: &http.Client{Timeout: requestTimeout}}
}

// PlayedSince returns the games that the player last played at since or later, the last played
// first. Exophase lists the player's games the last played first, so the client reads no page
// after one that ends before since.
func (c *Client) PlayedSince(ctx context.Context, since time.Time) ([]Game, error) {
	games := []Game{}

	for number := 1; number <= maxPages; number++ {
		listed, err := c.page(ctx, number)
		if err != nil {
			return nil, err
		}

		for _, game := range listed {
			if !game.LastPlayed.Before(since) {
				games = append(games, game)
			}
		}

		if len(listed) == 0 || listed[len(listed)-1].LastPlayed.Before(since) {
			lastPlayedFirst(games)

			return games, nil
		}
	}

	return nil, fmt.Errorf("%s: more than %d pages", listing, maxPages)
}

// lastPlayedFirst sorts games the last played first. Games last played at the same time keep
// their order.
func lastPlayedFirst(games []Game) {
	slices.SortStableFunc(games, func(a, b Game) int { return b.LastPlayed.Compare(a.LastPlayed) })
}

// page returns the games on the page of the player's games list that number counts from 1. A
// page past the last has none.
func (c *Client) page(ctx context.Context, number int) ([]Game, error) {
	target := c.api.JoinPath("public/player", url.PathEscape(c.player), "games")
	// The query that plan §2 records: all platforms, the last played first, and no hidden games.
	target.RawQuery = "page=" + strconv.Itoa(number) + "&environment=&sort=1&showHidden=0"

	var found listPage

	err := c.get(ctx, target, &found)
	if err != nil {
		return nil, err
	}

	// Exophase says success false for a page past the last, and for the first page of a profile
	// that doesn't exist or isn't public.
	if !found.Success && number == 1 {
		return nil, fmt.Errorf("%s: Exophase has no public profile with this id", listing)
	}

	if !found.Success {
		return []Game{}, nil
	}

	games := make([]Game, 0, len(found.Games))
	for _, listed := range found.Games {
		games = append(games, listed.game())
	}

	return games, nil
}

// get asks Exophase for target and reads its answer, a 200, into answer. Its errors start with
// listing, or with errAnswer's "read Exophase's answer" when the answer is no use.
func (c *Client) get(ctx context.Context, target *url.URL, answer any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return fmt.Errorf("%s: %w", listing, err)
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	res, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", listing, err)
	}

	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return statusError(res)
	}

	err = json.UnmarshalRead(io.LimitReader(res.Body, maxAnswer), answer)
	if err != nil {
		return fmt.Errorf("%w: %w", errAnswer, err)
	}

	return nil
}

// statusError returns the error of an answer whose status isn't 200, which says so when it's
// Cloudflare's challenge, its bot check.
func statusError(res *http.Response) error {
	if res.Header.Get("Cf-Mitigated") == "challenge" {
		return fmt.Errorf("%s: %s, a Cloudflare challenge", listing, res.Status)
	}

	return fmt.Errorf("%s: %s", listing, res.Status)
}
