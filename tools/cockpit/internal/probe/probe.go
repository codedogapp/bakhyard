// Package probe checks whether a service is up: whether it answers its health check,
// GET /healthz, with 200 OK (docs/plan.md §4.7).
package probe

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/codedogapp/bakhyard/internal/env"
)

// timeout is how long a service has to answer its health check (decision 60).
const timeout = time.Second

// Prober checks services. Its methods are safe to call at the same time.
type Prober struct {
	client *http.Client
}

// New returns a prober that opens a new connection for each check, so a service that stops
// taking connections fails its next check, and that follows no redirect and no proxy.
func New() *Prober {
	return &Prober{client: &http.Client{
		Transport: &http.Transport{DisableKeepAlives: true},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}}
}

// Check returns nil if the service that listens on listen, tcp://host:port, answers its health
// check with 200 OK within a second and before ctx ends, and why not otherwise. It doesn't read
// the answer's body.
func (p *Prober) Check(ctx context.Context, listen string) error {
	addr, err := env.Address(listen)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeoutCause(
		ctx, timeout, fmt.Errorf("no answer within %s", timeout),
	)
	defer cancel()

	target := (&url.URL{Scheme: "http", Host: addr, Path: "/healthz"}).String()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}

	// The body is never read, and the connection is never reused, so there is nothing to lose.
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s answered %s", target, resp.Status)
	}

	return nil
}
