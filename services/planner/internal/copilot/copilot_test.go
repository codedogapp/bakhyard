package copilot

import (
	"errors"
	"io/fs"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	sdk "github.com/github/copilot-sdk/go"
)

func TestClientOptions(t *testing.T) {
	settings := Settings{
		CLI: "/tools/copilot", Home: "/planner/home", Work: "/planner/work",
		Host: "example.ghe.com", Token: "secret-token",
	}

	options := clientOptions(settings, []string{"HOME=/home/user", "GH_TOKEN=personal-token"})

	want := sdk.StdioConnection{
		Path: "/tools/copilot", Env: []string{"HOME=/home/user", "COPILOT_GH_HOST=example.ghe.com"},
	}
	if !reflect.DeepEqual(options.Connection, want) {
		t.Errorf("Connection = %#v, want %#v", options.Connection, want)
	}

	if options.WorkingDirectory != "/planner/work" || options.BaseDirectory != "/planner/home" {
		t.Errorf("folders = %q, %q, want the work and home folders",
			options.WorkingDirectory, options.BaseDirectory)
	}

	if options.GitHubToken != "secret-token" {
		t.Errorf("GitHubToken = %q, want the settings' token", options.GitHubToken)
	}

	if options.UseLoggedInUser == nil || *options.UseLoggedInUser {
		t.Error("UseLoggedInUser isn't false, so the CLI may sign in as the logged-in user")
	}
}

func TestCLIEnv(t *testing.T) {
	tests := []struct {
		name          string
		environ, want []string
	}{
		{
			name: "keeps the rest",
			environ: []string{
				"PATH=/usr/bin", "HOME=/home/user", "GITHUB_ACTIONS=true", "GH=x", "NOT_GH_TOKEN=x",
			},
			want: []string{
				"PATH=/usr/bin", "HOME=/home/user", "GITHUB_ACTIONS=true", "GH=x", "NOT_GH_TOKEN=x",
				"COPILOT_GH_HOST=example.ghe.com",
			},
		},
		{
			name: "drops tokens",
			environ: []string{
				"COPILOT_GITHUB_TOKEN=a", "PATH=/usr/bin", "GH_TOKEN=b", "GITHUB_TOKEN=c",
				"GH_ENTERPRISE_TOKEN=d",
			},
			want: []string{"PATH=/usr/bin", "COPILOT_GH_HOST=example.ghe.com"},
		},
		{
			name: "drops settings",
			environ: []string{
				"COPILOT_ALLOW_ALL=true",
				"COPILOT_GH_HOST=github.com",
				"GH_HOST=github.com",
				"COPILOT_HOME=/home/user/.copilot",
				"COPILOT_PROVIDER_BASE_URL=http://x",
				"COPILOT_X",
			},
			want: []string{"COPILOT_GH_HOST=example.ghe.com"},
		},
		{
			name: "from nothing",
			want: []string{"COPILOT_GH_HOST=example.ghe.com"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			environ := slices.Clone(test.environ)

			if got := cliEnv(environ, "example.ghe.com"); !slices.Equal(got, test.want) {
				t.Errorf("cliEnv = %q, want %q", got, test.want)
			}

			if !slices.Equal(environ, test.environ) {
				t.Errorf("cliEnv changed its environ to %q", environ)
			}
		})
	}
}

func TestStartWithoutCLI(t *testing.T) {
	work := t.TempDir()
	client := New(Settings{CLI: filepath.Join(work, "copilot"), Work: work, Token: "x"}, nil)

	if err := client.Start(t.Context()); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Start = %v, want an error for the missing CLI", err)
	}
}
