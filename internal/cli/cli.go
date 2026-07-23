// Package cli implements the dfd command-line interface.
package cli

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/bilus/dfd/layout"
	"github.com/bilus/dfd/parse"
	"github.com/bilus/dfd/render"
)

const Version = "0.1.0"

type options struct {
	out      string
	format   string
	box      string
	maxWidth int
	perRow   int
	fontSize int
	scale    float64
	version  bool
	input    string // positional; "" = stdin
}

func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dfd", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var o options
	fs.StringVar(&o.out, "o", "", "output file (.svg/.png); \"-\" = stdout")
	fs.StringVar(&o.format, "format", "", "svg or png; overrides -o extension")
	fs.StringVar(&o.box, "box", "160x60", "box size WxH")
	fs.IntVar(&o.maxWidth, "max-width", 1000, "target canvas width")
	fs.IntVar(&o.perRow, "per-row", 0, "fixed boxes per row (overrides --max-width)")
	fs.IntVar(&o.fontSize, "font-size", 13, "label font size")
	fs.Float64Var(&o.scale, "scale", 2, "PNG resolution multiplier")
	fs.BoolVar(&o.version, "version", false, "print version")
	// Accept flags both before and after the positional input path.
	var positionals []string
	for {
		if err := fs.Parse(args); err != nil {
			return 1
		}
		args = fs.Args()
		if len(args) == 0 {
			break
		}
		positionals = append(positionals, args[0])
		args = args[1:]
	}
	if o.version {
		fmt.Fprintf(stdout, "dfd %s\n", Version)
		return 0
	}
	if len(positionals) > 1 {
		fmt.Fprintln(stderr, "dfd: at most one input file")
		return 1
	}
	if len(positionals) == 1 {
		o.input = positionals[0]
	}
	if err := run(o, stdin, stdout); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

// run is the real pipeline: read input, parse, layout, render, write.
// stdin/stdout modes and PNG arrive in later iterations.
func run(o options, stdin io.Reader, stdout io.Writer) (err error) {
	if o.input == "" || o.out == "" || o.out == "-" {
		return fmt.Errorf("dfd: only file-to-file rendering is implemented; pass input.dfd and -o out.svg")
	}
	f, err := os.Open(o.input)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); err == nil && cerr != nil {
			err = cerr
		}
	}()
	d, err := parse.Parse(f, o.input)
	if err != nil {
		return err
	}
	var boxW, boxH int
	if _, err := fmt.Sscanf(o.box, "%dx%d", &boxW, &boxH); err != nil {
		return fmt.Errorf("dfd: invalid --box %q (want WxH)", o.box)
	}
	scene, err := layout.Arrange(d, layout.Config{
		BoxW: boxW, BoxH: boxH,
		MaxWidth: o.maxWidth, PerRow: o.perRow,
		FontSize: o.fontSize,
	})
	if err != nil {
		return err
	}
	out, err := os.Create(o.out)
	if err != nil {
		return err
	}
	if err := render.SVG(scene, out); err != nil {
		if cerr := out.Close(); cerr != nil {
			return fmt.Errorf("%v; close: %v", err, cerr)
		}
		return err
	}
	return out.Close()
}
