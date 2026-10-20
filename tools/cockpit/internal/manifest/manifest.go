// Package manifest reads services.toml, the services the cockpit runs.
package manifest

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Service is one service the cockpit runs.
type Service struct {
	Name        string `toml:"name"`
	Description string `toml:"description"`
	// Command is the program and its arguments, run without a shell.
	Command   []string `toml:"command"`
	DependsOn []string `toml:"depends_on"`
}

type document struct {
	Version  int       `toml:"version"`
	Services []Service `toml:"service"`
}

const version = 1

// Names end up in file names and on the command line, so they stay plain.
var namePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// Parse reads a manifest and returns its services in start order: each one after the services
// it depends on, and otherwise in the manifest's order. Unknown keys are errors, so a typo
// can't go unnoticed.
func Parse(data []byte) ([]Service, error) {
	var doc document

	err := toml.NewDecoder(bytes.NewReader(data)).DisallowUnknownFields().Decode(&doc)
	if err != nil {
		return nil, decodeError(err)
	}

	if doc.Version != version {
		return nil, fmt.Errorf("version must be %d, not %d", version, doc.Version)
	}

	if len(doc.Services) == 0 {
		return nil, errors.New("no services")
	}

	err = validate(doc.Services)
	if err != nil {
		return nil, err
	}

	return startOrder(doc.Services)
}

// decodeError says where in the file decoding failed.
func decodeError(err error) error {
	decodeErr, ok := errors.AsType[*toml.DecodeError](err)
	if !ok {
		return err
	}

	line, _ := decodeErr.Position()
	message := strings.TrimPrefix(decodeErr.Error(), "toml: ")

	// Strict mode calls an unknown table "missing", which reads the wrong way round.
	if _, unknown := errors.AsType[*toml.StrictMissingError](err); unknown {
		message = "unknown key"
	}

	if key := decodeErr.Key(); len(key) > 0 {
		return fmt.Errorf("line %d: %s: %s", line, strings.Join(key, "."), message)
	}

	return fmt.Errorf("line %d: %s", line, message)
}

func validate(services []Service) error {
	names := make(map[string]bool, len(services))

	for index, svc := range services {
		if svc.Name == "" {
			return fmt.Errorf("service %d: no name", index+1)
		}

		err := svc.check()
		if err != nil {
			return fmt.Errorf("service %q: %w", svc.Name, err)
		}

		if names[svc.Name] {
			return fmt.Errorf("service %q: defined twice", svc.Name)
		}

		names[svc.Name] = true
	}

	for _, svc := range services {
		for _, dependency := range svc.DependsOn {
			if !names[dependency] {
				return fmt.Errorf("service %q: unknown dependency %q", svc.Name, dependency)
			}
		}
	}

	return nil
}

func (s Service) check() error {
	switch {
	case !namePattern.MatchString(s.Name):
		return errors.New("names use a-z, 0-9 and hyphens, and start with a letter")
	case strings.TrimSpace(s.Description) == "":
		return errors.New("no description")
	case len(s.Command) == 0 || s.Command[0] == "":
		return errors.New("no command")
	default:
		return nil
	}
}
