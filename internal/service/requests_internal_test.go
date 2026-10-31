package service

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEachRequestHasAnIDThatItsLogsName(t *testing.T) {
	keepDefaultLogger(t)

	var out bytes.Buffer

	err := setLogs(logFormatEnv("json"), &out)
	if err != nil {
		t.Fatal(err)
	}

	routes := func(context.Context, func(string) string) (http.Handler, error) {
		logger := slog.Default()

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.ErrorContext(r.Context(), "list the items")
			w.WriteHeader(http.StatusInternalServerError)
		}), nil
	}

	handler, err := buildHandler(t.Context(), logFormatEnv("json"), routes)
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))

	var line struct {
		RequestID string `json:"request_id"`
	}

	err = json.Unmarshal(out.Bytes(), &line)
	if err != nil {
		t.Fatalf("%q isn't one JSON line: %v", out.Bytes(), err)
	}

	id := rec.Result().Header.Get("X-Request-Id")
	if id == "" || line.RequestID != id {
		t.Errorf("the answer's ID is %q and the line's %q, want one ID", id, line.RequestID)
	}
}
