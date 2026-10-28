package cli

import (
	"context"
	"flag"
	"fmt"
	"os/signal"
	"strings"
	"syscall"
	"uuid"

	"github.com/codedogapp/bakhyard/tools/nudge/internal/door"
)

type options []string

func (o *options) String() string     { return strings.Join(*o, ", ") }
func (o *options) Set(v string) error { *o = append(*o, v); return nil }

// runAsk asks the question that args hold and prints the answer.
func runAsk(ctx context.Context, args []string, env Env) int {
	flags := flag.NewFlagSet("nudge ask", flag.ContinueOnError)
	flags.SetOutput(env.Stderr)

	var choices options

	details := flags.String("details", "", "what the user needs to know to answer")
	flags.Var(&choices, "option", "an answer the user can pick (repeatable)")

	if err := flags.Parse(args); err != nil {
		return exitUsage
	}

	if flags.NArg() != 1 {
		_, _ = fmt.Fprintln(
			env.Stderr,
			"usage: nudge ask [--details TEXT] [--option TEXT]... QUESTION",
		)

		return exitUsage
	}

	client, err := connect(env)
	if err != nil {
		return fail(env, "ask", err)
	}

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	answer, err := client.Ask(ctx, door.Question{
		ID: uuid.New().String(), Project: project(env), Text: flags.Arg(0),
		Details: *details, Options: choices,
	})
	if err != nil {
		return fail(env, "ask", err)
	}

	if answer.Option != "" {
		_, _ = fmt.Fprintf(env.Stdout, "option: %s\n", answer.Option)
	}

	if answer.Text != "" {
		_, _ = fmt.Fprintf(env.Stdout, "text: %s\n", answer.Text)
	}

	return exitOK
}
