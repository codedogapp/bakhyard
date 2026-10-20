package supervisor_test

import (
	"context"
	"errors"
	"io/fs"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/tools/cockpit/internal/manifest"
	"github.com/codedogapp/bakhyard/tools/cockpit/internal/supervisor"
)

// stoppable runs until SIGTERM, then reports it. The sleep starts before the trap is set,
// since a signal that reaches a forked shell before it runs the command is lost.
var stoppable = []string{
	"sh", "-c", `sleep 60 & trap "echo stopped; exit 0" TERM; echo ready; wait`,
}

func TestRunStopsTheServicesInReverseOrder(t *testing.T) {
	services := []manifest.Service{
		{Name: "home", Command: stoppable},
		{Name: "web", Command: stoppable},
		{Name: "api-gateway", Command: stoppable},
	}
	ctx, cancel := context.WithCancel(t.Context())
	out := make(lines, bufferedLines)
	result := make(chan error, 1)

	go func() {
		result <- supervisor.Run(ctx, services, out)
	}()

	var got []string

	for ready := 0; ready < len(services); {
		line := <-out
		if strings.HasSuffix(line, " | ready\n") {
			ready++
		} else {
			got = append(got, line)
		}
	}

	cancel()

	err := <-result
	if err != nil {
		t.Fatal(err)
	}

	got = append(got, out.rest()...)

	want := []string{
		"cockpit     | started home (pid N)\n",
		"cockpit     | started web (pid N)\n",
		"cockpit     | started api-gateway (pid N)\n",
		"cockpit     | stopping api-gateway\n",
		"api-gateway | stopped\n",
		"cockpit     | stopping web\n",
		"web         | stopped\n",
		"cockpit     | stopping home\n",
		"home        | stopped\n",
	}
	if !slices.Equal(withoutPIDs(got), want) {
		t.Errorf("got\n%s\nwant\n%s", strings.Join(got, ""), strings.Join(want, ""))
	}
}

func TestRunStopsTheOthersWhenAServiceExits(t *testing.T) {
	services := []manifest.Service{
		{Name: "a", Command: []string{"sleep", "60"}},
		{Name: "b", Command: []string{"sh", "-c", `echo failing; printf "last words"; exit 3`}},
	}
	out := make(lines, bufferedLines)

	err := supervisor.Run(t.Context(), services, out)
	if err == nil || err.Error() != "b exited: exit status 3" {
		t.Errorf("got %v, want b's exit", err)
	}

	got := out.rest()

	want := map[string][]string{
		"cockpit": {"started a (pid N)", "started b (pid N)", "stopping a"},
		"b":       {"failing", "last words"},
	}
	for name, texts := range want {
		if !slices.Equal(linesOf(name, got), texts) {
			t.Errorf("%s's lines: got %q, want %q", name, linesOf(name, got), texts)
		}
	}
}

func TestRunTreatsAnyExitAsAFailure(t *testing.T) {
	services := []manifest.Service{
		{Name: "a", Command: []string{"sleep", "60"}},
		{Name: "b", Command: []string{"true"}},
	}

	err := supervisor.Run(t.Context(), services, make(lines, bufferedLines))
	if err == nil || err.Error() != "b exited" {
		t.Errorf("got %v, want b's exit", err)
	}
}

func TestRunGivesEachServiceItsAddressAndItsPeers(t *testing.T) {
	// The services inherit the cockpit's environment too.
	t.Setenv("BAKHYARD_NAME", "Ada")

	// exec, so that SIGTERM reaches sleep with no shell in between.
	report := []string{
		"sh", "-c", `echo "$BAKHYARD_LISTEN [$BAKHYARD_PEERS] $BAKHYARD_NAME"; exec sleep 60`,
	}
	home, web := newHealthServer(t, true), newHealthServer(t, true)
	services := []manifest.Service{
		{Name: "home", Command: report, Listen: home.listen},
		{Name: "web", Command: report, Listen: web.listen},
		{
			Name:      "gateway",
			Command:   report,
			Listen:    "tcp://127.0.0.1:8080",
			DependsOn: []string{"home", "web"},
		},
	}
	ctx, cancel := context.WithCancel(t.Context())
	out := make(lines, bufferedLines)
	result := make(chan error, 1)

	go func() {
		result <- supervisor.Run(ctx, services, out)
	}()

	var got []string

	for reported := 0; reported < len(services); {
		line := <-out
		got = append(got, line)

		if !strings.HasPrefix(line, "cockpit ") {
			reported++
		}
	}

	cancel()

	err := <-result
	if err != nil {
		t.Fatal(err)
	}

	got = append(got, out.rest()...)

	want := map[string][]string{
		"home": {home.listen + " [] Ada"},
		"web":  {web.listen + " [] Ada"},
		"gateway": {
			"tcp://127.0.0.1:8080 [home=" + home.listen + ",web=" + web.listen + "] Ada",
		},
	}
	for name, texts := range want {
		if !slices.Equal(linesOf(name, got), texts) {
			t.Errorf("%s's lines: got %q, want %q", name, linesOf(name, got), texts)
		}
	}
}

func TestRunStopsWhatStartedWhenAServiceCannotStart(t *testing.T) {
	services := []manifest.Service{
		{Name: "a", Command: []string{"sleep", "60"}},
		{Name: "b", Command: []string{"./no-such-program"}},
		{Name: "c", Command: []string{"sleep", "60"}},
	}
	out := make(lines, bufferedLines)

	err := supervisor.Run(t.Context(), services, out)
	if !errors.Is(err, fs.ErrNotExist) || !strings.HasPrefix(err.Error(), "start b: ") {
		t.Errorf("got %v, want b's start to fail", err)
	}

	got := withoutPIDs(out.rest())
	want := []string{"cockpit | started a (pid N)\n", "cockpit | stopping a\n"}

	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRunReportsAFailedWrite(t *testing.T) {
	services := []manifest.Service{{Name: "a", Command: []string{"sleep", "60"}}}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := supervisor.Run(ctx, services, failingWriter{})
	if !errors.Is(err, errWrite) || err.Error() != "write the output: disk full" {
		t.Errorf("got %v, want the write error", err)
	}
}

// bufferedLines is more than any test writes, so Run never waits for a test to read.
const bufferedLines = 100

// lines receives what Run writes, which is a whole line per write.
type lines chan string

func (l lines) Write(data []byte) (int, error) {
	l <- string(data)

	return len(data), nil
}

// rest returns the lines not read yet. Call it once Run has returned.
func (l lines) rest() []string {
	close(l)

	var rest []string
	for line := range l {
		rest = append(rest, line)
	}

	return rest
}

var pid = regexp.MustCompile(`pid \d+`)

func withoutPIDs(got []string) []string {
	clean := make([]string, len(got))
	for index, line := range got {
		clean[index] = pid.ReplaceAllString(line, "pid N")
	}

	return clean
}

// linesOf returns the texts of name's lines, in order and without pids.
func linesOf(name string, got []string) []string {
	var texts []string

	for _, line := range withoutPIDs(got) {
		marker, text, _ := strings.Cut(strings.TrimSuffix(line, "\n"), " | ")
		if strings.TrimSpace(marker) == name {
			texts = append(texts, text)
		}
	}

	return texts
}

var errWrite = errors.New("disk full")

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errWrite
}
