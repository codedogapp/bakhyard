// Command nudge is the agent's side of nudge: it asks the user a question on their phone, and
// serves the same through MCP. The service itself is part of Bakhyard.
package main

import (
	"context"
	"os"

	"github.com/codedogapp/bakhyard/tools/nudge/internal/cli"
)

// version is set at build time.
var version = "dev"

func main() {
	os.Exit(cli.Run(context.Background(), os.Args[1:], cli.Env{
		Version: version,
		Getenv:  os.Getenv,
		Config:  os.UserConfigDir,
		Host:    os.Hostname,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	}))
}
