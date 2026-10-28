package api_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/codedogapp/bakhyard/services/notify/internal/api"
	"github.com/codedogapp/bakhyard/services/notify/internal/push"
	"github.com/codedogapp/bakhyard/services/notify/internal/subscription"
)

type fakeDevices struct{ err error }

func (f fakeDevices) Add(context.Context, subscription.Subscription) error { return f.err }

type fakeSender struct {
	err  error
	sent []push.Message
}

func (f *fakeSender) Send(_ context.Context, message push.Message) error {
	f.sent = append(f.sent, message)

	return f.err
}

func serve(
	t *testing.T, devices api.Subscriptions, sender api.Sender,
) (http.Handler, *bytes.Buffer) {
	t.Helper()

	logs := &bytes.Buffer{}
	router := chi.NewRouter()
	api.Register(
		router,
		"PUBLIC",
		api.Backends{Devices: devices, Sender: sender},
		slog.New(slog.NewTextHandler(logs, nil)),
	)

	return router, logs
}

func do(router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))

	return rec
}

func TestKeyAnswersThePublicKey(t *testing.T) {
	router, _ := serve(t, fakeDevices{}, &fakeSender{})

	rec := do(router, http.MethodGet, "/notify/key", "")
	if rec.Code != http.StatusOK || rec.Body.String() != `{"key":"PUBLIC"}` {
		t.Errorf("got %d %s", rec.Code, rec.Body)
	}
}

func TestPostsAnswerByTheirOutcome(t *testing.T) {
	tests := []struct {
		name, path, body string
		devices          error
		sender           error
		want             int
		logged           string
	}{
		{
			"subscribe",
			"/notify/subscriptions",
			`{"endpoint":"e","keys":{"p256dh":"p","auth":"a"}}`,
			nil,
			nil,
			http.StatusNoContent,
			"",
		},
		{
			"subscribe invalid", "/notify/subscriptions", `{"endpoint":"e"}`,
			subscription.ErrInvalid, nil, http.StatusBadRequest, "",
		},
		{
			"subscribe unknown member", "/notify/subscriptions", `{"x":1}`,
			nil, nil, http.StatusBadRequest, "",
		},
		{
			"subscribe database", "/notify/subscriptions", `{"endpoint":"e"}`,
			errors.New("down"), nil, http.StatusInternalServerError, "store the subscription",
		},
		{
			"subscribe too large",
			"/notify/subscriptions",
			`{"endpoint":"` + strings.Repeat("a", 70000) + `"}`,
			nil,
			nil,
			http.StatusRequestEntityTooLarge,
			"",
		},
		{
			"send", "/internal/send", `{"title":"t","body":"b","url":"/ask"}`,
			nil, nil, http.StatusNoContent, "",
		},
		{
			"send bad url", "/internal/send", `{"body":"b","url":"//x"}`,
			nil, push.ErrBadURL, http.StatusBadRequest, "",
		},
		{"send garbage", "/internal/send", `nope`, nil, nil, http.StatusBadRequest, ""},
		{
			"send database", "/internal/send", `{"body":"b"}`,
			nil, io.ErrClosedPipe, http.StatusInternalServerError, "send the message",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router, logs := serve(t, fakeDevices{test.devices}, &fakeSender{err: test.sender})

			rec := do(router, http.MethodPost, test.path, test.body)
			if rec.Code != test.want || !strings.Contains(logs.String(), test.logged) {
				t.Errorf(
					"got %d (%s) with the log %q, want %d",
					rec.Code,
					rec.Body,
					logs,
					test.want,
				)
			}
		})
	}
}

func TestSendPassesTheMessageOn(t *testing.T) {
	sender := &fakeSender{}
	router, _ := serve(t, fakeDevices{}, sender)

	do(router, http.MethodPost, "/internal/send", `{"title":"t","body":"b","url":"/ask"}`)

	if want := (push.Message{Title: "t", Body: "b", URL: "/ask"}); len(sender.sent) != 1 ||
		sender.sent[0] != want {
		t.Errorf("sent %+v, want %+v", sender.sent, want)
	}
}
