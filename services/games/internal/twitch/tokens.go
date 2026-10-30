// Package twitch gets the app access token that IGDB's API takes from Twitch, keeps it, and gets
// a new one before it expires or when IGDB refuses it (docs/plan.md decisions 68, 300 and 304).
package twitch

import (
	"context"
	"time"
)

// renewShare is the share of a token's life after which Tokens gets a new one.
const renewShare = 0.9

// getter asks Twitch for a new app token and returns it with its lifetime.
type getter func(ctx context.Context) (string, time.Duration, error)

// Tokens keeps the latest app token of a Twitch app. It's safe for concurrent use.
type Tokens struct {
	get getter
	// turn holds a value while a call reads or changes the token, so that calls take turns and
	// only one at a time asks Twitch.
	turn  chan struct{}
	token string
	renew time.Time
}

// keep returns the Tokens that get their tokens from get.
func keep(get getter) *Tokens {
	return &Tokens{get: get, turn: make(chan struct{}, 1)}
}

// Token returns the token that Tokens keeps, or gets a new one when none is kept or the kept one
// has run 9 tenths of its life, counted from when the request for it began. One call at a time
// asks Twitch, and the others wait for its answer, since Twitch keeps only an app's 25 newest
// tokens. A failed request changes nothing, so the next call asks again. If ctx ends while the
// call waits, it returns ctx's error.
func (t *Tokens) Token(ctx context.Context) (string, error) {
	select {
	case t.turn <- struct{}{}:
	case <-ctx.Done():
		return "", ctx.Err()
	}

	defer func() { <-t.turn }()

	if t.token != "" && time.Now().Before(t.renew) {
		return t.token, nil
	}

	began := time.Now()

	token, life, err := t.get(ctx)
	if err != nil {
		return "", err
	}

	t.token = token
	t.renew = began.Add(time.Duration(float64(life) * renewShare))

	return token, nil
}

// Drop forgets token when IGDB has refused it, so the next call of Token gets a new one. A token
// that Tokens no longer keeps changes nothing, so a refusal of an older token keeps the newer.
func (t *Tokens) Drop(token string) {
	t.turn <- struct{}{}

	defer func() { <-t.turn }()

	if token == t.token {
		t.token = ""
	}
}
