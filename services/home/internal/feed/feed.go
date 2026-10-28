// Package feed gathers what needs the user from home's peers, which list it over the attention
// contract (proto/bakhyard/attention/v1).
package feed

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"sync"
	"time"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"

	"github.com/codedogapp/bakhyard/internal/env"
	attentionv1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1"
	"github.com/codedogapp/bakhyard/internal/gen/bakhyard/attention/v1/attentionv1connect"
	"github.com/codedogapp/bakhyard/services/home/internal/screen"
)

// timeout is how long a peer has to answer before the Home goes without its items.
const timeout = 1500 * time.Millisecond

// Feed asks home's peers what needs the user.
type Feed struct {
	peers  []peer
	logger *slog.Logger
}

// peer is a service that home asks, with the name that its items' IDs and app take.
type peer struct {
	name   string
	client attentionv1connect.AttentionServiceClient
}

// New returns a Feed that asks peers over gRPC on h2c, HTTP/2 without TLS, as the services serve
// it, and logs their failures to logger.
func New(peers []env.Peer, logger *slog.Logger) *Feed {
	h2c := new(http.Protocols)
	h2c.SetUnencryptedHTTP2(true)

	httpClient := &http.Client{Transport: &http.Transport{Protocols: h2c}}
	feed := &Feed{logger: logger}

	for _, setting := range peers {
		server := url.URL{Scheme: "http", Host: setting.Address}
		transport := connecthttp.NewTransport(httpClient, server.String(), connecthttp.WithGRPC())
		client := attentionv1connect.NewAttentionServiceClient(connect.NewClient(transport))
		feed.peers = append(feed.peers, peer{name: setting.Name, client: client})
	}

	return feed
}

// Items asks every peer at once and returns their items, each peer's in the order it sent them.
// A peer that doesn't serve the contract has none. One that fails, or doesn't answer within
// timeout, is left out and logged, unless ctx has ended: the client has gone.
func (f *Feed) Items(ctx context.Context) []screen.Item {
	lists := make([][]screen.Item, len(f.peers))

	var wg sync.WaitGroup
	for i, p := range f.peers {
		wg.Go(func() { lists[i] = p.items(ctx, f.logger) })
	}

	wg.Wait()

	return slices.Concat(lists...)
}

// items asks the peer for its items. See Feed.Items.
func (p peer) items(ctx context.Context, logger *slog.Logger) []screen.Item {
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	res, err := p.client.ListItems(callCtx, &attentionv1.ListItemsRequest{})
	if err != nil {
		if ctx.Err() == nil && connect.CodeOf(err) != connect.CodeUnimplemented {
			logger.ErrorContext(ctx, "list the items", "peer", p.name, "err", err)
		}

		return nil
	}

	items := make([]screen.Item, 0, len(res.GetItems()))
	for _, listed := range res.GetItems() {
		items = append(items, homeItem(p.name, listed))
	}

	return items
}
