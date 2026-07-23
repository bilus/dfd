// Package cli implements the dfd command-line interface.
package cli

import (
	"flag"
	"fmt"
	"io"
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
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if o.version {
		fmt.Fprintf(stdout, "dfd %s\n", Version)
		return 0
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(stderr, "dfd: at most one input file")
		return 1
	}
	o.input = fs.Arg(0)
	if err := run(o, stdin, stdout); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

// run is the real pipeline: read input, parse, layout, render, write.
// HOLE_1: filled across Tasks 7/10/12 as parse/layout/render land.
func run(o options, stdin io.Reader, stdout io.Writer) error {
	return fmt.Errorf("dfd: not implemented")
}
