// Package door is the client of nudge's agent routes: it asks the user a question, and keeps a
// session that reports how the work goes. Both go through the Bakhyard gateway, over the tailnet.
package door

import (
	"bytes"
	"cmp"
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

// defaultBackoff is how long a question waits before it asks again after a broken connection;
// each further failure doubles it, up to maxBackoff.
const (
	defaultBackoff = time.Second
	maxBackoff     = 30 * time.Second
	// maxFailures is how many requests in a row may fail quickly before Ask gives up: a service
	// that restarts is back well within them, one that is unreachable is not. A request that was
	// held for longer than heldLongEnough was connected, so its failure starts the count again.
	maxFailures    = 8
	heldLongEnough = 10 * time.Second
	// withdrawTimeout bounds the request that withdraws a question the agent gave up on.
	withdrawTimeout = 5 * time.Second
)

// Question is what an agent asks.
type Question struct {
	ID      string   `json:"id"`
	Project string   `json:"project"`
	Text    string   `json:"text"`
	Details string   `json:"details"`
	Options []string `json:"options"`
}

// Answer is the user's reply: one of the question's options, a text, or both.
type Answer struct {
	Option string `json:"option"`
	Text   string `json:"text"`
}

// ErrGone means the question was withdrawn or has expired, so no answer will come.
var ErrGone = errors.New("the question was withdrawn or has expired")

// Client reaches the agent routes of the gateway at Server.
type Client struct {
	// Server is the gateway's https URL.
	Server *url.URL
	// HTTP has no time limit: a question waits for as long as the user takes to answer.
	HTTP *http.Client
	// Backoff is the wait before the first retry; zero means a second.
	Backoff time.Duration
}

// New returns a client for the gateway at server, an https URL.
func New(server string) (*Client, error) {
	parsed, err := url.Parse(server)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return nil, fmt.Errorf("the server %q isn't an https URL", server)
	}

	return &Client{Server: parsed, HTTP: &http.Client{}}, nil
}

// Ask asks question, which must have an ID, and waits for its answer. The question's ID makes the
// request safe to repeat: when the connection breaks or the service restarts, Ask asks again
// with the same ID, and attaches to the question that waits already. When ctx ends, the question
// is withdrawn. A refusal becomes an error with the server's reason.
func (c *Client) Ask(ctx context.Context, question Question) (Answer, error) {
	body, err := json.Marshal(question)
	if err != nil {
		return Answer{}, fmt.Errorf("write the question: %w", err)
	}

	delay := cmp.Or(c.Backoff, defaultBackoff)
	failures := 0

	for {
		started := time.Now()

		answer, retry, err := c.tryAsk(ctx, body)
		if !retry {
			return answer, err
		}

		failures++
		if time.Since(started) >= heldLongEnough {
			failures, delay = 1, cmp.Or(c.Backoff, defaultBackoff)
		}

		if ctx.Err() != nil {
			c.withdraw(question.ID)

			return Answer{}, fmt.Errorf("wait for the answer: %w", context.Cause(ctx))
		}

		if failures >= maxFailures {
			return Answer{}, fmt.Errorf("giving up after %d failed attempts: %w", failures, err)
		}

		if pause(ctx, delay) {
			delay = min(delay*2, maxBackoff)
		}
	}
}

// tryAsk makes one request for the answer, and reports whether to ask again: the connection broke
// or the service answered that it can't serve now.
func (c *Client) tryAsk(ctx context.Context, body []byte) (Answer, bool, error) {
	resp, err := c.do(ctx, http.MethodPost, "/agent/questions", body)
	if err != nil {
		return Answer{}, true, err
	}

	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusOK:
		var answer Answer
		if err := json.UnmarshalRead(resp.Body, &answer); err != nil {
			// The connection broke while the answer came; the answer is kept, so ask again.
			return Answer{}, true, fmt.Errorf("read the answer: %w", err)
		}

		return answer, false, nil
	case http.StatusGone:
		return Answer{}, false, ErrGone
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return Answer{}, true, refusal(resp)
	default:
		return Answer{}, false, refusal(resp)
	}
}

// pause waits for delay, and reports whether it did: it doesn't when ctx ends first.
func pause(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// withdraw tells the service that the agent no longer waits for the question, as well as it can:
// ctx has ended, so the request has a time limit of its own.
func (c *Client) withdraw(id string) {
	ctx, cancel := context.WithTimeout(context.Background(), withdrawTimeout)
	defer cancel()

	resp, err := c.do(ctx, http.MethodDelete, "/agent/questions/"+id, nil)
	if err == nil {
		_ = resp.Body.Close()
	}
}

// do sends a request to path on the server, with body as JSON when it has one.
func (c *Client) do(
	ctx context.Context, method, path string, body []byte,
) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, method, c.Server.JoinPath(path).String(),
		bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("make the request: %w", err)
	}

	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTP.Do(request)
	if err != nil {
		return nil, fmt.Errorf("reach nudge: %w", err)
	}

	return resp, nil
}

// maxReason caps how much of a refusal's body the error carries.
const maxReason = 1 << 10

// refusal turns a reply that isn't a success into an error with the server's reason.
func refusal(resp *http.Response) error {
	reason, _ := io.ReadAll(io.LimitReader(resp.Body, maxReason))

	return errors.New(cmp.Or(strings.TrimSpace(string(reason)), resp.Status))
}
