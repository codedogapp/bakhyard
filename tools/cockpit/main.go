// Command cockpit runs and watches Bakhyard's services (docs/plan.md §4.9).
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/codedogapp/bakhyard/tools/cockpit/internal/cli"
)

func main() {
	// Ctrl-C, a kill and a closed terminal all end the context, so up stops the services
	// before the cockpit exits.
	ctx, stop := signal.NotifyContext(
		context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGHUP,
	)

	err := cli.Run(ctx, os.Args[1:], os.DirFS("."), os.Stdout)

	stop()

	if err != nil {
		fmt.Fprintln(os.Stderr, "cockpit:", err)
		os.Exit(1)
	}
}
