// Package api serves notify's HTTP routes: the app's key and subscriptions, and the internal
// route that services send notifications through.
package api

import (
	"context"
	"encoding/json/v2"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/internal/jsonbody"
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

// Backends are what notify's endpoints run on.
type Backends struct {
	Devices Adder
	Sender  Sender
}

// Register serves notify on router. GET /notify/key answers the public key that a browser
// subscribes with, and POST /notify/subscriptions stores a browser's subscription. POST
// /internal/send, outside /notify so that the gateway never forwards to it, pushes a message to
// every device; only other services in the cluster reach it. Other methods get 405, and logger
// gets the errors that the database and the push run into.
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
}

// route is a POST endpoint that reads a T from the body and hands it to do: 204 when do
// succeeds, 400 when the body isn't a T or do answers invalid, and 500, with the cause in the
// log as what, when do fails otherwise.
type route[T any] struct {
	refusal string
	what    string
	invalid error
	do      func(context.Context, T) error
}

func (rt route[T]) handler(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body T
		if !jsonbody.Read(w, r, &body, rt.refusal) {
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
