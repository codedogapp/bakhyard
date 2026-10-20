package cli_test

import (
	"bytes"
	"errors"
	"io/fs"
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
depends_on = ["home", "web"]

[[service]]
name = "home"
description = "The Home screen"
command = ["bin/home"]

[[service]]
name = "web"
description = "The web app"
command = ["bin/web"]
`

func TestListPrintsTheServicesInStartOrder(t *testing.T) {
	fsys := fstest.MapFS{"services.toml": {Data: []byte(manifestText)}}

	var out bytes.Buffer

	err := cli.Run([]string{"list"}, fsys, &out)
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

func TestRunRejectsAnythingButList(t *testing.T) {
	fsys := fstest.MapFS{"services.toml": {Data: []byte(manifestText)}}

	for _, args := range [][]string{nil, {"start"}, {"list", "home"}, {"LIST"}} {
		var out bytes.Buffer

		err := cli.Run(args, fsys, &out)
		if err == nil || err.Error() != "usage: cockpit list" {
			t.Errorf("Run(%q) = %v, want the usage", args, err)
		}

		if out.Len() != 0 {
			t.Errorf("Run(%q) printed %q", args, out.String())
		}
	}
}

func TestListWithoutAManifest(t *testing.T) {
	var out bytes.Buffer

	err := cli.Run([]string{"list"}, fstest.MapFS{}, &out)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("got %v, want a missing file", err)
	}

	if !strings.Contains(err.Error(), "services.toml") {
		t.Errorf("the error %q doesn't name the manifest", err)
	}
}

func TestListWithABadManifest(t *testing.T) {
	fsys := fstest.MapFS{"services.toml": {Data: []byte("version = 2\n")}}

	var out bytes.Buffer

	err := cli.Run([]string{"list"}, fsys, &out)
	if err == nil || err.Error() != "services.toml: version must be 1, not 2" {
		t.Errorf("got %v, want the manifest's error", err)
	}

	if out.Len() != 0 {
		t.Errorf("printed %q", out.String())
	}
}

func TestListReportsAFailedWrite(t *testing.T) {
	fsys := fstest.MapFS{"services.toml": {Data: []byte(manifestText)}}

	err := cli.Run([]string{"list"}, fsys, failingWriter{})
	if !errors.Is(err, errWrite) {
		t.Errorf("got %v, want the write error", err)
	}
}

var errWrite = errors.New("disk full")

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errWrite
}
