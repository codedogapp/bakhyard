package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/codedogapp/bakhyard/services/planner/internal/copilot"
)

// token is the secrets of a Copilot that is set up.
var token = map[string]string{"copilot-token": "secret"}

func TestPlannerStopsOnABadSetup(t *testing.T) {
	home := t.TempDir()
	failing := cliOn(t, "/usr/bin/false")

	file := filepath.Join(t.TempDir(), "Copilot")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name, home, path string
		secrets          map[string]string
		want             string
	}{
		{"no home", "", failing, token, "BAKHYARD_COPILOT_HOME is not set"},
		{
			"a relative home", "Copilot", failing, token,
			`BAKHYARD_COPILOT_HOME: "Copilot" is not an absolute path`,
		},
		{"no token", home, failing, nil, "read the Copilot token: no secret copilot-token"},
		{
			"no CLI", home, t.TempDir(), token,
			`find the Copilot CLI: exec: "copilot": executable file not found in $PATH`,
		},
		{"a home that is a file", file, failing, token, "create the Copilot CLI's folder: "},
		{"a CLI that fails", home, failing, token, "start the copilot CLI: "},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("PATH", test.path)

			var stopping sync.WaitGroup

			_, err := routes(t.Context(), copilotEnv(test.home), secretsWith(test.secrets),
				&stopping)
			if err == nil || !strings.HasPrefix(err.Error(), test.want) {
				t.Errorf("got %v, want the error %s", err, test.want)
			}
		})
	}
}

func TestCopilotSettingsComeFromTheEnvironment(t *testing.T) {
	folder := cliOn(t, "/usr/bin/false")
	t.Setenv("PATH", folder)

	home := filepath.Join(t.TempDir(), "Copilot")

	settings, err := copilotSettings(t.Context(), copilotEnv(home), secretsWith(token))

	want := copilot.Settings{
		CLI: filepath.Join(folder, "copilot"), Home: home, Work: filepath.Join(home, "work"),
		Host: "example.ghe.com", Token: "secret",
	}
	if err != nil || settings != want {
		t.Errorf("got %+v, %v, want %+v", settings, err, want)
	}

	for _, made := range []string{home, want.Work} {
		info, err := os.Stat(made)
		if err != nil || info.Mode() != os.ModeDir|0o700 {
			t.Errorf("%s: got %v, %v, want a folder that only the user opens", made, info, err)
		}
	}
}

func TestPlannerRunsCopilotUntilItStops(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", cliOn(t, executable))

	home := t.TempDir()
	ctx, stop := context.WithCancel(t.Context())

	var stopping sync.WaitGroup

	router, err := routes(ctx, copilotEnv(home), secretsWith(token), &stopping)
	if err != nil {
		t.Fatal(err)
	}

	// The client answers, not the 404 of a Copilot that isn't set up.
	rec := serve(t, router, http.MethodGet, "/planner/plans/x")
	if rec.Code != http.StatusNotFound || rec.Body.String() != "no such plan\n" {
		t.Errorf("got %d %q, want 404 no such plan", rec.Code, rec.Body)
	}

	stop()
	stopping.Wait()

	if _, err := os.Stat(filepath.Join(home, "work", "stopped")); err != nil {
		t.Errorf("got %v, want the CLI to have stopped in the work folder", err)
	}
}

// copilotEnv returns an environment where Copilot is set up, with home as the CLI's folder.
func copilotEnv(home string) func(string) string {
	return envOf(map[string]string{hostVar: "example.ghe.com", homeVar: home})
}

// cliOn returns a folder that holds only target, under the name copilot.
func cliOn(t *testing.T, target string) string {
	t.Helper()

	folder := t.TempDir()
	if err := os.Symlink(target, filepath.Join(folder, "copilot")); err != nil {
		t.Fatal(err)
	}

	return folder
}
