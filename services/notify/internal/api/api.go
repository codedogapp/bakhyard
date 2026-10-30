// Package api serves notify's HTTP routes: the app's key, subscriptions and alerts, and the
// internal routes that services and Alertmanager send notifications through.
package api

import (
	"context"
	"encoding/json/v2"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/internal/jsonbody"
	"github.com/codedogapp/bakhyard/services/notify/internal/alert"
	"github.com/codedogapp/bakhyard/services/notify/internal/history"
	"github.com/codedogapp/bakhyard/services/notify/internal/push"
	"github.com/codedogapp/bakhyard/services/notify/internal/subscription"
)

// Adder keeps a device's subscription.
type Adder interface {
	Add(ctx context.Context, sub subscription.Subscription) error
}

// Sender sends a message to every device.
type Sender interface {
	Send(ctx context.Context, message push.Message) error
}

// History keeps the alerts that Alertmanager posts, and lists them.
type History interface {
	Keep(ctx context.Context, alerts []alert.Alert) error
	List(ctx context.Context) ([]history.Entry, error)
}

// Backends are what notify's endpoints run on.
type Backends struct {
	Devices Adder
	Sender  Sender
	Alerts  History
}

// Register serves notify on router. GET /notify/key answers the public key that a browser
// subscribes with, POST /notify/subscriptions stores a browser's subscription, and GET
// /notify/alerts answers the alerts that fire and those that resolved in the last 30 days. POST
// /internal/send, outside /notify so that the gateway never forwards to it, pushes a message to
// every device; only other services in the cluster reach it. POST /internal/alerts, outside
// /notify too, takes the webhook of the cluster's Alertmanager, which alone reaches it, keeps its
// alerts and then pushes the group to every device. Other methods get 405, and logger gets the
// errors that the database and the push run into.
func Register(router chi.Router, publicKey string, backends Backends, logger *slog.Logger) {
	router.Get("/notify/key", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.MarshalWrite(w, struct {
			Key string `json:"key"`
		}{publicKey})
	})
	router.Post("/notify/subscriptions", route[subscription.Subscription]{
		refusal: "the body isn't a push subscription",
		what:    "store the subscription",
		invalid: subscription.ErrInvalid,
		do:      backends.Devices.Add,
	}.handler(logger))
	router.Post("/internal/send", route[push.Message]{
		refusal: "the body isn't a message with a title, a body and a url",
		what:    "send the message",
		invalid: push.ErrBadURL,
		do:      backends.Sender.Send,
	}.handler(logger))
	router.Post("/internal/alerts", route[alert.Group]{
		refusal: "the body isn't Alertmanager's webhook",
		what:    "send the alerts",
		invalid: alert.ErrInvalid,
		lenient: true,
		do:      backends.alerts,
	}.handler(logger))
	router.Get("/notify/alerts", listAlerts(backends.Alerts, logger))
}

// listAlerts returns the handler of GET /notify/alerts. When the alerts can't be read, the client
// gets a 500 and logger gets the cause.
func listAlerts(alerts History, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entries, err := alerts.List(r.Context())
		if err != nil {
			logger.ErrorContext(r.Context(), "list the alerts", "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		// The alerts hold text from the UTF8 database alone, so they always encode: an error
		// means the client has gone.
		_ = json.MarshalWrite(w, struct {
			Alerts []history.Entry `json:"alerts"`
		}{entries})
	}
}

// alerts keeps group's alerts and then pushes the group. A group that isn't one of alerts that
// fire or resolved is neither kept nor pushed, nor is one that can't be kept.
func (b Backends) alerts(ctx context.Context, group alert.Group) error {
	message, err := group.Message()
	if err != nil {
		return err
	}

	if err := b.Alerts.Keep(ctx, group.Alerts); err != nil {
		return err
	}

	return b.Sender.Send(ctx, message)
}

// route is a POST endpoint that reads a T from the body and hands it to do: 204 when do
// succeeds, 400 when the body isn't a T or do answers invalid, and 500, with the cause in the
// log as what, when do fails otherwise. A lenient route skips the body's members that T lacks,
// which the others refuse.
type route[T any] struct {
	refusal string
	what    string
	invalid error
	lenient bool
	do      func(context.Context, T) error
}

func (rt route[T]) handler(logger *slog.Logger) http.HandlerFunc {
	read := jsonbody.Read
	if rt.lenient {
		read = jsonbody.ReadLenient
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var body T
		if !read(w, r, &body, rt.refusal) {
			return
		}

		err := rt.do(r.Context(), body)
		if errors.Is(err, rt.invalid) {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		if err != nil {
			logger.ErrorContext(r.Context(), rt.what, "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
