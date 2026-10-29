package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/codedogapp/bakhyard/internal/jira"
	"github.com/codedogapp/bakhyard/services/time/internal/worklogs"
)

// poll starts polling Jira with client for the user's worklogs, until ctx ends, and returns the
// poller that keeps the time report that they make, whose days follow loc, or nil when Jira is off
// and client is nil.
func poll(
	ctx context.Context,
	client *jira.Client,
	loc *time.Location,
	logger *slog.Logger,
) *worklogs.Poller {
	if client == nil {
		return nil
	}

	poller := worklogs.NewPoller(client.Worklogs, loc, logger)
	go poller.Run(ctx)

	return poller
}
