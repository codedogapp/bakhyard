package resources

import (
	"maps"
	"os"
	"os/exec"
	"testing"
)

func TestMeasureMeasuresTheProcessesStillRunning(t *testing.T) {
	usage, err := Measure(t.Context(), []int{os.Getpid(), exited(t)})
	if err != nil {
		t.Fatal(err)
	}

	own, ok := usage[os.Getpid()]
	if len(usage) != 1 || !ok || own.Memory <= 0 || own.CPU < 0 {
		t.Errorf("got %+v, want this test's usage alone", usage)
	}
}

func TestMeasureWithNoneRunningIsEmpty(t *testing.T) {
	usage, err := Measure(t.Context(), []int{exited(t)})
	if err != nil || len(usage) != 0 {
		t.Errorf("got %v and %v, want nothing", usage, err)
	}
}

func TestMeasureWithNoPIDsIsEmpty(t *testing.T) {
	usage, err := Measure(t.Context(), nil) // ps itself fails on an empty list
	if err != nil || len(usage) != 0 {
		t.Errorf("got %v and %v, want nothing", usage, err)
	}
}

func TestMeasureIgnoresTheLocale(t *testing.T) {
	t.Setenv("LC_ALL", "de_DE.UTF-8") // in which ps on macOS prints 0,5 for 0.5

	_, err := Measure(t.Context(), []int{os.Getpid()})
	if err != nil {
		t.Error(err)
	}
}

func TestRunReadsEveryLine(t *testing.T) {
	usage, err := run(exec.Command("printf", `   42  2048   0.5\n    7     1  12.0\n`))
	if err != nil {
		t.Fatal(err)
	}

	want := map[int]Usage{42: {CPU: 0.5, Memory: 2048 * 1024}, 7: {CPU: 12, Memory: 1024}}
	if !maps.Equal(usage, want) {
		t.Errorf("got %+v, want %+v", usage, want)
	}
}

func TestRunWithNoOutputAndExit1IsEmpty(t *testing.T) {
	usage, err := run(exec.Command("false"))
	if err != nil || usage == nil || len(usage) != 0 {
		t.Errorf("got %v and %v, want an empty map", usage, err)
	}
}

func TestRunFailures(t *testing.T) {
	for _, test := range []struct {
		name string
		ps   *exec.Cmd
		want string
	}{
		{
			"with an error",
			exec.Command("sh", "-c", "echo 'ps: no' >&2; exit 1"),
			"measure resources: exit status 1: ps: no",
		},
		{
			"silently",
			exec.Command("sh", "-c", "exit 2"),
			"measure resources: exit status 2",
		},
		{
			"to start",
			exec.Command("/nonexistent"),
			"measure resources: fork/exec /nonexistent: no such file or directory",
		},
		{
			"with too few columns",
			exec.Command("echo", "42 2048"),
			`measure resources: read "42 2048": want a PID, the resident memory and the CPU`,
		},
		{
			"with a word",
			exec.Command("echo", "42 2048 x"),
			`measure resources: read "42 2048 x": strconv.ParseFloat: parsing "x": invalid syntax`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			usage, err := run(test.ps)
			if err == nil || err.Error() != test.want || usage != nil {
				t.Errorf("got %v and %v, want %q", usage, err, test.want)
			}
		})
	}
}

// exited returns the PID of a process that has exited.
func exited(t *testing.T) int {
	t.Helper()

	cmd := exec.Command("true")

	err := cmd.Run()
	if err != nil {
		t.Fatal(err)
	}

	return cmd.Process.Pid
}
