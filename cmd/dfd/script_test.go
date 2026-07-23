package main_test

import (
	"flag"
	"os"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"

	"github.com/bilus/dfd/internal/cli"
)

var update = flag.Bool("update", false, "rewrite txtar fixtures with actual output")

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"dfd": func() {
			os.Exit(cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
		},
	})
}

func TestScript(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir:           "testdata/script",
		UpdateScripts: *update,
	})
}
