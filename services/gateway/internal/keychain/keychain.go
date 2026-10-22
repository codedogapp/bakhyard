// Package keychain reads passwords from the user's login Keychain with macOS's security tool, so
// a secret never sits in a file or in the environment (decision 79).
package keychain

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Security is the path of macOS's security tool.
const Security = "/usr/bin/security"

// itemNotFound is the exit code of security when the Keychain has no matching item.
const itemNotFound = 44

// Reader reads passwords by running a security tool: Security, or a stand-in in tests.
type Reader struct {
	program string
}

// NewReader returns a Reader that runs program.
func NewReader(program string) Reader {
	return Reader{program: program}
}

// Password returns the password of the generic password item with service and account. It
// fails if the item is missing or empty, or if the security tool fails.
func (r Reader) Password(ctx context.Context, service, account string) (string, error) {
	// security prints the password and a newline.
	//nolint:gosec // G204: no shell runs, and the program and item names come from our code.
	out, err := exec.CommandContext(
		ctx, r.program, "find-generic-password", "-s", service, "-a", account, "-w",
	).Output()
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		if exitErr.ExitCode() == itemNotFound {
			return "", fmt.Errorf("no Keychain item with service %s and account %s",
				service, account)
		}

		return "", fmt.Errorf("%s: %w: %s",
			r.program, err, strings.TrimSpace(string(exitErr.Stderr)))
	}

	if err != nil {
		return "", fmt.Errorf("%s: %w", r.program, err)
	}

	secret := strings.TrimSuffix(string(out), "\n")
	if secret == "" {
		return "", fmt.Errorf("the Keychain item with service %s and account %s is empty",
			service, account)
	}

	return secret, nil
}
