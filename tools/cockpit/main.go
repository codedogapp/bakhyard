// Command cockpit runs and watches Bakhyard's services (docs/plan.md §4.9).
package main

import (
	"fmt"
	"os"

	"github.com/codedogapp/bakhyard/tools/cockpit/internal/cli"
)

func main() {
	err := cli.Run(os.Args[1:], os.DirFS("."), os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cockpit:", err)
		os.Exit(1)
	}
}
