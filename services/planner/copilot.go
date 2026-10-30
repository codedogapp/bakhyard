package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/codedogapp/bakhyard/internal/env"
	"github.com/codedogapp/bakhyard/internal/secrets"
	"github.com/codedogapp/bakhyard/services/planner/internal/api"
	"github.com/codedogapp/bakhyard/services/planner/internal/copilot"
)

// hostVar is the setting of the Copilot account's GitHub host, such as example.ghe.com
// (docs/setup.md). Copilot is off when it isn't set.
const hostVar = "BAKHYARD_COPILOT_HOST"

// homeVar is the setting of the Copilot CLI's own folder, an absolute path.
const homeVar = "BAKHYARD_COPILOT_HOME"

// tokenAccount is the Keychain account of the Copilot token, and the name of its file.
const tokenAccount = "copilot-token"

// folderMode lets only the user into the CLI's folders.
const folderMode = 0o700

// startPlanner starts the Copilot CLI in the settings, with the token that readSecret reads, and
// returns its client, or nil when Copilot is off. The CLI stops once ctx ends, and stopping waits
// for it.
func startPlanner(
	ctx context.Context,
	getenv func(string) string,
	readSecret secrets.Reader,
	stopping *sync.WaitGroup,
) (api.Planner, error) {
	if getenv(hostVar) == "" {
		return nil, nil
	}

	settings, err := copilotSettings(ctx, getenv, readSecret)
	if err != nil {
		return nil, err
	}

	client := copilot.New(settings, os.Environ())
	if err := client.Start(ctx); err != nil {
		return nil, err
	}

	stopping.Add(1)
	context.AfterFunc(ctx, func() {
		defer stopping.Done()

		if err := client.Stop(); err != nil {
			slog.Error("stop Copilot", "err", err)
		}
	})

	return client, nil
}

// copilotSettings returns the Copilot CLI's settings, with the token that readSecret reads, once
// it has made the empty folder that the CLI works in.
func copilotSettings(
	ctx context.Context,
	getenv func(string) string,
	readSecret secrets.Reader,
) (copilot.Settings, error) {
	home, err := env.Required(getenv, homeVar)
	if err != nil {
		return copilot.Settings{}, err
	}

	// The CLI runs in another folder, where a relative path would point elsewhere.
	if !filepath.IsAbs(home) {
		return copilot.Settings{}, fmt.Errorf("%s: %q is not an absolute path", homeVar, home)
	}

	token, err := readSecret(ctx, tokenAccount)
	if err != nil {
		return copilot.Settings{}, fmt.Errorf("read the Copilot token: %w", err)
	}

	cli, err := exec.LookPath("copilot")
	if err != nil {
		return copilot.Settings{}, fmt.Errorf("find the Copilot CLI: %w", err)
	}

	work := filepath.Join(home, "work")
	if err := os.MkdirAll(work, folderMode); err != nil {
		return copilot.Settings{}, fmt.Errorf("create the Copilot CLI's folder: %w", err)
	}

	return copilot.Settings{CLI: cli, Home: home, Work: work, Host: getenv(hostVar), Token: token},
		nil
}
