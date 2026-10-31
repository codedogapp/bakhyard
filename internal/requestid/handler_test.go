package requestid_test

import (
	"bytes"
	"encoding/json/v2"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/codedogapp/bakhyard/internal/requestid"
)

// logLine holds the keys of a JSON log line that the tests check.
type logLine struct {
	Level     string `json:"level"`
	Msg       string `json:"msg"`
	Peer      string `json:"peer"`
	RequestID string `json:"request_id"`
	Call      struct {
		RequestID string `json:"request_id"`
	} `json:"call"`
}

func TestHandlerNamesTheRequest(t *testing.T) {
	var out bytes.Buffer

	logger := slog.New(requestid.NewHandler(slog.NewJSONHandler(&out, nil)))
	handler := requestid.Middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		logger.With("peer", "todos").ErrorContext(r.Context(), "list the items")
		logger.WithGroup("call").InfoContext(r.Context(), "asked")
		logger.DebugContext(r.Context(), "below the level")
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.Header.Set(requestid.Header, canonical)
	handler.ServeHTTP(httptest.NewRecorder(), req)
	logger.InfoContext(t.Context(), "outside a request")

	want := []logLine{
		{Level: "ERROR", Msg: "list the items", Peer: "todos", RequestID: canonical},
		{Level: "INFO", Msg: "asked"},
		{Level: "INFO", Msg: "outside a request"},
	}
	want[1].Call.RequestID = canonical

	var got []logLine

	for text := range bytes.Lines(out.Bytes()) {
		var line logLine

		err := json.Unmarshal(text, &line)
		if err != nil {
			t.Fatalf("%q isn't a JSON line: %v", text, err)
		}

		got = append(got, line)
	}

	if !slices.Equal(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}
