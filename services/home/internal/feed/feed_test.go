package feed_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/codedogapp/bakhyard/internal/env"
	attentionv1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1"
	"github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1/attentionv1connect"
	"github.com/codedogapp/bakhyard/services/home/internal/feed"
	"github.com/codedogapp/bakhyard/services/home/internal/screen"
)

func TestGatherGathersEveryPeersItems(t *testing.T) {
	due := time.Date(2026, time.October, 18, 9, 30, 0, 0, time.UTC)
	todos := serveAttention(t, answer(
		attentionv1.Item_builder{
			Id: new("WEB-9"), Title: new("Fix the export"), Detail: new("In Progress"),
			Source: new(attentionv1.Source_SOURCE_JIRA), Due: timestamppb.New(due),
		}.Build(),
		attentionv1.Item_builder{Id: new("7"), Title: new("Call the bank")}.Build(),
		attentionv1.Item_builder{Id: new("8"), Source: attentionv1.Source(99).Enum()}.Build(),
	))
	pulls := serveAttention(t, answer(
		attentionv1.Item_builder{
			Id: new("web#1"), Title: new("Ship"), Source: new(attentionv1.Source_SOURCE_BITBUCKET),
		}.Build(),
	))
	github := serveAttention(t, answer(
		attentionv1.Item_builder{
			Id: new("acme/api#3"), Title: new("Cache the search"),
			Source: new(attentionv1.Source_SOURCE_GITHUB),
		}.Build(),
	))

	got, sources, logs := gather(
		t,
		env.Peer{Name: "todos", Address: todos},
		env.Peer{Name: "pulls", Address: pulls},
		env.Peer{Name: "github", Address: github},
	)

	want := []screen.Item{
		{
			ID: "todos/WEB-9", App: "todos", Title: "Fix the export", Detail: "In Progress",
			Source: screen.SourceJira, Due: due,
		},
		{ID: "todos/7", App: "todos", Title: "Call the bank", Source: screen.SourceBakhyard},
		{ID: "todos/8", App: "todos", Source: screen.SourceBakhyard},
		{ID: "pulls/web#1", App: "pulls", Title: "Ship", Source: screen.SourceBitbucket},
		{
			ID: "github/acme/api#3", App: "github", Title: "Cache the search",
			Source: screen.SourceGitHub,
		},
	}
	if !slices.EqualFunc(got, want, sameItem) {
		t.Errorf("got %+v, want %+v", got, want)
	}

	checkSources(t, sources, ok("todos"), ok("pulls"), ok("github"))

	if logs != "" {
		t.Errorf("logged %q, want nothing", logs)
	}
}

func TestGatherCountsAPeerWithoutTheContractAsOKQuietly(t *testing.T) {
	// A service without the contract answers 404, which gRPC reads as Unimplemented.
	notes := serveH2C(t, http.NewServeMux())
	pulls := serveAttention(t, answer(attentionv1.Item_builder{Id: new("web#1")}.Build()))

	got, sources, logs := gather(
		t,
		env.Peer{Name: "notes", Address: notes},
		env.Peer{Name: "pulls", Address: pulls},
	)

	if ids := itemIDs(got); !slices.Equal(ids, []string{"pulls/web#1"}) {
		t.Errorf("got %q, want pulls/web#1 alone", ids)
	}

	checkSources(t, sources, ok("notes"), ok("pulls"))

	if logs != "" {
		t.Errorf("logged %q, want nothing", logs)
	}
}

func TestGatherMarksAFailingPeerUnavailableAndLogsIt(t *testing.T) {
	tests := map[string]string{
		"a peer that fails": serveAttention(t, failing{
			connect.NewError(connect.CodeInternal, "the database is down"),
		}),
		"a peer that is down": downAddress(t),
	}

	for name, address := range tests {
		t.Run(name, func(t *testing.T) {
			pulls := serveAttention(t, answer(attentionv1.Item_builder{Id: new("web#1")}.Build()))

			got, sources, logs := gather(t, env.Peer{Name: "todos", Address: address},
				env.Peer{Name: "pulls", Address: pulls})

			if ids := itemIDs(got); !slices.Equal(ids, []string{"pulls/web#1"}) {
				t.Errorf("got %q, want pulls/web#1 alone", ids)
			}

			checkSources(t, sources, unavailable("todos"), ok("pulls"))

			if !strings.Contains(logs, `level=ERROR msg="list the items" peer=todos err=`) ||
				strings.Count(logs, "\n") != 1 {
				t.Errorf("logged %q, want one line about todos", logs)
			}
		})
	}
}

