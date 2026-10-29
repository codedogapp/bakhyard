package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/codedogapp/bakhyard/internal/jira"
	"github.com/codedogapp/bakhyard/services/time/internal/api"
	"github.com/codedogapp/bakhyard/services/time/internal/worklogs"
)

// poll starts polling Jira with client for the user's worklogs, until ctx ends, and returns the
// backends of time's API: the poller that keeps the time report that they make, whose days follow
// loc, and client's AddWorklog, so that the polls find the worklogs that client adds (docs/plan.md
// decision 191). When Jira is off and client is nil, it returns the zero Backends.
func poll(
	ctx context.Context,
	client *jira.Client,
	loc *time.Location,
	logger *slog.Logger,
) api.Backends {
	if client == nil {
		return api.Backends{}
	}

	poller := worklogs.NewPoller(client.Worklogs, loc, logger)
	go poller.Run(ctx)

	return api.Backends{Reports: poller, Add: client.AddWorklog}
}
