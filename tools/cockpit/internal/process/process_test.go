package process_test

import (
	"errors"
	"io"
	"io/fs"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/codedogapp/bakhyard/tools/cockpit/internal/process"
)

// stoppable runs until SIGTERM, then reports it. The sleep starts before the trap is set,
// since a signal that reaches a forked shell before it runs the command is lost.
const stoppable = `sleep 60 & trap "echo stopped; exit 0" TERM; echo ready; wait`

func TestStartRunsTheCommandWithoutAShell(t *testing.T) {
	out := newRecorder()

	proc, err := process.Start([]string{"echo", "$HOME", "a b"}, out)
	if err != nil {
		t.Fatal(err)
	}

	<-proc.Done()

	if proc.Err() != nil {
		t.Fatal(proc.Err())
	}

	if out.String() != "$HOME a b\n" {
		t.Errorf("got %q, want the arguments as given", out.String())
	}
}

func TestErrReportsTheExitStatus(t *testing.T) {
	const status = 3

	proc, err := process.Start([]string{"sh", "-c", "exit 3"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}

	<-proc.Done()

	exitErr, ok := errors.AsType[*exec.ExitError](proc.Err())
	if !ok || exitErr.ExitCode() != status {
		t.Errorf("got %v, want exit status %d", proc.Err(), status)
	}
}

func TestStartReportsAMissingProgram(t *testing.T) {
	_, err := process.Start([]string{"./no-such-program"}, io.Discard)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("got %v, want a missing file", err)
	}
}

func TestStopEndsTheWholeGroup(t *testing.T) {
	out := newRecorder()
	// The shell waits for a child shell that reports SIGTERM.
	command := []string{"sh", "-c", `sh -c "$1" & wait`, "sh", stoppable}

	proc, err := process.Start(command, out)
	if err != nil {
		t.Fatal(err)
	}

	out.waitFor("ready")

	group, err := syscall.Getpgid(proc.PID())
	if err != nil || group != proc.PID() {
		t.Errorf("process group %d (%v), want %d", group, err, proc.PID())
	}

	err = proc.Stop(time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out.String(), "stopped\n") {
		t.Errorf("the child didn't get SIGTERM; output %q", out.String())
	}

	if proc.Err() == nil || proc.Err().Error() != "signal: terminated" {
		t.Errorf("got %v, want SIGTERM", proc.Err())
	}
}

func TestStopKillsAProcessThatIgnoresSIGTERM(t *testing.T) {
	out := newRecorder()
	command := []string{"sh", "-c", `trap "" TERM; echo ready; while :; do sleep 1; done`}

	proc, err := process.Start(command, out)
	if err != nil {
		t.Fatal(err)
	}

	out.waitFor("ready")

	err = proc.Stop(time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	if proc.Err() == nil || proc.Err().Error() != "signal: killed" {
		t.Errorf("got %v, want SIGKILL", proc.Err())
	}
}

func TestStopAfterTheProcessExited(t *testing.T) {
	proc, err := process.Start([]string{"true"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}

	<-proc.Done()

	err = proc.Stop(time.Minute)
	if err != nil {
		t.Errorf("got %v, want nil", err)
	}
}

// recorder keeps what a process writes, and lets a test wait for a line.
type recorder struct {
	mu      sync.Mutex
	written *sync.Cond
	text    strings.Builder
}

func newRecorder() *recorder {
	rec := &recorder{}
	rec.written = sync.NewCond(&rec.mu)

	return rec
}

func (r *recorder) Write(data []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.text.Write(data)
	r.written.Broadcast()

	return len(data), nil
}

func (r *recorder) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.text.String()
}

// waitFor returns once line has been written.
func (r *recorder) waitFor(line string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for !strings.Contains(r.text.String(), line+"\n") {
		r.written.Wait()
	}
}
