package main

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"

	attentionv1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1"
	"github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1/attentionv1connect"
	"github.com/codedogapp/bakhyard/internal/secrets"
)

func TestPullsListsMyPullsAsAttentionItems(t *testing.T) {
	address, _ := bitbucketServer(t)

	got := listItems(t, envOf(bitbucketSettings(address)),
		secretsWith(map[string]string{"bitbucket-token": "secret"}))

	want := []string{"acme/web#1 Ship", "labs/web#1 Ship"}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPullsListsNoAttentionItemsWithoutBitbucket(t *testing.T) {
	if got := listItems(t, envOf(nil), secretsWith(nil)); len(got) != 0 {
		t.Errorf("got %q, want no items", got)
	}
}

func TestPullsRemindsToRenewTheBitbucketToken(t *testing.T) {
	address, _ := bitbucketServer(t)
	settings := bitbucketSettings(address)
	settings[bitbucketExpiresVar] = time.Now().UTC().AddDate(0, 0, 3).Format(time.DateOnly)

	got := listItems(t, envOf(settings),
		secretsWith(map[string]string{"bitbucket-token": "secret"}))

	want := []string{
		"acme/web#1 Ship", "labs/web#1 Ship", "bitbucket-token Renew the Bitbucket token",
	}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPullsRemindsToRenewTheBitbucketTokenWithoutBitbucket(t *testing.T) {
	expires := time.Now().UTC().AddDate(0, 0, 3).Format(time.DateOnly)

	got := listItems(t, envOf(map[string]string{bitbucketExpiresVar: expires}), secretsWith(nil))

	want := []string{"bitbucket-token Renew the Bitbucket token"}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// listItems returns the ID and title of each item that pulls lists over the attention contract,
// called with the gRPC protocol, with the settings in getenv and the secrets that readSecret reads.
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
		items = append(items, item.GetId()+" "+item.GetTitle())
	}

	return items
}
