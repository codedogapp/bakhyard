package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"

	"github.com/codedogapp/bakhyard/internal/databasetest"
	attentionv1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1"
	"github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1/attentionv1connect"
	"github.com/codedogapp/bakhyard/internal/secrets"
	"github.com/codedogapp/bakhyard/services/notify/internal/vapid"
)

// group is a group of two alerts, as Alertmanager posts it: DiskFull fires, and BatteryLow, which
// fired from %[1]s, resolved at %[2]s.
const group = `{
	"status": "firing",
	"alerts": [{
		"status": "firing",
		"labels": {"alertname": "DiskFull"},
		"annotations": {"summary": "A filesystem is nearly full", "description": "/ has 4 GiB left"},
		"startsAt": "2026-10-30T12:00:00.123456Z",
		"endsAt": "0001-01-01T00:00:00Z",
		"fingerprint": "c4f6a1b2d3e4f5a6"
	}, {
		"status": "resolved",
		"labels": {"alertname": "BatteryLow"},
		"annotations": {"summary": "The battery is low"},
		"startsAt": "%[1]s",
		"endsAt": "%[2]s",
		"fingerprint": "b1a2c3d4e5f6a7b8"
	}]
}`

func TestNotifyListsTheAlertsThatFireAsAttentionItems(t *testing.T) {
	router := served(t)

	now := time.Now().UTC()
	body := fmt.Sprintf(group, now.Add(-time.Hour).Format(time.RFC3339),
		now.Add(-time.Minute).Format(time.RFC3339))
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/internal/alerts",
		strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("got %d %q, want the alerts kept", rec.Code, rec.Body)
	}

	want := []string{
		"c4f6a1b2d3e4f5a6@2026-10-30T12:00:00.123456Z | A filesystem is nearly full | " +
			"/ has 4 GiB left | no source",
	}
	if got := listItems(t, router); !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// served returns notify's routes, with a VAPID key of their own and their database in a schema
// of its own.
func served(t *testing.T) http.Handler {
	t.Helper()

	key, err := vapid.Generate()
	if err != nil {
		t.Fatal(err)
	}

	schema := databasetest.NewSchema(t)
	getenv := func(name string) string { return map[string]string{databaseVar: schema.URL}[name] }

	router, err := routes(t.Context(), getenv, secretsWith(map[string]string{
		vapidAccount: key, databaseAccount: schema.Password,
	}))
	if err != nil {
		t.Fatal(err)
	}

	return router
}

// listItems returns the items that router lists over the attention contract, called with the
// gRPC protocol, one line each: their ID, title, detail and source.
func listItems(t *testing.T, router http.Handler) []string {
	t.Helper()

	// gRPC needs HTTP/2; between services it runs without TLS (h2c), as internal/server allows.
	h2c := new(http.Protocols)
	h2c.SetUnencryptedHTTP2(true)

	server := httptest.NewUnstartedServer(router)
	server.Config.Protocols = h2c
	server.Start()
	t.Cleanup(server.Close)

	transport := &http.Transport{Protocols: h2c}
	t.Cleanup(transport.CloseIdleConnections)

	httpClient := &http.Client{Transport: transport}
	client := attentionv1connect.NewAttentionServiceClient(
		connect.NewClient(connecthttp.NewTransport(httpClient, server.URL, connecthttp.WithGRPC())))

	res, err := client.ListItems(t.Context(), &attentionv1.ListItemsRequest{})
	if err != nil {
		t.Fatal(err)
	}

	var lines []string

	for _, item := range res.GetItems() {
		source := "no source"
		if item.HasSource() {
			source = item.GetSource().String()
		}

		lines = append(lines, fmt.Sprintf("%s | %s | %s | %s", item.GetId(), item.GetTitle(),
			item.GetDetail(), source))
	}

	return lines
}

// secretsWith returns a fake secrets reader that has only items, by name.
func secretsWith(items map[string]string) secrets.Reader {
	return func(_ context.Context, name string) (string, error) {
		secret, ok := items[name]
		if !ok {
			return "", fmt.Errorf("no secret %s", name)
		}

		return secret, nil
	}
}
