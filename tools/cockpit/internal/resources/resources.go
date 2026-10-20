// Package resources measures how much CPU and memory processes use.
package resources

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Usage is how much CPU and memory a process uses.
type Usage struct {
	CPU    float64 // percent of one core, as ps reports it: on macOS, a decaying average
	Memory int64   // resident memory, in bytes
}

const (
	kibibyte = 1024 // ps's unit for resident memory
	columns  = 3    // the PID, the resident memory and the CPU
)

// Measure returns the usage of each process in pids that is still running, from one run of ps.
func Measure(ctx context.Context, pids []int) (map[int]Usage, error) {
	if len(pids) == 0 {
		return map[int]Usage{}, nil
	}

	return run(command(ctx, pids))
}

// run runs ps and reads the usage it prints.
func run(ps *exec.Cmd) (map[int]Usage, error) {
	out, err := ps.Output()
	if err == nil {
		return parse(out)
	}

	if noneRunning(out, err) {
		return map[int]Usage{}, nil
	}

	return nil, fmt.Errorf("measure resources: %w", withStderr(err))
}

// command returns ps for pids. ps by its absolute path, so no directory on PATH can stand in for
// it; separate -o flags, because POSIX makes everything after a = the header; and the C locale,
// because others can print 0,5 for 0.5.
func command(ctx context.Context, pids []int) *exec.Cmd {
	list := make([]string, 0, len(pids))
	for _, pid := range pids {
		list = append(list, strconv.Itoa(pid))
	}

	//nolint:gosec // G204: only the PIDs vary, and they are numbers the supervisor got from the OS.
	cmd := exec.CommandContext(ctx, "/bin/ps",
		"-o", "pid=", "-o", "rss=", "-o", "%cpu=", "-p", strings.Join(list, ","))
	cmd.Env = append(cmd.Environ(), "LC_ALL=C")

	return cmd
}

// noneRunning reports whether ps failed only because none of its PIDs is running, which it
// tells by exiting 1 with no output.
func noneRunning(out []byte, err error) bool {
	exitErr, ok := errors.AsType[*exec.ExitError](err)

	return ok && exitErr.ExitCode() == 1 && len(out) == 0 && len(exitErr.Stderr) == 0
}

// withStderr adds what ps wrote to its standard error, if anything, to err.
func withStderr(err error) error {
	exitErr, ok := errors.AsType[*exec.ExitError](err)
	if !ok || len(bytes.TrimSpace(exitErr.Stderr)) == 0 {
		return err
	}

	return fmt.Errorf("%w: %s", err, bytes.TrimSpace(exitErr.Stderr))
}

// parse reads ps's output: a line per process, with its PID, resident memory in KiB and CPU.
func parse(out []byte) (map[int]Usage, error) {
	usage := map[int]Usage{}

	for line := range strings.Lines(string(out)) {
		pid, use, err := parseLine(line)
		if err != nil {
			return nil, fmt.Errorf("measure resources: read %q: %w", strings.TrimSpace(line), err)
		}

		usage[pid] = use
	}

	return usage, nil
}

func parseLine(line string) (int, Usage, error) {
	fields := strings.Fields(line)
	if len(fields) != columns {
		return 0, Usage{}, errors.New("want a PID, the resident memory and the CPU")
	}

	pid, pidErr := strconv.Atoi(fields[0])
	kib, memoryErr := strconv.ParseInt(fields[1], 10, 64)
	cpu, cpuErr := strconv.ParseFloat(fields[2], 64)

	err := errors.Join(pidErr, memoryErr, cpuErr)
	if err != nil {
		return 0, Usage{}, err
	}

	return pid, Usage{CPU: cpu, Memory: kib * kibibyte}, nil
}
