package cli_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/tools/nudge/internal/cli"
	"github.com/codedogapp/bakhyard/tools/nudge/internal/setup"
)

type result struct {
	code        int
	out, errOut string
}

// run runs nudge with args, with a config folder that holds settings (when not empty) and the
// environment variable NUDGE_SERVER set to server.
// scene is what a run of nudge sits around.
type scene struct {
	server   string
	settings string
	client   *http.Client
	tool     setup.Run
}

func run(t *testing.T, around scene, args ...string) result {
	t.Helper()

	config := t.TempDir()
	if around.settings != "" {
		dir := filepath.Join(config, "nudge")
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(
			filepath.Join(dir, "settings.json"),
			[]byte(around.settings),
			0o600,
		); err != nil {
			t.Fatal(err)
		}
	}

	var out, errOut bytes.Buffer

	code := cli.Run(context.Background(), args, cli.Env{
		Version: "v9",
		Getenv:  func(string) string { return around.server },
		Config:  func() (string, error) { return config, nil },
		Host:    func() (string, error) { return "work-laptop.local", nil },
		HTTP:    around.client,
		Run:     around.tool,
		Stdout:  &out,
		Stderr:  &errOut,
	})

	return result{code, out.String(), errOut.String()}
}

func TestAskPrintsTheAnswer(t *testing.T) {
	var got string

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got = string(body)

		_, _ = w.Write([]byte(`{"option":"Yes","text":"go on"}`))
	}))
	defer server.Close()

	res := run(
		t,
		scene{server: server.URL, client: server.Client()},
		"ask",
		"--details",
		"CI is green",
		"--option",
		"Yes",
		"--option",
		"No",
		"Merge?",
	)

	if res.code != 0 || res.out != "option: Yes\ntext: go on\n" {
		t.Errorf("got %+v", res)
	}

	for _, want := range []string{
		`"text":"Merge?"`, `"details":"CI is green"`, `"options":["Yes","No"]`,
		`(work-laptop)"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("request %s lacks %s", got, want)
		}
	}
}

func TestAskReportsAFailure(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "the question has no text", http.StatusBadRequest)
	}))
	defer server.Close()

	res := run(t, scene{server: server.URL, client: server.Client()}, "ask", "x")
	if res.code != 1 || !strings.Contains(res.errOut, "the question has no text") {
		t.Errorf("got %+v", res)
	}
}

func TestTheServerComesFromTheSettingsFile(t *testing.T) {
	res := run(t, scene{settings: `{"server":"http://insecure.example","login":"old"}`}, "ask", "x")
	if res.code != 1 || !strings.Contains(res.errOut, "isn't an https URL") {
		t.Errorf("got %+v", res)
	}
}

func TestFailuresOfTheSettings(t *testing.T) {
	for name, settings := range map[string]string{"no server": "", "invalid file": "{"} {
		res := run(t, scene{settings: settings}, "ask", "x")
		if res.code != 1 || res.errOut == "" {
			t.Errorf("%s: got %+v", name, res)
		}
	}

	if res := run(t, scene{}, "mcp"); res.code != 1 {
		t.Errorf("mcp without a server: got %+v", res)
	}
}

func TestUsage(t *testing.T) {
	tests := map[string][]string{
		"no command":      {},
		"unknown command": {"frobnicate"},
		"no question":     {"ask"},
		"unknown flag":    {"ask", "--frob", "x"},
		"no server":       {"setup"},
	}

	for name, args := range tests {
		if res := run(t, scene{}, args...); res.code != 2 || res.errOut == "" {
			t.Errorf("%s: got %+v", name, res)
		}
	}
}

func TestVersion(t *testing.T) {
	if res := run(t, scene{}, "version"); res.code != 0 || res.out != "v9\n" {
		t.Errorf("got %+v", res)
	}
}

func TestTheConfigFolderIsMissing(t *testing.T) {
	var errOut bytes.Buffer

	code := cli.Run(context.Background(), []string{"ask", "x"}, cli.Env{
		Config: func() (string, error) { return "", errors.New("no home") },
		Stderr: &errOut,
	})
	if code != 1 || !strings.Contains(errOut.String(), "no home") {
		t.Errorf("got %d %q", code, errOut.String())
	}
}

func TestSetupWritesTheSettingsAndAddsNudgeToCopilot(t *testing.T) {
	var calls []string

	tool := func(_ context.Context, name string, args ...string) ([]byte, error) {
		calls = append(calls, name+" "+strings.Join(args, " "))

		return nil, nil
	}

	res := run(t, scene{tool: tool}, "setup", "https://bakhyard.example")
	if res.code != 0 || !strings.Contains(res.out, "settings: wrote") {
		t.Fatalf("got %+v", res)
	}

	if len(calls) != 2 || !strings.HasPrefix(calls[0], "copilot mcp get nudge") ||
		!strings.HasPrefix(calls[1], "copilot skill add ") {
		t.Errorf("calls %q", calls)
	}
}

func TestSetupStopsAtTheFirstFailure(t *testing.T) {
	res := run(t, scene{}, "setup", "http://not-https.example")
	if res.code != 1 || !strings.Contains(res.errOut, "isn't an https URL") {
		t.Errorf("got %+v", res)
	}

	tool := func(context.Context, string, ...string) ([]byte, error) {
		return nil, errors.New("copilot is not installed")
	}

	res = run(t, scene{tool: tool}, "setup", "https://bakhyard.example")
	if res.code != 1 || !strings.Contains(res.errOut, "copilot is not installed") {
		t.Errorf("got %+v", res)
	}
}
