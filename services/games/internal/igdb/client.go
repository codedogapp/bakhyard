// Package igdb finds the covers of games on IGDB, the Internet Game Database, with the app token
// that Twitch gives (docs/plan.md decisions 68 and 305).
package igdb

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"
)

const (
	// coverURL is the address of a cover's 264×374 JPEG, with the image's id for %s.
	coverURL = "https://images.igdb.com/igdb/image/upload/t_cover_big/%s.jpg"
	// filtered searches for a title, with the title for %s, among main games, expansions,
	// standalone expansions, remakes, remasters, expanded games and ports that aren't versions of
	// another game, so editions, DLC, bundles and packs don't come first.
	filtered = `search "%s"; fields name,cover.image_id; ` +
		`where game_type = (0,2,4,8,9,10,11) & version_parent = null; limit 1;`
	// named finds a game of any type named the title, with the title for %s, ignoring case.
	named = `fields name,cover.image_id; where name ~ "%s"; limit 1;`
)

// Tokens gives the app token that IGDB takes, and forgets one that IGDB refuses: *twitch.Tokens.
type Tokens interface {
	Token(ctx context.Context) (string, error)
	Drop(token string)
}

// Client finds covers on IGDB. It's safe for concurrent use.
type Client struct {
	endpoint string
	clientID string
	tokens   Tokens
	client   *http.Client
	// turn holds a value while a request is under way, so that requests go one at a time.
	turn chan struct{}
	// next is the earliest time at which the next request may start.
	next time.Time
}

// New returns the Client that asks IGDB's API at api, https://api.igdb.com, as the Twitch app
// with clientID, with the tokens that tokens gives.
func New(api *url.URL, clientID string, tokens Tokens) *Client {
	return &Client{
		endpoint: api.JoinPath("v4/games").String(),
		clientID: clientID,
		tokens:   tokens,
		client:   &http.Client{Timeout: requestTimeout},
		turn:     make(chan struct{}, 1),
	}
}

// Cover returns the address of the cover of the game named title, or "" when IGDB has no such
// game or no cover for it, or the title is empty.
func (c *Client) Cover(ctx context.Context, title string) (string, error) {
	if title == "" {
		return "", nil
	}

	found, err := c.match(ctx, title)
	if err != nil {
		return "", fmt.Errorf("find the cover of %q: %w", title, err)
	}

	if found.Cover.ImageID == "" {
		return "", nil
	}

	return fmt.Sprintf(coverURL, url.PathEscape(found.Cover.ImageID)), nil
}

// match returns the game that IGDB has for title, or no game. The filtered search's top result
// is the match when its name is the title, ignoring case. Otherwise a game named exactly the
// title, of any type, wins, since the top result may be another game, and the top result stays
// when there's no such game, since titles often differ from IGDB's names in subtitles or numerals.
func (c *Client) match(ctx context.Context, title string) (game, error) {
	quoted := quote(title)

	top, err := c.ask(ctx, fmt.Sprintf(filtered, quoted))
	if err != nil {
		return game{}, err
	}

	if len(top) > 0 && strings.EqualFold(top[0].Name, title) {
		return top[0], nil
	}

	same, err := c.ask(ctx, fmt.Sprintf(named, quoted))
	if err != nil {
		return game{}, err
	}

	if len(same) > 0 {
		return same[0], nil
	}

	if len(top) > 0 {
		return top[0], nil
	}

	return game{}, nil
}

// quote returns title as a query's string holds it: with a backslash before each backslash and
// quote, and without control characters.
func quote(title string) string {
	printable := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}

		return r
	}, title)

	return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(printable)
}
