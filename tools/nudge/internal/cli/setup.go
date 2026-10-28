package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/codedogapp/bakhyard/tools/nudge/internal/setup"
)

const setupUsage = "usage: nudge setup SERVER   (SERVER is Bakhyard's https URL)"

// runSetup points this machine at the server, adds nudge to Copilot CLI, prints what each step
// did, and stops at the first that fails.
func runSetup(ctx context.Context, args []string, env Env) int {
	if len(args) != 1 {
		_, _ = fmt.Fprintln(env.Stderr, setupUsage)

		return exitUsage
	}

	settings, err := settingsFile(env)
	if err != nil {
		return fail(env, "setup", err)
	}

	bin, err := os.Executable()
	if err != nil {
		return fail(env, "setup", fmt.Errorf("find this program: %w", err))
	}

	run := env.Run
	if run == nil {
		run = setup.Exec
	}

	skills := filepath.Join(filepath.Dir(settings), "skills")

	steps := []func() ([]string, error){
		func() ([]string, error) { return one(setup.WriteSettings(settings, args[0])) },
		func() ([]string, error) { return nil, setup.WriteSkill(skills) },
		func() ([]string, error) { return setup.AddToCopilot(ctx, run, bin, skills) },
	}

	for _, step := range steps {
		lines, err := step()
		for _, line := range lines {
			_, _ = fmt.Fprintln(env.Stdout, line)
		}

		if err != nil {
			return fail(env, "setup", err)
		}
	}

	_, _ = fmt.Fprintln(env.Stdout, "Start a Copilot session with /nudge to ask on the phone.")

	return exitOK
}

func one(line string, err error) ([]string, error) {
	return []string{line}, err
}
