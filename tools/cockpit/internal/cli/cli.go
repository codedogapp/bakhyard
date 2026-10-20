// Package cli runs the cockpit's commands.
package cli

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"text/tabwriter"

	"github.com/codedogapp/bakhyard/tools/cockpit/internal/manifest"
)

const (
	manifestFile = "services.toml"
	usage        = "usage: cockpit list"
	columnGap    = 2
)

// Run runs the command in args, reading the manifest from fsys and writing to stdout.
func Run(args []string, fsys fs.FS, stdout io.Writer) error {
	if len(args) != 1 || args[0] != "list" {
		return errors.New(usage)
	}

	data, err := fs.ReadFile(fsys, manifestFile)
	if err != nil {
		return fmt.Errorf("read the manifest: %w", err)
	}

	services, err := manifest.Parse(data)
	if err != nil {
		return fmt.Errorf("%s: %w", manifestFile, err)
	}

	err = list(stdout, services)
	if err != nil {
		return fmt.Errorf("print the services: %w", err)
	}

	return nil
}

// list prints the services in start order, as a table.
func list(stdout io.Writer, services []manifest.Service) error {
	table := tabwriter.NewWriter(stdout, 0, 0, columnGap, ' ', 0)

	_, err := fmt.Fprintln(table, "SERVICE\tDEPENDS ON\tCOMMAND\tDESCRIPTION")
	if err != nil {
		return err
	}

	for _, svc := range services {
		_, err = fmt.Fprintf(table, "%s\t%s\t%s\t%s\n",
			svc.Name,
			cmp.Or(strings.Join(svc.DependsOn, ", "), "-"),
			strings.Join(svc.Command, " "),
			svc.Description,
		)
		if err != nil {
			return err
		}
	}

	return table.Flush()
}
