package cli_test

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/codedogapp/bakhyard/tools/cockpit/internal/cli"
)

const manifestText = `
version = 1

[[service]]
name = "gateway"
description = "The way in"
command = ["bin/gateway", "-v"]
listen = "tcp://127.0.0.1:8080"
depends_on = ["home", "web"]

[[service]]
name = "home"
description = "The Home screen"
command = ["bin/home"]
listen = "tcp://127.0.0.1:8081"

[[service]]
name = "web"
description = "The web app"
command = ["bin/web"]
listen = "tcp://127.0.0.1:8082"
`

func TestListPrintsTheServicesInStartOrder(t *testing.T) {
	fsys := fstest.MapFS{"services.toml": {Data: []byte(manifestText)}}

	var out bytes.Buffer

	err := cli.Run(t.Context(), []string{"list"}, fsys, &out)
	if err != nil {
		t.Fatal(err)
	}

	want := strings.Join([]string{
		"SERVICE  DEPENDS ON  COMMAND         DESCRIPTION",
		"home     -           bin/home        The Home screen",
		"web      -           bin/web         The web app",
		"gateway  home, web   bin/gateway -v  The way in",
		"",
	}, "\n")
	if out.String() != want {
		t.Errorf("got\n%s\nwant\n%s", out.String(), want)
	}
}

func TestRunRejectsUnknownCommands(t *testing.T) {
	fsys := fstest.MapFS{"services.toml": {Data: []byte(manifestText)}}

	for _, args := range [][]string{{"start"}, {"list", "home"}, {"up", "home"}, {"LIST"}, {""}} {
		var out bytes.Buffer

		err := cli.Run(t.Context(), args, fsys, &out)
		if err == nil || err.Error() != "usage: cockpit [list|up]" {
			t.Errorf("Run(%q) = %v, want the usage", args, err)
		}

		if out.Len() != 0 {
			t.Errorf("Run(%q) printed %q", args, out.String())
		}
	}
}

// commands are the ways to run the cockpit that read the manifest; nil opens the TUI.
var commands = [][]string{nil, {"list"}, {"up"}}

func TestCommandsWithoutAManifest(t *testing.T) {
	for _, args := range commands {
		var out bytes.Buffer

		err := cli.Run(t.Context(), args, fstest.MapFS{}, &out)
		if !errors.Is(err, fs.ErrNotExist) || !strings.Contains(err.Error(), "services.toml") {
			t.Errorf("Run(%q): got %v, want the missing manifest", args, err)
		}
	}
}

func TestCommandsWithABadManifest(t *testing.T) {
	fsys := fstest.MapFS{"services.toml": {Data: []byte("version = 2\n")}}

	for _, args := range commands {
		var out bytes.Buffer

		err := cli.Run(t.Context(), args, fsys, &out)
		if err == nil || err.Error() != "services.toml: version must be 1, not 2" {
			t.Errorf("Run(%q): got %v, want the manifest's error", args, err)
		}

		if out.Len() != 0 {
			t.Errorf("Run(%q) printed %q", args, out.String())
		}
	}
}

func TestListReportsAFailedWrite(t *testing.T) {
	fsys := fstest.MapFS{"services.toml": {Data: []byte(manifestText)}}

	err := cli.Run(t.Context(), []string{"list"}, fsys, failingWriter{})
	if !errors.Is(err, errWrite) {
		t.Errorf("got %v, want the write error", err)
	}
}

func TestUpRunsTheServicesUntilTheContextEnds(t *testing.T) {
	const sleeper = `
version = 1

[[service]]
name = "sleeper"
description = "Sleeps"
command = ["sleep", "60"]
listen = "tcp://127.0.0.1:8083"
`

	fsys := fstest.MapFS{"services.toml": {Data: []byte(sleeper)}}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	var out bytes.Buffer

	err := cli.Run(ctx, []string{"up"}, fsys, &out)
	if err != nil {
		t.Fatal(err)
	}

	want := regexp.MustCompile(
		`^cockpit \| started sleeper \(pid \d+\)\ncockpit \| stopping sleeper\n$`,
	)
	if !want.MatchString(out.String()) {
		t.Errorf("got\n%s", out.String())
	}
}

var errWrite = errors.New("disk full")

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errWrite
}
