// Package secrets reads the services' secrets by name: on the laptop, from the files of the
// Kubernetes Secrets mounted in the folder that BAKHYARD_SECRETS names, and on the Mac, where it's
// unset, from the login Keychain (decisions 79, 140 and 154).
package secrets

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// folderVar names the folder that holds the secrets' files.
const folderVar = "BAKHYARD_SECRETS"

// keychainService is the service of the login Keychain items that hold the secrets, each under
// its name as the account.
const keychainService = "bakhyard"

// Reader returns the secret name, or an error if there's no such secret, it's empty, or it can't
// be read.
type Reader func(ctx context.Context, name string) (string, error)

// New returns the Reader of the files in the folder that BAKHYARD_SECRETS names in getenv, or,
// when it's unset, of the Keychain items that password reads: keychain.Reader.Password, or a fake
// in tests.
func New(
	getenv func(string) string,
	password func(ctx context.Context, service, account string) (string, error),
) Reader {
	folder := getenv(folderVar)
	if folder == "" {
		return func(ctx context.Context, name string) (string, error) {
			return password(ctx, keychainService, name)
		}
	}

	return func(_ context.Context, name string) (string, error) {
		return readFile(folder, name)
	}
}

// readFile returns what the file name in folder holds, byte for byte, as a Secret's key holds it.
// The file may be a relative link that stays in folder, as Kubernetes mounts each key.
func readFile(folder, name string) (string, error) {
	root, err := os.OpenRoot(folder)
	if err != nil {
		return "", fmt.Errorf("open the secrets folder: %w", err)
	}
	defer func() { _ = root.Close() }()

	content, err := root.ReadFile(name)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "", fmt.Errorf("no secret file %s in %s", name, folder)
	case err != nil:
		return "", fmt.Errorf("read the secret file %s in %s: %w", name, folder, err)
	case len(content) == 0:
		return "", fmt.Errorf("the secret file %s in %s is empty", name, folder)
	}

	return string(content), nil
}
