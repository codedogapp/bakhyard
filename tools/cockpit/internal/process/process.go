// Package process runs a command in a process group of its own, so that stopping the command
// also stops whatever it started.
package process

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"syscall"
	"time"
)

// Process is a command that has started.
type Process struct {
	cmd  *exec.Cmd
	done chan struct{}
	err  error
}

// Start runs command, the program and its arguments, without a shell and in a new process
// group. The command's output and errors go to out, and its input is empty. command must not
// be empty.
func Start(command []string, out io.Writer) (*Process, error) {
	//nolint:gosec // G204: running the commands that services.toml lists is the cockpit's job.
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdout = out
	cmd.Stderr = out
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	proc := &Process{cmd: cmd, done: make(chan struct{})}
	go proc.wait()

	return proc, nil
}

func (p *Process) wait() {
	p.err = p.cmd.Wait()
	close(p.done)
}

// PID returns the process's ID, which is also its group's.
func (p *Process) PID() int {
	return p.cmd.Process.Pid
}

// Done returns a channel that is closed once the process has exited and all its output is
// written.
func (p *Process) Done() <-chan struct{} {
	return p.done
}

// Err returns how the process ended, once Done is closed: nil if it exited with status 0, an
// *exec.ExitError if it didn't, or the error that stopped the copying of its output.
func (p *Process) Err() error {
	return p.err
}

// Stop sends SIGTERM to the process group, then SIGKILL if the process is still running after
// timeout. It returns once the process has exited, or with an error if a signal can't be sent.
func (p *Process) Stop(timeout time.Duration) error {
	err := p.signal(syscall.SIGTERM)
	if err != nil {
		return err
	}

	select {
	case <-p.done:
		return nil
	case <-time.After(timeout):
	}

	err = p.signal(syscall.SIGKILL)
	if err != nil {
		return err
	}

	<-p.done

	return nil
}

// signal sends sig to the process group. A group that is already gone is not an error.
func (p *Process) signal(sig syscall.Signal) error {
	err := syscall.Kill(-p.PID(), sig)
	if err != nil && !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("send %v to process group %d: %w", sig, p.PID(), err)
	}

	return nil
}
