package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/codedogapp/bakhyard/internal/notifier"
	"github.com/codedogapp/bakhyard/services/pulls/internal/bitbucket"
)

func TestReminderReadsBitbucketAndPushesThroughNotify(t *testing.T) {
	address, _ := bitbucketServer(t)
	client := bitbucketClientAt(t, address)
	mine := bitbucket.NewPoller(client.Mine, slog.Default())

	sent := make(chan string, 1)
	notifyServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			sent <- r.Method + " " + r.URL.Path + " " + string(body)

			w.WriteHeader(http.StatusNoContent)
		}),
	)
	t.Cleanup(notifyServer.Close)

	notify, err := notifyClient(envOf(map[string]string{
		"BAKHYARD_PEERS": "notify=tcp://" + notifyServer.Listener.Addr().String(),
	}))
	if err != nil {
		t.Fatal(err)
	}

	reminder := newReminder(client, mine, notify)
	if reminder.Mine != mine || reminder.Location != time.Local ||
		reminder.Logger != slog.Default() {
		t.Errorf(
			"got the reminder %+v, want one of mine, in the host's zone, with the default logs",
			reminder,
		)
	}

	approvals, err := reminder.Approvals(t.Context(), bitbucket.Pull{ID: 1, Repo: "acme/web"})
	if want := (bitbucket.Approvals{Given: 0, Required: 1}); err != nil || approvals != want {
		t.Errorf("got the approvals %+v, %v, want %+v", approvals, err, want)
	}

	message := notifier.Message{Title: "Remind your reviewers", Body: "Ship", URL: "/pulls"}

	err = reminder.Pusher.Send(t.Context(), message)
	if err != nil {
		t.Fatal(err)
	}

	want := `POST /internal/send {"title":"Remind your reviewers","body":"Ship","url":"/pulls"}`
	if got := <-sent; got != want {
		t.Errorf("notify got %q, want %q", got, want)
	}
}

func TestReminderIsOffWithoutBitbucketOrNotify(t *testing.T) {
	client := bitbucketClientAt(t, bitbucketAPI)
	mine := bitbucket.NewPoller(client.Mine, slog.Default())

	if reminder := newReminder(nil, nil, &notifier.Client{}); reminder != nil {
		t.Errorf("got %+v without Bitbucket, want no reminder", reminder)
	}

	if reminder := newReminder(client, mine, nil); reminder != nil {
		t.Errorf("got %+v without notify, want no reminder", reminder)
	}
}

func TestNotifyClientIsOffWithoutTheNotifyPeer(t *testing.T) {
	for _, peers := range []string{"", "todos=tcp://127.0.0.1:8084"} {
		notify, err := notifyClient(envOf(map[string]string{"BAKHYARD_PEERS": peers}))
		if notify != nil || err != nil {
			t.Errorf("%q: got %+v, %v, want no client and no error", peers, notify, err)
		}
	}
}

func TestPullsStopsOnABadPeersSetting(t *testing.T) {
	// No token: the peers are checked before it is read.
	settings := bitbucketSettings(bitbucketAPI)
	settings["BAKHYARD_PEERS"] = "notify"

	_, err := routes(t.Context(), envOf(settings), secretsWith(nil))

	want := `BAKHYARD_PEERS: "notify" is not name=tcp://host:port`
	if err == nil || err.Error() != want {
		t.Errorf("got %v, want the error %s", err, want)
	}
}

// bitbucketClientAt returns the client of the Bitbucket API at address, in the settings that
// bitbucketSettings returns, with the token secret.
func bitbucketClientAt(t *testing.T, address string) *bitbucket.Client {
	t.Helper()

	client, err := bitbucketClient(envOf(bitbucketSettings(address)),
		secretsWith(map[string]string{"bitbucket-token": "secret"}))
	if err != nil {
		t.Fatal(err)
	}

	return client
}
