// Package cli is the nudge command: ask, mcp and setup. Each subcommand reaches the Bakhyard
// gateway at the server that the settings give.
package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/codedogapp/bakhyard/tools/nudge/internal/setup"
)

// Exit codes.
const (
	exitOK    = 0
	exitError = 1
	exitUsage = 2
)

const usage = `usage: nudge <command>

  ask     ask the user a question on their phone, and print the answer
  mcp     serve the ask and report tools over MCP, on standard input and output
  setup   point this machine at the server and add nudge to Copilot CLI
`

// Env is what the command takes from the machine it runs on.
type Env struct {
	Version string
	Getenv  func(string) string
	Config  func() (string, error)
	Host    func() (string, error)
	// Run runs the outside tools that setup needs; nil means the real ones.
	Run setup.Run
	// HTTP is the client for the server; nil means a default one.
	HTTP   *http.Client
	Stdout io.Writer
	Stderr io.Writer
}

// Run runs the command that args name, and returns the exit code.
func Run(ctx context.Context, args []string, env Env) int {
	if len(args) == 0 {
		_, _ = fmt.Fprint(env.Stderr, usage)

		return exitUsage
	}

	switch args[0] {
	case "ask":
		return runAsk(ctx, args[1:], env)
	case "mcp":
		return runMCP(ctx, env)
	case "setup":
		return runSetup(ctx, args[1:], env)
	case "version", "--version":
		_, _ = fmt.Fprintln(env.Stdout, env.Version)

		return exitOK
	default:
		_, _ = fmt.Fprintf(env.Stderr, "nudge: unknown command %q\n\n%s", args[0], usage)

		return exitUsage
	}
}

func fail(env Env, command string, err error) int {
	_, _ = fmt.Fprintf(env.Stderr, "nudge %s: %v\n", command, err)

	return exitError
}
