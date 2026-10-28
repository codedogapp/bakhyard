// Package notifier asks notify to push a notification to the user's devices.
package notifier

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/url"
)

// Message is a notification: its title and body, and the path in the app that a tap opens.
type Message struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
}

// Client sends notifications through notify.
type Client struct {
	// Notify is the base URL of the notify service.
	Notify *url.URL
	HTTP   *http.Client
}

// Send asks notify to push message to every device.
func (c Client) Send(ctx context.Context, message Message) error {
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("write the message: %w", err)
	}

	endpoint := c.Notify.JoinPath("internal", "send").String()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("ask notify to push: %w", err)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("ask notify to push: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("ask notify to push: notify answered %s", resp.Status)
	}

	return nil
}
