package keychain_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/internal/keychain"
)

// standIn is a fake security tool that answers by account: see testdata/security.
const standIn = "testdata/security"

func TestTheReaderReturnsThePassword(t *testing.T) {
	got, err := keychain.NewReader(standIn).Password(t.Context(), "bakhyard", "found")
	if err != nil || got != "secret of bakhyard" {
		t.Errorf("got %q, %v; want %q without the newline", got, err, "secret of bakhyard")
	}
}

func TestTheReaderFails(t *testing.T) {
	for account, want := range map[string]string{
		"missing": "no Keychain item with service bakhyard and account missing",
		"empty":   "the Keychain item with service bakhyard and account empty is empty",
		"locked": standIn + ": exit status 36: " +
			"security: SecKeychainSearchCopyNext: User interaction is not allowed.",
	} {
		got, err := keychain.NewReader(standIn).Password(t.Context(), "bakhyard", account)
		if err == nil || err.Error() != want || got != "" {
			t.Errorf("account %s: got %q, %v; want the error %q", account, got, err, want)
		}
	}
}

func TestTheReaderFailsWithoutTheTool(t *testing.T) {
	_, err := keychain.NewReader("testdata/nothing").Password(t.Context(), "bakhyard", "found")
	if err == nil || !strings.HasPrefix(err.Error(), "testdata/nothing: ") {
		t.Errorf("got %v, want an error that names the missing tool", err)
	}
}

func TestTheReaderStopsWithItsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := keychain.NewReader(standIn).Password(ctx, "bakhyard", "found")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want context.Canceled", err)
	}
}
