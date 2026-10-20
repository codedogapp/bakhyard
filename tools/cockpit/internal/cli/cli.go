// Package cli runs the cockpit's commands.
package cli

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"text/tabwriter"

	"github.com/codedogapp/bakhyard/tools/cockpit/internal/manifest"
	"github.com/codedogapp/bakhyard/tools/cockpit/internal/supervisor"
)

const (
	manifestFile = "services.toml"
	usage        = "usage: cockpit list|up"
	columnGap    = 2
)

// Run runs the command in args, reading the manifest from fsys and writing to stdout. up runs
// until ctx ends.
func Run(ctx context.Context, args []string, fsys fs.FS, stdout io.Writer) error {
	if len(args) != 1 {
		return errors.New(usage)
	}

	switch args[0] {
	case "list":
		return list(fsys, stdout)
	case "up":
		return up(ctx, fsys, stdout)
	default:
		return errors.New(usage)
	}
}

// list prints the services in start order.
func list(fsys fs.FS, stdout io.Writer) error {
	services, err := readManifest(fsys)
	if err != nil {
		return err
	}

	err = printTable(stdout, services)
	if err != nil {
		return fmt.Errorf("print the services: %w", err)
	}

	return nil
}

// up runs every service in the foreground until ctx ends or one of them exits.
func up(ctx context.Context, fsys fs.FS, stdout io.Writer) error {
	services, err := readManifest(fsys)
	if err != nil {
		return err
	}

	return supervisor.Run(ctx, services, stdout)
}

// readManifest returns the services in the manifest, in start order.
func readManifest(fsys fs.FS) ([]manifest.Service, error) {
	data, err := fs.ReadFile(fsys, manifestFile)
	if err != nil {
		return nil, fmt.Errorf("read the manifest: %w", err)
	}

	services, err := manifest.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", manifestFile, err)
	}

	return services, nil
}

// printTable prints the services as a table.
func printTable(stdout io.Writer, services []manifest.Service) error {
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