func TestGatherWithoutPeers(t *testing.T) {
	got, sources, logs := gather(t)

	if len(got) != 0 || len(sources) != 0 || logs != "" {
		t.Errorf("got %+v, sources %+v and logs %q, want none", got, sources, logs)
	}
}

// gather returns what a Feed of peers gathers, and what it logs.
func gather(t *testing.T, peers ...env.Peer) ([]screen.Item, []screen.SourceStatus, string) {
	t.Helper()

	var logs bytes.Buffer

	got, sources := feed.New(peers, slog.New(slog.NewTextHandler(&logs, nil))).Gather(t.Context())

	return got, sources, logs.String()
}

// checkSources fails the test unless got is want.
func checkSources(t *testing.T, got []screen.SourceStatus, want ...screen.SourceStatus) {
	t.Helper()

	if !slices.Equal(got, want) {
		t.Errorf("sources = %+v, want %+v", got, want)
	}
}

func ok(app screen.App) screen.SourceStatus {
	return screen.SourceStatus{App: app, State: screen.StateOK}
}

func unavailable(app screen.App) screen.SourceStatus {
	return screen.SourceStatus{App: app, State: screen.StateUnavailable}
}

// answering answers ListItems with its items.
type answering []*attentionv1.Item

func answer(items ...*attentionv1.Item) answering {
	return items
}

func (a answering) ListItems(
	context.Context, *attentionv1.ListItemsRequest,
) (*attentionv1.ListItemsResponse, error) {
	return attentionv1.ListItemsResponse_builder{Items: a}.Build(), nil
}

// failing answers ListItems with its error.
type failing struct {
	err error
}

func (f failing) ListItems(
	context.Context, *attentionv1.ListItemsRequest,
) (*attentionv1.ListItemsResponse, error) {
	return nil, f.err
}

// serveAttention serves handler over the attention contract on h2c and returns its address.
func serveAttention(t *testing.T, handler attentionv1connect.AttentionServiceHandler) string {
	t.Helper()

	server := connect.NewServer()
	attentionv1connect.RegisterAttentionServiceHandler(server, handler)

	mux := http.NewServeMux()
	connecthttp.Mount(mux, server)

	return serveH2C(t, grpcOnly(mux))
}

// grpcOnly refuses a call that isn't gRPC, which the services speak to each other (decision 158),
// though the contract's handlers answer Connect's protocol and gRPC-Web too.
func grpcOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/grpc" &&
			!strings.HasPrefix(contentType, "application/grpc+") {
			http.Error(w, "gRPC only", http.StatusUnsupportedMediaType)

			return
		}

		next.ServeHTTP(w, r)
	})
}

// serveH2C serves handler on HTTP/2 without TLS, as the services do, and returns its address.
func serveH2C(t *testing.T, handler http.Handler) string {
	t.Helper()

	h2c := new(http.Protocols)
	h2c.SetUnencryptedHTTP2(true)

	server := httptest.NewUnstartedServer(handler)
	server.Config.Protocols = h2c
	server.Start()
	t.Cleanup(server.Close)

	return server.Listener.Addr().String()
}

// downAddress returns an address where nothing listens any more.
func downAddress(t *testing.T) string {
	t.Helper()

	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()

	return server.Listener.Addr().String()
}

func itemIDs(items []screen.Item) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}

	return ids
}

func sameItem(a, b screen.Item) bool {
	due := a.Due.Equal(b.Due)
	a.Due, b.Due = time.Time{}, time.Time{}

	return due && a == b
}
