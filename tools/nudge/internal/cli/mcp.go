package cli

import (
	"context"
	"os/signal"
	"syscall"
	"uuid"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/codedogapp/bakhyard/tools/nudge/internal/door"
	"github.com/codedogapp/bakhyard/tools/nudge/internal/mcptool"
)

// runMCP serves the ask and report tools on standard input and output until the agent goes away.
func runMCP(ctx context.Context, env Env) int {
	client, err := connect(env)
	if err != nil {
		return fail(env, "mcp", err)
	}

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	label := project(env)
	reporter := door.NewReporter(ctx, client, label)

	ask := func(ctx context.Context, question door.Question) (door.Answer, error) {
		question.ID, question.Project = uuid.New().String(), label

		return client.Ask(ctx, question)
	}

	// The session shows up on the phone page at once; a failure is retried by the first report.
	go func() { _ = reporter.Register() }()

	err = mcptool.Run(
		ctx,
		env.Version,
		mcptool.Door{Ask: ask, Report: reporter.Report},
		&mcp.StdioTransport{},
	)
	if err != nil && ctx.Err() == nil {
		return fail(env, "mcp", err)
	}

	return exitOK
}
