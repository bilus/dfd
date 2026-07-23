// Command dfd renders condensed text descriptions of linear data flows
// as SVG or PNG diagrams.
package main

import (
	"os"

	"github.com/bilus/dfd/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
