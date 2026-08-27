// Package cli implements the dfd command-line interface.
package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bilus/dfd/layout"
	"github.com/bilus/dfd/parse"
	"github.com/bilus/dfd/render"
	"github.com/bilus/dfd/theme"
)

const Version = "0.1.0"

type options struct {
	out      string
	format   string
	box      string
	perRow   int
	fontSize int
	scale    float64
	theme    string
	number   bool
	prefix   string
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
	fs.IntVar(&o.perRow, "per-row", layout.DefaultPerRow, "boxes per row")
	fs.IntVar(&o.fontSize, "font-size", 13, "label font size")
	fs.Float64Var(&o.scale, "scale", 2, "PNG resolution multiplier")
	fs.StringVar(&o.theme, "theme", "default", "visual theme: "+strings.Join(theme.Names(), " or "))
	fs.BoolVar(&o.number, "number", false, "number processes 1, 2, 3 and datastores D1, D2")
	fs.StringVar(&o.prefix, "number-prefix", "", "prefix for process numbers, e.g. \"2.\" for a levelled diagram")
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
// PNG arrives in a later iteration.
func run(o options, stdin io.Reader, stdout io.Writer) (err error) {
	src := stdin
	name := "<stdin>"
	if o.input != "" {
		f, err := os.Open(o.input)
		if err != nil {
			return err
		}
		defer func() {
			if cerr := f.Close(); err == nil && cerr != nil {
				err = cerr
			}
		}()
		src, name = f, o.input
	}
	d, err := parse.Parse(src, name)
	if err != nil {
		return err
	}
	var boxW, boxH int
	if _, err := fmt.Sscanf(o.box, "%dx%d", &boxW, &boxH); err != nil {
		return fmt.Errorf("dfd: invalid --box %q (want WxH)", o.box)
	}
	th, err := theme.Lookup(o.theme, o.fontSize)
	if err != nil {
		return fmt.Errorf("dfd: invalid --theme %q (want %s)", o.theme, strings.Join(theme.Names(), " or "))
	}
	scene, err := layout.Arrange(d, layout.Config{
		BoxW: boxW, BoxH: boxH,
		PerRow:   o.perRow,
		FontSize: o.fontSize, Theme: th,
		Number: o.number, NumberPrefix: o.prefix,
	})
	if err != nil {
		return err
	}
	format, outPath, err := resolveOutput(o)
	if err != nil {
		return err
	}
	draw := func(w io.Writer) error {
		if format == "png" {
			return render.PNG(scene, th, o.scale, w)
		}
		return render.SVG(scene, th, w)
	}
	if outPath == "" {
		return draw(stdout)
	}
	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	if err := draw(out); err != nil {
		if cerr := out.Close(); cerr != nil {
			return fmt.Errorf("%v; close: %v", err, cerr)
		}
		return err
	}
	return out.Close()
}

// resolveOutput decides the output format and destination ("" = stdout)
// from -o, --format, and the input path.
func resolveOutput(o options) (format, path string, err error) {
	switch o.format {
	case "", "svg", "png":
	default:
		return "", "", fmt.Errorf("dfd: invalid --format %q (want svg or png)", o.format)
	}
	format = o.format
	path = o.out
	if path == "-" {
		path = ""
	}
	if path != "" && format == "" {
		switch ext := filepath.Ext(path); ext {
		case ".svg":
			format = "svg"
		case ".png":
			format = "png"
		default:
			return "", "", fmt.Errorf("dfd: cannot infer format from %q; use --format", ext)
		}
	}
	if format == "" {
		format = "svg"
	}
	if o.out == "" && o.input != "" {
		ext := ".svg"
		if format == "png" {
			ext = ".png"
		}
		path = strings.TrimSuffix(o.input, filepath.Ext(o.input)) + ext
	}
	if path == "" && format == "png" {
		return "", "", fmt.Errorf("dfd: refusing to write binary PNG to stdout; use -o")
	}
	return format, path, nil
}
