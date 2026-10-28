// Package push sends a notification to every subscribed device as a Web Push.
package push

import (
	"cmp"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/SherClockHolmes/webpush-go"

	"github.com/codedogapp/bakhyard/services/notify/internal/subscription"
	"github.com/codedogapp/bakhyard/services/notify/internal/vapid"
)

const (
	// subscriber says who sends the pushes, as the VAPID JWT's subject. Apple wants a URL or a
	// mailto: address.
	subscriber = "https://github.com/codedogapp/bakhyard"
	// ttl is how many seconds a push service keeps a push for a device that's offline: a day.
	ttl = 24 * 60 * 60
	// maxTitle and maxBody, in characters, keep the payload in one 4,096-byte record, even when
	// every character is escaped as \u00XX.
	maxTitle = 100
	maxBody  = 500
	maxURL   = 200
	// maxReason is how much of a push service's answer goes to the log.
	maxReason = 512
	// defaultTitle titles a notification that has none.
	defaultTitle = "Bakhyard"
)

// ErrBadURL means a message's URL isn't a path in the app.
var ErrBadURL = errors.New("the url must be a path in the app, such as /ask")

// Message is a notification, as the service worker reads it: its title and body, and the path in
// the app that a tap opens, "/" when it has none.
type Message struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url,omitempty"`
}

// Sender sends pushes to the subscribed devices, signed with the VAPID key.
type Sender struct {
	Key     vapid.Key
	Devices *subscription.Store
	Client  *http.Client
	Logger  *slog.Logger
}

// Send sends message to each subscribed device in turn, and forgets a device whose push service
// says its subscription is gone. A device that can't be reached or refuses is logged, and the
// others still get theirs; push is best effort. It fails only when the message is bad or the
// devices can't be listed.
func (s *Sender) Send(ctx context.Context, message Message) error {
	payload, err := encode(message)
	if err != nil {
		return err
	}

	devices, err := s.Devices.List(ctx)
	if err != nil {
		return err
	}

	for _, device := range devices {
		s.sendTo(ctx, device, payload)
	}

	return nil
}

// encode checks message and writes it as the push's payload.
func encode(message Message) ([]byte, error) {
	if message.URL != "" && (!strings.HasPrefix(message.URL, "/") ||
		strings.HasPrefix(message.URL, "//") || utf8.RuneCountInString(message.URL) > maxURL) {
		return nil, ErrBadURL
	}

	payload, err := json.Marshal(Message{
		Title: cut(cmp.Or(message.Title, defaultTitle), maxTitle),
		Body:  cut(message.Body, maxBody),
		URL:   message.URL,
	})
	if err != nil {
		return nil, fmt.Errorf("write the push: %w", err)
	}

	return payload, nil
}

// sendTo sends payload to device. The log names its push service, never its endpoint: that's the
// device's address.
func (s *Sender) sendTo(ctx context.Context, device subscription.Subscription, payload []byte) {
	service := ""
	if endpoint, err := url.Parse(device.Endpoint); err == nil {
		service = endpoint.Host
	}

	resp, err := webpush.SendNotificationWithContext(ctx, payload, &webpush.Subscription{
		Endpoint: device.Endpoint,
		Keys:     webpush.Keys{Auth: device.Keys.Auth, P256dh: device.Keys.P256dh},
	}, &webpush.Options{
		HTTPClient:      s.Client,
		Subscriber:      subscriber,
		TTL:             ttl,
		Urgency:         webpush.UrgencyHigh,
		VAPIDPublicKey:  s.Key.Public,
		VAPIDPrivateKey: s.Key.Private,
	})
	if err != nil {
		s.Logger.WarnContext(ctx, "push failed", "service", service, "err", withoutURL(err))
		return
	}
	defer func() { _ = resp.Body.Close() }()

	switch code := resp.StatusCode; {
	case code == http.StatusNotFound || code == http.StatusGone:
		s.forget(ctx, device.Endpoint, service, code)
	case code >= http.StatusMultipleChoices:
		reason, _ := io.ReadAll(io.LimitReader(resp.Body, maxReason))
		s.Logger.WarnContext(ctx, "push refused",
			"service", service, "status", code, "reason", string(reason))
	}
}

// forget removes the device with endpoint, whose subscription is gone.
func (s *Sender) forget(ctx context.Context, endpoint, service string, status int) {
	if err := s.Devices.Remove(ctx, endpoint); err != nil {
		s.Logger.WarnContext(ctx, "couldn't forget a device", "service", service, "err", err)
		return
	}

	s.Logger.InfoContext(ctx, "forgot a device", "service", service, "status", status)
}

// cut shortens text to limit characters, the last of them "…", when it's longer.
func cut(text string, limit int) string {
	if utf8.RuneCountInString(text) <= limit {
		return text
	}

	return string([]rune(text)[:limit-1]) + "…"
}

// withoutURL drops the endpoint from the error of a request that failed.
func withoutURL(err error) error {
	if failed, ok := errors.AsType[*url.Error](err); ok {
		return failed.Err
	}

	return err
}
