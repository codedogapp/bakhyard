package notifier_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/codedogapp/bakhyard/services/nudge/internal/notifier"
)

func TestSendPostsTheMessageToNotify(t *testing.T) {
	var path, body string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		path, body = r.Method+" "+r.URL.Path, string(raw)

		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	address, _ := url.Parse(server.URL)
	client := notifier.Client{Notify: address, HTTP: server.Client()}

	err := client.Send(t.Context(), notifier.Message{Title: "atlas", Body: "Merge?", URL: "/nudge"})
	if want := `{"title":"atlas","body":"Merge?","url":"/nudge"}`; err != nil ||
		path != "POST /internal/send" || body != want {
		t.Errorf("got %q %q (%v), want POST /internal/send %s", path, body, err, want)
	}
}

func TestSendFailsWhenNotifyRefusesOrIsGone(t *testing.T) {
	refusing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "no", http.StatusBadRequest)
	}))
	address, _ := url.Parse(refusing.URL)
	client := notifier.Client{Notify: address, HTTP: refusing.Client()}

	if err := client.Send(t.Context(), notifier.Message{}); err == nil {
		t.Error("Send succeeded against a notify that answers 400")
	}

	refusing.Close()

	if err := client.Send(t.Context(), notifier.Message{}); err == nil {
		t.Error("Send succeeded against a notify that is gone")
	}
}
