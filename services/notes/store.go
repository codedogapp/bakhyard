package main

import (
	"context"
	"fmt"

	"github.com/codedogapp/bakhyard/internal/env"
	"github.com/codedogapp/bakhyard/internal/secrets"
	"github.com/codedogapp/bakhyard/services/notes/internal/api"
	"github.com/codedogapp/bakhyard/services/notes/internal/gitea"
)

// repoVar is the setting of the vault's repository in Gitea's API, such as
// https://<gitea>/api/v1/repos/<owner>/<repo>, through which notes saves notes (docs/setup.md).
// notes is read-only when it isn't set (decision 420).
const repoVar = "BAKHYARD_VAULT_REPO"

// tokenAccount is the Keychain account of the token of Gitea's bot user, which commits the saves,
// and the name of its file (decision 420).
const tokenAccount = "vault-write-token"

// vaultStore returns the vault's repository in the settings, which notes reaches with the token
// that readSecret reads, or nil when the repository isn't set.
func vaultStore(
	ctx context.Context, getenv func(string) string, readSecret secrets.Reader,
) (api.Store, error) {
	if getenv(repoVar) == "" {
		return nil, nil
	}

	repo, err := env.SecureURL(getenv, repoVar)
	if err != nil {
		return nil, err
	}

	token, err := readSecret(ctx, tokenAccount)
	if err != nil {
		return nil, fmt.Errorf("read the vault's write token: %w", err)
	}

	return gitea.New(gitea.Settings{Repo: repo, Token: token}), nil
}
