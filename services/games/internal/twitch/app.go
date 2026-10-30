package twitch

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// maxAnswer bounds the size of each answer from Twitch: many times what a token's needs.
	maxAnswer = 64 << 10
	// requestTimeout bounds each request for a token.
	requestTimeout = 10 * time.Second
	// getting starts the errors of a request for a token.
	getting = "get a Twitch token"
)

// errAnswer marks the error of an answer from Twitch that holds no token that Tokens can keep.
var errAnswer = errors.New("read Twitch's answer")

// app asks Twitch for the tokens of the app whose client ID and secret it holds.
type app struct {
	api      *url.URL
	clientID string
	secret   string
	client   *http.Client
}

// New returns the Tokens of the Twitch app with clientID and secret, which ask Twitch's OAuth
// server at api, https://id.twitch.tv, only when they need a token.
func New(api *url.URL, clientID, secret string) *Tokens {
	return keep(newApp(api, clientID, secret).token)
}

// newApp returns the app with clientID and secret, which asks Twitch's OAuth server at api.
func newApp(api *url.URL, clientID, secret string) *app {
	return &app{
		api: api, clientID: clientID, secret: secret,
		client: &http.Client{Timeout: requestTimeout},
	}
}

// token asks Twitch for a new app token with the client credentials grant, whose credentials go
// in the form body, as Twitch's docs show, rather than in the URL, where errors would show them.
func (a *app) token(ctx context.Context) (string, time.Duration, error) {
	form := url.Values{
		"client_id":     {a.clientID},
		"client_secret": {a.secret},
		"grant_type":    {"client_credentials"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		a.api.JoinPath("oauth2/token").String(), strings.NewReader(form.Encode()))
	if err != nil {
		return "", 0, fmt.Errorf("%s: %w", getting, err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	res, err := a.client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("%s: %w", getting, err)
	}

	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return "", 0, refusal(res)
	}

	return issued(res.Body)
}

// issued returns the token in body, Twitch's answer to a request that it granted, with the
// lifetime that the answer gives it.
func issued(body io.Reader) (string, time.Duration, error) {
	var answer struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}

	err := json.UnmarshalRead(io.LimitReader(body, maxAnswer), &answer)
	if err != nil {
		return "", 0, fmt.Errorf("%w: %w", errAnswer, err)
	}

	if answer.AccessToken == "" {
		return "", 0, fmt.Errorf("%w: no access token", errAnswer)
	}

	if answer.ExpiresIn <= 0 {
		return "", 0, fmt.Errorf("%w: the token expires in %d seconds", errAnswer, answer.ExpiresIn)
	}

	return answer.AccessToken, time.Duration(answer.ExpiresIn) * time.Second, nil
}

// refusal returns the error of an answer whose status isn't 200, with the reason that Twitch
// gives, such as "invalid client", when it gives one.
func refusal(res *http.Response) error {
	var answer struct {
		Message string `json:"message"`
	}

	err := json.UnmarshalRead(io.LimitReader(res.Body, maxAnswer), &answer)
	if err != nil || answer.Message == "" {
		return fmt.Errorf("%s: %s", getting, res.Status)
	}

	return fmt.Errorf("%s: %s, %s", getting, res.Status, answer.Message)
}
