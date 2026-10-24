package vault_test

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/codedogapp/bakhyard/services/notes/internal/vault"
)

func TestScanListsEveryFileOfTheVault(t *testing.T) {
	root := vaultWith(t,
		"Inbox.md", "Daily/2026-10-24.md", "Projects/Deep/Plan.md", "img/photo.png",
	)

	files, err := vault.Scan(root.FS())
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"Daily/2026-10-24.md", "Inbox.md", "Projects/Deep/Plan.md", "img/photo.png"}
	if !slices.Equal(files, want) {
		t.Errorf("got %q, want %q", files, want)
	}
}

func TestScanSkipsDotFoldersAndDotFiles(t *testing.T) {
	root := vaultWith(t,
		"Note.md", ".DS_Store", ".obsidian/app.json", ".obsidian/plugins/sync/data.json",
		".git/config", ".trash/Old.md", "Daily/.hidden.md", "Daily/.cache/Draft.md",
	)

	files, err := vault.Scan(root.FS())
	if err != nil {
		t.Fatal(err)
	}

	if want := []string{"Note.md"}; !slices.Equal(files, want) {
		t.Errorf("got %q, want %q", files, want)
	}
}

func TestScanSkipsNamesThatArentUTF8(t *testing.T) {
	// macOS refuses such names, but Linux allows them, so a map stands in for the disk.
	fsys := fstest.MapFS{
		"Note.md":          {},
		"Bad\xff.md":       {},
		"Bad\xff/Inner.md": {},
		"Daily/Today.md":   {},
		"Daily/Bad\xfe.md": {},
	}

	files, err := vault.Scan(fsys)
	if err != nil {
		t.Fatal(err)
	}

	if want := []string{"Daily/Today.md", "Note.md"}; !slices.Equal(files, want) {
		t.Errorf("got %q, want %q", files, want)
	}
}

func TestScanListsNoSymlink(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "secret.md")
	if err := os.WriteFile(outside, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}

	root := vaultWith(t, "Note.md", "Daily/Today.md")
	for name, target := range map[string]string{
		"Link.md":    "Note.md",
		"Folder":     "Daily",
		"Outside.md": outside,
		"Gone.md":    "Missing.md",
	} {
		if err := root.Symlink(target, name); err != nil {
			t.Fatal(err)
		}
	}

	files, err := vault.Scan(root.FS())
	if err != nil {
		t.Fatal(err)
	}

	if want := []string{"Daily/Today.md", "Note.md"}; !slices.Equal(files, want) {
		t.Errorf("got %q, want %q", files, want)
	}
}

func TestScanOfAnEmptyVaultListsNothing(t *testing.T) {
	root := vaultWith(t, ".obsidian/app.json")

	files, err := vault.Scan(root.FS())
	if err != nil || len(files) != 0 {
		t.Errorf("got %q, %v; want no files and no error", files, err)
	}
}

func TestScanFailsWhenTheVaultCantBeRead(t *testing.T) {
	files, err := vault.Scan(unreadable{})

	if files != nil || !errors.Is(err, fs.ErrPermission) ||
		err.Error() != "scan the vault: open .: permission denied" {
		t.Errorf("got %q, %v; want no files and the permission error with its context", files, err)
	}
}

// unreadable is a file system that refuses every open.
type unreadable struct{}

func (unreadable) Open(name string) (fs.File, error) {
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrPermission}
}

// vaultWith returns a vault, opened as a root, that holds the files names, each with some text.
func vaultWith(t *testing.T, names ...string) *os.Root {
	t.Helper()

	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	})

	for _, name := range names {
		if err := root.MkdirAll(path.Dir(name), 0o700); err != nil {
			t.Fatal(err)
		}

		if err := root.WriteFile(name, []byte("# "+name), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	return root
}
