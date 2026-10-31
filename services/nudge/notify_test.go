package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/codedogapp/bakhyard/internal/notifier"
	"github.com/codedogapp/bakhyard/internal/requestid"
)

func TestNudgePushesWithTheRequestsID(t *testing.T) {
	const requestID = "01972f6e-8b2a-7c4d-9e1f-2a3b4c5d6e7f"

	asked := make(chan string, 1)
	notify := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked <- r.Method + " " + r.URL.Path + " " + r.Header.Get(requestid.Header)

		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(notify.Close)

	client := notifyClient(notify.Listener.Addr().String())

	// nudge pushes the question as it handles the agent's request, whose ID is requestID.
	handler := requestid.Middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		err := client.Send(r.Context(), notifier.Message{Title: "atlas", Body: "Merge?"})
		if err != nil {
			t.Error(err)
		}
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/agent/questions", nil)
	req.Header.Set(requestid.Header, requestID)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	select {
	case got := <-asked:
		if want := "POST /internal/send " + requestID; got != want {
			t.Errorf("notify got %q, want %q", got, want)
		}
	default:
		t.Error("notify got no push")
	}
}
