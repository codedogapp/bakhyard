package main

import (
	"fmt"
	"strings"

	"github.com/codedogapp/bakhyard/internal/weblink"
)

// historyVar is the setting of the web address of the vault's history in Gitea, such as
// https://<gitea>/<owner>/<repo>/commits/branch/main, after which a note's path gives the note's
// own history, which its page links to (decision 425). The pages have no such link without it.
const historyVar = "BAKHYARD_VAULT_HISTORY"

// vaultHistory returns the web address of the vault's history in the settings, without a slash at
// its end, or an empty string when it isn't set. Like every link that the app shows, it must be an
// https URL (decision 201), and with no query or fragment, which a note's path can't follow.
func vaultHistory(getenv func(string) string) (string, error) {
	value := getenv(historyVar)
	if value == "" {
		return "", nil
	}

	if weblink.HTTPS(value) == "" || strings.ContainsAny(value, "?#") {
		return "", fmt.Errorf("%s: %q is not an https URL with no query or fragment",
			historyVar, value)
	}

	return strings.TrimRight(value, "/"), nil
}
