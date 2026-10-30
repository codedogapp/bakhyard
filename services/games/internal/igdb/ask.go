package igdb

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// gap is the least time between the starts of two requests, so IGDB gets 4 a second at most.
	gap = 250 * time.Millisecond
	// maxAnswer bounds the size of each answer from IGDB: many times what a game's needs.
	maxAnswer = 64 << 10
	// requestTimeout bounds each request to IGDB.
	requestTimeout = 10 * time.Second
	// asking starts the errors of a request that IGDB didn't answer with games.
	asking = "ask IGDB"
)

// errRefused is the error of a request whose token IGDB refused.
var errRefused = errors.New(asking + ": 401 Unauthorized")

// game is a game in IGDB's answer, with the fields that the queries ask for.
type game struct {
	Name  string `json:"name"`
	Cover struct {
		ImageID string `json:"image_id"`
	} `json:"cover"`
}

// ask returns the games that IGDB answers to query. When IGDB refuses the token, which it may
// have revoked, ask asks once more with a new one.
func (c *Client) ask(ctx context.Context, query string) ([]game, error) {
	games, err := c.askWithToken(ctx, query)
	if errors.Is(err, errRefused) {
		return c.askWithToken(ctx, query)
	}

	return games, err
}

// askWithToken asks query with the token that the Tokens give, and drops it when IGDB refuses it.
func (c *Client) askWithToken(ctx context.Context, query string) ([]game, error) {
	token, err := c.tokens.Token(ctx)
	if err != nil {
		return nil, err
	}

	games, err := c.send(ctx, token, query)
	if errors.Is(err, errRefused) {
		c.tokens.Drop(token)
	}

	return games, err
}

// send sends query with token in its turn, once the previous request started gap ago. If ctx
// ends while it waits, it returns ctx's error.
func (c *Client) send(ctx context.Context, token, query string) ([]game, error) {
	select {
	case c.turn <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	defer func() { <-c.turn }()

	pause := time.NewTimer(time.Until(c.next))
	defer pause.Stop()

	select {
	case <-pause.C:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	c.next = time.Now().Add(gap)

	return c.post(ctx, token, query)
}

// post posts query to IGDB's games endpoint with token, and returns the games it answers.
func (c *Client) post(ctx context.Context, token, query string) ([]game, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint,
		strings.NewReader(query))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", asking, err)
	}

	req.Header.Set("Client-ID", c.clientID)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")

	res, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", asking, err)
	}

	defer func() { _ = res.Body.Close() }()

	switch res.StatusCode {
	case http.StatusOK:
		return answered(res.Body)
	case http.StatusUnauthorized:
		return nil, errRefused
	default:
		return nil, fmt.Errorf("%s: %s", asking, res.Status)
	}
}

// answered returns the games in body, IGDB's answer to a query.
func answered(body io.Reader) ([]game, error) {
	var games []game

	err := json.UnmarshalRead(io.LimitReader(body, maxAnswer), &games)
	if err != nil {
		return nil, fmt.Errorf("read IGDB's answer: %w", err)
	}

	return games, nil
}
