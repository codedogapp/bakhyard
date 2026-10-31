// Package week asks the time service for this week's hours, for the Home's "This week", over the
// time contract (proto/bakhyard/time/v1).
package week

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"slices"
	"time"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"

	"github.com/codedogapp/bakhyard/internal/env"
	timev1 "github.com/codedogapp/bakhyard/internal/gen/bakhyard/time/v1"
	"github.com/codedogapp/bakhyard/internal/gen/bakhyard/time/v1/timev1connect"
	"github.com/codedogapp/bakhyard/internal/requestid"
	"github.com/codedogapp/bakhyard/services/home/internal/screen"
)

// peerName is the time service's name among home's peers.
const peerName = "time"

// timeout is how long time has to answer before the Home goes without its week, as long as each
// peer has to list its items.
const timeout = 1500 * time.Millisecond

// Asker asks the time service for this week's hours. Its zero value has no time service to ask.
type Asker struct {
	client timev1connect.TimeServiceClient
	logger *slog.Logger
}

// New returns an Asker that asks the peer named time over gRPC on h2c, HTTP/2 without TLS, as the
// services serve it, sends each call the ID of the request that it's made for, and logs its
// failures to logger. Without a peer named time, it asks nothing.
func New(peers []env.Peer, logger *slog.Logger) Asker {
	index := slices.IndexFunc(peers, func(peer env.Peer) bool { return peer.Name == peerName })
	if index < 0 {
		return Asker{}
	}

	h2c := new(http.Protocols)
	h2c.SetUnencryptedHTTP2(true)

	httpClient := &http.Client{Transport: requestid.Transport(&http.Transport{Protocols: h2c})}
	server := url.URL{Scheme: "http", Host: peers[index].Address}
	transport := connecthttp.NewTransport(httpClient, server.String(), connecthttp.WithGRPC())

	return Asker{
		client: timev1connect.NewTimeServiceClient(connect.NewClient(transport)),
		logger: logger,
	}
}

// Ask asks time for this week's hours. It returns the week, or nil when time has none, and how
// asking went: OK once time answers, and unavailable when it fails, doesn't answer within
// timeout, or answers hours that aren't finite, which JSON can't encode. A failure is logged
// unless ctx has ended, since the client has gone then. Without a time service to ask, Ask
// returns no week and no state.
func (a Asker) Ask(ctx context.Context) (*screen.Week, screen.State) {
	if a.client == nil {
		return nil, ""
	}

	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	res, err := a.client.GetWeek(callCtx, &timev1.GetWeekRequest{})
	if err != nil {
		a.logFailure(ctx, err)

		return nil, screen.StateUnavailable
	}

	if !res.HasWeek() {
		return nil, screen.StateOK
	}

	answer := res.GetWeek()

	logged, target := answer.GetLoggedHours(), answer.GetTargetHours()
	if !finite(logged) || !finite(target) {
		a.logFailure(ctx, fmt.Errorf("hours that aren't finite: %v of %v", logged, target))

		return nil, screen.StateUnavailable
	}

	return &screen.Week{LoggedHours: logged, TargetHours: target}, screen.StateOK
}

// logFailure logs why asking failed, unless ctx has ended, since the client has gone then.
func (a Asker) logFailure(ctx context.Context, err error) {
	if ctx.Err() == nil {
		a.logger.ErrorContext(ctx, "get the week", "peer", peerName, "err", err)
	}
}

// finite reports whether hours is neither NaN nor infinite.
func finite(hours float64) bool {
	return !math.IsNaN(hours) && !math.IsInf(hours, 0)
}
