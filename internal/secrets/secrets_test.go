package secrets_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/internal/secrets"
)

func TestTheSecretsComeFromTheKeychainWithoutAFolder(t *testing.T) {
	ctx := t.Context()
	locked := errors.New("the Keychain is locked")

	var asked []string

	read := secrets.New(
		folderAt(""),
		func(got context.Context, service, account string) (string, error) {
			if got != ctx {
				t.Error("the Keychain got another context")
			}

			asked = append(asked, service+"/"+account)
			if account == "locked" {
				return "", locked
			}

			return "secret of " + account, nil
		},
	)

	got, err := read(ctx, "jira-token")
	if err != nil || got != "secret of jira-token" {
		t.Errorf("got %q, %v; want %q", got, err, "secret of jira-token")
	}

	_, err = read(ctx, "locked")
	if !errors.Is(err, locked) {
		t.Errorf("got %v, want %v", err, locked)
	}

	if want := []string{"bakhyard/jira-token", "bakhyard/locked"}; !slices.Equal(asked, want) {
		t.Errorf("asked the Keychain for %v, want %v", asked, want)
	}
}

func TestTheSecretsComeFromTheFolder(t *testing.T) {
	folder := t.TempDir()
	// Nothing is trimmed: the file holds the secret as the Secret's key does.
	write(t, filepath.Join(folder, "postgres-todos"), " s3cret\n")

	got, err := secrets.New(folderAt(folder), noKeychain(t))(t.Context(), "postgres-todos")
	if err != nil || got != " s3cret\n" {
		t.Errorf("got %q, %v; want %q", got, err, " s3cret\n")
	}
}

func TestTheSecretsFollowKubernetesLinks(t *testing.T) {
	// Kubernetes mounts each key as a link to ..data/<key>, and ..data as a link to the folder of
	// the Secret's current version, both relative.
	folder := t.TempDir()
	version := "..2026_10_28_09_41_00.123456789"

	err := os.Mkdir(filepath.Join(folder, version), 0o700)
	if err != nil {
		t.Fatal(err)
	}

	write(t, filepath.Join(folder, version, "jira-token"), "token")
	link(t, version, filepath.Join(folder, "..data"))
	link(t, filepath.Join("..data", "jira-token"), filepath.Join(folder, "jira-token"))

	got, err := secrets.New(folderAt(folder), noKeychain(t))(t.Context(), "jira-token")
	if err != nil || got != "token" {
		t.Errorf("got %q, %v; want %q", got, err, "token")
	}
}

func TestTheSecretsStayInTheFolder(t *testing.T) {
	parent := t.TempDir()
	outside := filepath.Join(parent, "jira-token")
	folder := filepath.Join(parent, "secrets")

	err := os.Mkdir(folder, 0o700)
	if err != nil {
		t.Fatal(err)
	}

	write(t, outside, "outside")
	link(t, filepath.Join("..", "jira-token"), filepath.Join(folder, "relative"))
	link(t, outside, filepath.Join(folder, "absolute"))

	read := secrets.New(folderAt(folder), noKeychain(t))
	for _, name := range []string{"relative", "absolute", "../jira-token", outside} {
		got, err := read(t.Context(), name)

		want := "read the secret file " + name + " in " + folder + ": "
		if err == nil || !strings.HasPrefix(err.Error(), want) || got != "" {
			t.Errorf("%s: got %q, %v; want an error that starts with %s", name, got, err, want)
		}
	}
}

func TestReadingASecretFileFails(t *testing.T) {
	folder := t.TempDir()
	write(t, filepath.Join(folder, "empty"), "")
	write(t, filepath.Join(folder, "file"), "secret")
	link(t, "nowhere", filepath.Join(folder, "dangling"))

	err := os.Mkdir(filepath.Join(folder, "folder"), 0o700)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct{ folder, name, want string }{
		{folder, "missing", "no secret file missing in " + folder},
		{folder, "dangling", "no secret file dangling in " + folder},
		{folder, "empty", "the secret file empty in " + folder + " is empty"},
		{folder, "folder", "read the secret file folder in " + folder + ": "},
		{filepath.Join(folder, "missing"), "file", "open the secrets folder: "},
		{filepath.Join(folder, "file"), "file", "open the secrets folder: "},
	} {
		got, err := secrets.New(folderAt(test.folder), noKeychain(t))(t.Context(), test.name)
		if err == nil || !strings.HasPrefix(err.Error(), test.want) || got != "" {
			t.Errorf("%s in %s: got %q, %v; want an error that starts with %s",
				test.name, test.folder, got, err, test.want)
		}
	}
}

// folderAt returns an environment whose BAKHYARD_SECRETS is folder.
func folderAt(folder string) func(string) string {
	return func(name string) string { return map[string]string{"BAKHYARD_SECRETS": folder}[name] }
}

// noKeychain returns a Keychain reader that fails the test when it's read.
func noKeychain(t *testing.T) func(context.Context, string, string) (string, error) {
	t.Helper()

	return func(context.Context, string, string) (string, error) {
		t.Error("the Keychain was read")

		return "", errors.New("no Keychain")
	}
}

// write writes content into the file at path.
func write(t *testing.T, path, content string) {
	t.Helper()

	err := os.WriteFile(path, []byte(content), 0o600)
	if err != nil {
		t.Fatal(err)
	}
}

// link makes a link at path to target.
func link(t *testing.T, target, path string) {
	t.Helper()

	err := os.Symlink(target, path)
	if err != nil {
		t.Fatal(err)
	}
}
