package main

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"

	attentionv1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1"
	"github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1/attentionv1connect"
	"github.com/codedogapp/bakhyard/internal/secrets"
)

func TestGitHubListsMyReviewRequestsAndIssuesAsAttentionItems(t *testing.T) {
	got := listItems(t, envOf(map[string]string{githubAPIVar: githubServer(t)}),
		secretsWith(map[string]string{"github-token": "secret"}))

	// The stand-in's search for my open PRs finds one in acme/web, which isn't listed.
	want := []string{
		"acme/api#1 acme/api #1 · Review requested",
		"acme/docs#1 acme/docs #1 · Assigned",
	}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGitHubListsNoAttentionItemsWithoutGitHub(t *testing.T) {
	if got := listItems(t, envOf(nil), secretsWith(nil)); len(got) != 0 {
		t.Errorf("got %q, want no items", got)
	}
}

// listItems returns the ID and detail of each item that github lists over the attention
// contract, called with the gRPC protocol, with the settings in getenv and the secrets that
// readSecret reads.
func listItems(t *testing.T, getenv func(string) string, readSecret secrets.Reader) []string {
	t.Helper()

	router, err := routes(t.Context(), getenv, readSecret)
	if err != nil {
		t.Fatal(err)
	}

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

	var items []string

	for _, item := range res.GetItems() {
		items = append(items, item.GetId()+" "+item.GetDetail())
	}

	return items
}
