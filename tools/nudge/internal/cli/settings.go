package cli

import (
	"cmp"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/codedogapp/bakhyard/tools/nudge/internal/door"
)

const serverVar = "NUDGE_SERVER"

// settingsFile is the settings file's path: nudge/settings.json in the config folder.
func settingsFile(env Env) (string, error) {
	config, err := env.Config()
	if err != nil {
		return "", fmt.Errorf("find the config folder: %w", err)
	}

	return filepath.Join(config, "nudge", "settings.json"), nil
}

// connect returns the client of the server that NUDGE_SERVER names, else the settings file's
// "server". A missing file and members it doesn't know, such as an older nudge's, are fine.
func connect(env Env) (*door.Client, error) {
	path, err := settingsFile(env)
	if err != nil {
		return nil, err
	}

	var file struct {
		Server string `json:"server"`
	}

	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("read the settings file: %w", err)
	}

	if err == nil {
		if err := json.Unmarshal(data, &file); err != nil {
			return nil, fmt.Errorf("the settings file %s is invalid: %w", path, err)
		}
	}

	server := cmp.Or(env.Getenv(serverVar), file.Server)
	if server == "" {
		return nil, fmt.Errorf("no server: set %s, or run nudge setup <server>", serverVar)
	}

	client, err := door.New(server)
	if err != nil {
		return nil, err
	}

	if env.HTTP != nil {
		client.HTTP = env.HTTP
	}

	return client, nil
}

// project names the agent's work for the user: the folder it runs in and this machine, so that
// two machines' sessions in folders of one name stay apart.
func project(env Env) string {
	dir, err := os.Getwd()
	if err != nil {
		dir = "agent"
	}

	host, err := env.Host()
	if err != nil || host == "" {
		return filepath.Base(dir)
	}

	short, _, _ := strings.Cut(host, ".")

	return fmt.Sprintf("%s (%s)", filepath.Base(dir), short)
}
