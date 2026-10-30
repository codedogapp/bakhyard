// Package copilot works out plans with the Copilot CLI, through the Copilot SDK.
package copilot

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"

	sdk "github.com/github/copilot-sdk/go"
)

// Settings say where the Copilot CLI is and which account it works for.
type Settings struct {
	// CLI is the path of the copilot executable.
	CLI string
	// Home is the CLI's own folder for its state, the plans' sessions among it.
	Home string
	// Work is an empty folder, outside any git repository, that the CLI works in.
	Work string
	// Host is the account's GitHub host.
	Host string
	// Token is the account's fine-grained token with the Copilot Requests permission.
	Token string
}

// Client works out plans on a Copilot CLI of its own, and keeps those it has started.
type Client struct {
	copilot *sdk.Client
	work    string
	// stopped closes as the client stops, with stop, which ends the questions' waits.
	stopped <-chan struct{}
	stop    func()

	mu    sync.Mutex
	plans map[string]*plan
}

// New returns a client for the CLI that settings name. The CLI inherits environ, less the
// variables that set up Copilot or GitHub, so only settings decide whom it works for.
func New(settings Settings, environ []string) *Client {
	return newClient(clientOptions(settings, environ), settings.Work)
}

func newClient(options *sdk.ClientOptions, work string) *Client {
	stopped := make(chan struct{})

	return &Client{
		copilot: sdk.NewClient(options),
		work:    work,
		stopped: stopped,
		stop:    sync.OnceFunc(func() { close(stopped) }),
		plans:   map[string]*plan{},
	}
}

func clientOptions(settings Settings, environ []string) *sdk.ClientOptions {
	return &sdk.ClientOptions{
		Connection: sdk.StdioConnection{
			Path: settings.CLI,
			Env:  cliEnv(environ, settings.Host),
		},
		WorkingDirectory: settings.Work,
		BaseDirectory:    settings.Home,
		GitHubToken:      settings.Token,
		UseLoggedInUser:  new(false),
	}
}

// cliEnv returns environ without the variables that set up Copilot or GitHub, such as another
// account's token, and with host as the CLI's GitHub host.
func cliEnv(environ []string, host string) []string {
	env := slices.DeleteFunc(slices.Clone(environ), setsUpCopilot)

	return append(env, "COPILOT_GH_HOST="+host)
}

// setsUpCopilot reports whether variable, a NAME=value pair, sets up Copilot or GitHub.
func setsUpCopilot(variable string) bool {
	name, _, _ := strings.Cut(variable, "=")

	return strings.HasPrefix(name, "COPILOT_") || strings.HasPrefix(name, "GH_") ||
		name == "GITHUB_TOKEN"
}

// Start starts the CLI and connects to it.
func (c *Client) Start(ctx context.Context) error {
	if err := c.copilot.Start(ctx); err != nil {
		return fmt.Errorf("start the copilot CLI: %w", err)
	}

	return nil
}

// Stop ends the questions' waits for their answers, so that the CLI gets an error for each,
// leaves the plans' sessions, which the CLI keeps, and stops the CLI.
func (c *Client) Stop() error {
	c.stop()

	if err := c.copilot.Stop(); err != nil {
		return fmt.Errorf("stop the copilot CLI: %w", err)
	}

	return nil
}
