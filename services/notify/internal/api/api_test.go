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

	"github.com/codedogapp/bakhyard/services/notify/internal/alert"
	"github.com/codedogapp/bakhyard/services/notify/internal/api"
	"github.com/codedogapp/bakhyard/services/notify/internal/push"
	"github.com/codedogapp/bakhyard/services/notify/internal/subscription"
)

// webhook is a firing group of one alert, as Alertmanager posts it, with members that notify
// doesn't read.
const webhook = `{
	"version": "4",
	"groupKey": "{}:{alertname=\"DiskFull\"}",
	"truncatedAlerts": 0,
	"status": "firing",
	"receiver": "bakhyard",
	"groupLabels": {"alertname": "DiskFull"},
	"commonLabels": {"alertname": "DiskFull", "severity": "warning"},
	"commonAnnotations": {"summary": "A filesystem is nearly full"},
	"externalURL": "http://alertmanager:9093",
	"notification_reason": "first notification",
	"alerts": [{
		"status": "firing",
		"labels": {"alertname": "DiskFull", "mountpoint": "/"},
		"annotations": {"summary": "A filesystem is nearly full", "description": "/ is 91% full"},
		"startsAt": "2026-10-30T12:00:00Z",
		"endsAt": "0001-01-01T00:00:00Z",
		"generatorURL": "http://vmalert:8880/vmalert/alert?group_id=1&alert_id=2",
		"fingerprint": "c4f6a1b2d3e4f5a6"
	}]
}`

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
	t *testing.T, devices api.Adder, sender api.Sender,
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
			"subscribe as a browser sends it",
			"/notify/subscriptions",
			`{"endpoint":"e","expirationTime":null,"keys":{"p256dh":"p","auth":"a"}}`,
			nil, nil, http.StatusNoContent, "",
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
			"send unknown member", "/internal/send", `{"title":"t","body":"b","x":1}`,
			nil, nil, http.StatusBadRequest, "",
		},
		{
			"send database", "/internal/send", `{"body":"b"}`,
			nil, io.ErrClosedPipe, http.StatusInternalServerError, "send the message",
		},
		{"alerts", "/internal/alerts", webhook, nil, nil, http.StatusNoContent, ""},
		{
			"alerts without a status", "/internal/alerts", `{"version":"4","alerts":[{}]}`,
			nil, nil, http.StatusBadRequest, "",
		},
		{"alerts garbage", "/internal/alerts", `[]`, nil, nil, http.StatusBadRequest, ""},
		{
			"alerts too large",
			"/internal/alerts",
			`{"receiver":"` + strings.Repeat("a", 70000) + `"}`,
			nil,
			nil,
			http.StatusRequestEntityTooLarge,
			"",
		},
		{
			"alerts database", "/internal/alerts", webhook,
			nil, io.ErrClosedPipe, http.StatusInternalServerError, "send the alerts",
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

func TestAlertsPushTheGroup(t *testing.T) {
	sender := &fakeSender{}
	router, _ := serve(t, fakeDevices{}, sender)

	do(router, http.MethodPost, "/internal/alerts", webhook)

	want := push.Message{Title: "A filesystem is nearly full", Body: "/ is 91% full", URL: "/"}
	if len(sender.sent) != 1 || sender.sent[0] != want {
		t.Errorf("sent %+v, want %+v", sender.sent, want)
	}
}

func TestAlertsSayWhatAGroupNeeds(t *testing.T) {
	sender := &fakeSender{}
	router, _ := serve(t, fakeDevices{}, sender)

	rec := do(router, http.MethodPost, "/internal/alerts", `{"status":"pending","alerts":[{}]}`)
	if rec.Code != http.StatusBadRequest || rec.Body.String() != alert.ErrInvalid.Error()+"\n" ||
		len(sender.sent) != 0 {
		t.Errorf("got %d %q and sent %+v, want a 400 that says what's missing and no push",
			rec.Code, rec.Body, sender.sent)
	}
}
