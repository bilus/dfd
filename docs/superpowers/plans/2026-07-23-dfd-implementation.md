# dfd Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `dfd`, a CLI that turns condensed text (`[process]`, `|store|`, `>`/`<` arrow lines) into deterministic SVG/PNG data-flow diagrams with auto-snake layout.

**Architecture:** Outside-in pipeline per the approved spec ([../superpowers/specs/2026-07-23-text-to-dfd-design.md](../specs/2026-07-23-text-to-dfd-design.md)): `parse` → `ast.Diagram` → `layout.Scene` (display list) → `render.SVG`/`render.PNG`, wired by `internal/cli.Run`, thin `cmd/dfd`. Every feature starts with a testscript acceptance test; unit TDD underneath.

**Tech Stack:** Go ≥1.22, `rogpeppe/go-internal/testscript` (tests only), `fogleman/gg` + `golang.org/x/image` (`gofont/goregular`, `opentype`) for PNG and text metrics.

## Execution pivot (2026-07-23, after Task 5)

Tasks 6–13 below are superseded by user direction: iterate **outside-in** in
minimal feature increments instead of layer-by-layer. Each iteration: (1) add a
failing testscript acceptance script for the smallest next feature, (2) make it
pass through inner unit-TDD cycles, (3) commit. Ladder: one box → two boxes +
arrow → labeled arrows → store write → store read/write → multiple stores →
snake rows + flags → title wrapping → IO modes → error scripts → PNG →
original example + README. Tasks 1–5 (harness, ast, parse) were completed
before the pivot and stand. The code blocks in Tasks 6–13 remain valid
reference material for the target implementations; only the sequencing changed.

## Global Constraints

- Module `github.com/bilus/dfd`; repo root `/Users/bilus/dev/bilus/dfd`; all paths below relative to it.
- **Acceptance-first:** every feature task's Step 1 adds a txtar script under `cmd/dfd/testdata/script/` and runs it RED before any implementation. `want.svg` sections start empty and are seeded with `go test ./cmd/dfd -run TestScript -update` only after the slice passes; visually review seeded SVG before committing.
- **TDD + holes:** for each implementation step, first write signatures with `panic("HOLE: <contract>")` bodies (hole-driven-development-iterative-reasoning skill), fill holes most-constrained-first; unit test RED before filling.
- All tests in external test packages (`package foo_test`). Never discard errors with `_`. Function types get `Fn` suffix (none planned).
- Commit messages: plain imperative, no Co-Authored-By or any AI trailers.
- Only these deps: `github.com/rogpeppe/go-internal` (test), `github.com/fogleman/gg`, `golang.org/x/image` (+ transitive). No cgo.
- Determinism: no `time.Now`/`math/rand` anywhere; integer geometry; fixed SVG formatting; embedded font only.
- Normative geometry constants (spec table): box 160×60 min, hGap 90, vGap 90, margin 40, storeW 150 (widens: name width+20), storeH 36, storeArrow 64, storeGap 20, boxPad 12, labelGap 8, font 13px, arrowhead inset 3, title line height 17.
- Every task ends: `gofmt -l .` empty, `go vet ./...` clean, `go test ./...` green, then commit.

---

### Task 1: Module scaffold + testscript harness

**Files:**
- Create: `go.mod`, `.gitignore`, `cmd/dfd/main.go`, `internal/cli/cli.go`, `cmd/dfd/script_test.go`, `cmd/dfd/testdata/script/version.txtar`

**Interfaces:**
- Produces: `cli.Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int` — flag parsing only; real work errors with "not implemented". `const cli.Version = "0.1.0"`.

- [ ] **Step 1: Write the failing acceptance test**

`cmd/dfd/testdata/script/version.txtar`:

```
dfd --version
stdout '^dfd 0\.1\.0$'

! dfd --definitely-not-a-flag
stderr 'flag provided but not defined'
```

`cmd/dfd/script_test.go`:

```go
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
```

- [ ] **Step 2: Scaffold module and stubs; run test to verify RED**

```bash
cd /Users/bilus/dev/bilus/dfd
go mod init github.com/bilus/dfd
```

`.gitignore`:

```
/dfd
```

`cmd/dfd/main.go`:

```go
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
```

`internal/cli/cli.go` (flags only; `run` is a hole):

```go
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

func run(o options, stdin io.Reader, stdout io.Writer) error {
	return fmt.Errorf("dfd: not implemented")
}
```

```bash
go get github.com/rogpeppe/go-internal/testscript
go test ./cmd/dfd
```

Expected: FAIL until stubs compile, then PASS (both script assertions hold with flags-only Run). If `testscript.Main`'s signature differs in the fetched version (older releases use `RunMain(m, map[string]func() int)`), adapt to the compiler error — that is the only sanctioned deviation.

- [ ] **Step 3: Verify green + hygiene**

```bash
gofmt -l . && go vet ./... && go test ./...
```

Expected: no gofmt output, vet clean, `ok github.com/bilus/dfd/cmd/dfd`.

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "Scaffold dfd module with testscript acceptance harness"
```

---

### Task 2: ast package

**Files:**
- Create: `ast/ast.go`, Test: `ast/ast_test.go`

**Interfaces:**
- Produces: `ast.Diagram{Steps []Step}`, `ast.Step{Title string; In string; Stores []StoreLink}`, `ast.StoreLink{Name string; Put, Get *Arrow}`, `ast.Arrow{Label string}`, `ast.New(steps []Step) (*Diagram, error)`, `ast.NewStoreLink(name string, put, get *Arrow) (StoreLink, error)`.

- [ ] **Step 1: Write failing unit tests**

`ast/ast_test.go`:

```go
package ast_test

import (
	"testing"

	"github.com/bilus/dfd/ast"
)

func TestNewValidDiagram(t *testing.T) {
	d, err := ast.New([]ast.Step{{Title: "A"}, {Title: "B", In: "label"}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if len(d.Steps) != 2 || d.Steps[1].In != "label" {
		t.Fatalf("unexpected diagram: %+v", d)
	}
}

func TestNewRejectsInvalid(t *testing.T) {
	cases := []struct {
		name  string
		steps []ast.Step
	}{
		{"empty", nil},
		{"first step with incoming label", []ast.Step{{Title: "A", In: "x"}}},
		{"store link without arrows", []ast.Step{{Title: "A", Stores: []ast.StoreLink{{Name: "S"}}}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := ast.New(c.steps); err == nil {
				t.Fatal("want error, got nil")
			}
		})
	}
}

func TestNewStoreLink(t *testing.T) {
	if _, err := ast.NewStoreLink("S", nil, nil); err == nil {
		t.Fatal("want error for zero-arrow store link")
	}
	l, err := ast.NewStoreLink("S", &ast.Arrow{Label: "in"}, nil)
	if err != nil {
		t.Fatalf("NewStoreLink: %v", err)
	}
	if l.Put == nil || l.Put.Label != "in" || l.Get != nil {
		t.Fatalf("unexpected link: %+v", l)
	}
}
```

- [ ] **Step 2: Run to verify RED**

`go test ./ast` — expected: FAIL (package missing).

- [ ] **Step 3: Implement**

`ast/ast.go`:

```go
// Package ast defines the validated in-memory form of a dfd document:
// a strictly linear sequence of process steps with attached datastores.
package ast

import "fmt"

// Diagram is a linear flow. Construct with New so invariants hold.
type Diagram struct{ Steps []Step }

// Step is one process box. In labels the flow arrow from the previous
// step ("" = unlabeled); it is meaningless on the first step and must be
// empty there.
type Step struct {
	Title  string
	In     string
	Stores []StoreLink
}

// StoreLink attaches a datastore to a step. At least one of Put (step
// sends to store) and Get (store returns to step) is non-nil.
type StoreLink struct {
	Name     string
	Put, Get *Arrow
}

type Arrow struct{ Label string }

func New(steps []Step) (*Diagram, error) {
	if len(steps) == 0 {
		return nil, fmt.Errorf("no processes found")
	}
	if steps[0].In != "" {
		return nil, fmt.Errorf("arrow has no source process")
	}
	for _, s := range steps {
		for _, l := range s.Stores {
			if l.Put == nil && l.Get == nil {
				return nil, fmt.Errorf("datastore %q has no arrows", l.Name)
			}
		}
	}
	return &Diagram{Steps: steps}, nil
}

func NewStoreLink(name string, put, get *Arrow) (StoreLink, error) {
	if put == nil && get == nil {
		return StoreLink{}, fmt.Errorf("datastore %q has no arrows; add > and/or < lines before it", name)
	}
	return StoreLink{Name: name, Put: put, Get: get}, nil
}
```

- [ ] **Step 4: Verify green + hygiene**

`gofmt -l . && go vet ./... && go test ./...` — expected: all green.

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "Add ast package with validated diagram types"
```

---

### Task 3: parse — processes, flow arrows, comments

**Files:**
- Create: `parse/parse.go`, Test: `parse/parse_test.go`

**Interfaces:**
- Consumes: `ast.New`, `ast.Step`.
- Produces: `parse.Parse(r io.Reader, name string) (*ast.Diagram, error)`; `parse.Error{Name string; Line int; Msg string}` with `Error()` = `name:line: msg` (`name: msg` when Line 0).

- [ ] **Step 1: Write failing unit tests**

`parse/parse_test.go`:

```go
package parse_test

import (
	"strings"
	"testing"

	"github.com/bilus/dfd/ast"
	"github.com/bilus/dfd/parse"
)

func mustParse(t *testing.T, src string) *ast.Diagram {
	t.Helper()
	d, err := parse.Parse(strings.NewReader(src), "test.dfd")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return d
}

func TestParseLinearFlow(t *testing.T) {
	d := mustParse(t, "[Start]\n> go\n[Middle]\n[End]\n")
	want := []ast.Step{{Title: "Start"}, {Title: "Middle", In: "go"}, {Title: "End"}}
	if len(d.Steps) != len(want) {
		t.Fatalf("got %d steps, want %d", len(d.Steps), len(want))
	}
	for i := range want {
		if d.Steps[i].Title != want[i].Title || d.Steps[i].In != want[i].In {
			t.Errorf("step %d = %+v, want %+v", i, d.Steps[i], want[i])
		}
	}
}

func TestParseTrimsAndIgnores(t *testing.T) {
	src := "# comment\n\n   [ Padded ]\r\n  > label text \n\t[B]\n# tail\n"
	d := mustParse(t, src)
	if d.Steps[0].Title != "Padded" {
		t.Errorf("title = %q, want %q", d.Steps[0].Title, "Padded")
	}
	if d.Steps[1].In != "label text" {
		t.Errorf("label = %q, want %q", d.Steps[1].In, "label text")
	}
}

func TestParseUnlabeledArrowLine(t *testing.T) {
	d := mustParse(t, "[A]\n>\n[B]\n")
	if d.Steps[1].In != "" {
		t.Errorf("label = %q, want empty", d.Steps[1].In)
	}
}
```

- [ ] **Step 2: Run to verify RED**

`go test ./parse` — expected: FAIL (package missing).

- [ ] **Step 3: Implement with holes, fill most-constrained-first**

`parse/parse.go` (final state; grow it hole-by-hole — classification first, then binding):

```go
// Package parse turns dfd source text into an ast.Diagram.
package parse

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/bilus/dfd/ast"
)

// Error is a parse error tied to a line of the input. Line 0 means the
// error concerns the document as a whole.
type Error struct {
	Name string
	Line int
	Msg  string
}

func (e *Error) Error() string {
	if e.Line == 0 {
		return fmt.Sprintf("%s: %s", e.Name, e.Msg)
	}
	return fmt.Sprintf("%s:%d: %s", e.Name, e.Line, e.Msg)
}

type arrowLine struct {
	line  int
	back  bool // '<'
	label string
}

func Parse(r io.Reader, name string) (*ast.Diagram, error) {
	errf := func(line int, format string, args ...any) error {
		return &Error{Name: name, Line: line, Msg: fmt.Sprintf(format, args...)}
	}
	var steps []ast.Step
	var pending []arrowLine

	sc := bufio.NewScanner(r)
	n := 0
	for sc.Scan() {
		n++
		line := strings.TrimSpace(strings.TrimSuffix(sc.Text(), "\r"))
		switch {
		case line == "" || strings.HasPrefix(line, "#"):
			continue
		case strings.HasPrefix(line, "["):
			title, err := unbracket(line, ']', n, errf)
			if err != nil {
				return nil, err
			}
			if title == "" {
				return nil, errf(n, "empty process name")
			}
			in, err := bindFlow(steps, pending, errf)
			if err != nil {
				return nil, err
			}
			pending = nil
			steps = append(steps, ast.Step{Title: title, In: in})
		case strings.HasPrefix(line, "|"):
			nm, err := unbracket(line, '|', n, errf)
			if err != nil {
				return nil, err
			}
			if nm == "" {
				return nil, errf(n, "empty datastore name")
			}
			if len(steps) == 0 {
				return nil, errf(n, "datastore before any process")
			}
			link, err := bindStore(nm, pending, n, errf)
			if err != nil {
				return nil, err
			}
			pending = nil
			steps[len(steps)-1].Stores = append(steps[len(steps)-1].Stores, link)
		case strings.HasPrefix(line, ">") || strings.HasPrefix(line, "<"):
			pending = append(pending, arrowLine{
				line:  n,
				back:  line[0] == '<',
				label: strings.TrimSpace(line[1:]),
			})
		default:
			return nil, errf(n, "unrecognized line; expected [process], |store|, > or < arrow, or # comment")
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	if len(pending) > 0 {
		if len(steps) == 0 {
			return nil, errf(pending[0].line, "arrow has no source process")
		}
		return nil, errf(pending[0].line, "arrow has no target")
	}
	if len(steps) == 0 {
		return nil, errf(0, "no processes found")
	}
	d, err := ast.New(steps)
	if err != nil {
		return nil, errf(0, "%v", err)
	}
	return d, nil
}

// bindFlow resolves an arrow run that precedes a process line into the
// label of the flow arrow entering that process.
func bindFlow(steps []ast.Step, pending []arrowLine, errf func(int, string, ...any) error) (string, error) {
	if len(pending) == 0 {
		return "", nil
	}
	if len(steps) == 0 {
		return "", errf(pending[0].line, "arrow has no source process")
	}
	for _, a := range pending {
		if a.back {
			return "", errf(a.line, "'<' cannot point at a process; return arrows only precede a |store| line")
		}
	}
	if len(pending) > 1 {
		return "", errf(pending[1].line, "multiple flow arrows between processes")
	}
	return pending[0].label, nil
}

// bindStore resolves an arrow run that precedes a |store| line into a
// StoreLink for the nearest preceding process.
func bindStore(name string, pending []arrowLine, line int, errf func(int, string, ...any) error) (ast.StoreLink, error) {
	if len(pending) == 0 {
		return ast.StoreLink{}, errf(line, "datastore %q has no arrows; add > and/or < lines before it", name)
	}
	var put, get *ast.Arrow
	for _, a := range pending {
		arrow := &ast.Arrow{Label: a.label}
		if a.back {
			if get != nil {
				return ast.StoreLink{}, errf(a.line, "duplicate '<' arrow for datastore %q", name)
			}
			get = arrow
		} else {
			if put != nil {
				return ast.StoreLink{}, errf(a.line, "duplicate '>' arrow for datastore %q", name)
			}
			put = arrow
		}
	}
	link, err := ast.NewStoreLink(name, put, get)
	if err != nil {
		return ast.StoreLink{}, errf(line, "%v", err)
	}
	return link, nil
}

// unbracket parses a line of the form <open>text<close> where open is
// line[0] and close is the matching delimiter, honoring \<close> and \\
// escapes. The close delimiter must end the line.
func unbracket(line string, close byte, n int, errf func(int, string, ...any) error) (string, error) {
	var b strings.Builder
	i := 1
	for i < len(line) {
		c := line[i]
		switch {
		case c == '\\' && i+1 < len(line) && (line[i+1] == close || line[i+1] == '\\'):
			b.WriteByte(line[i+1])
			i += 2
		case c == close:
			if i != len(line)-1 {
				return "", errf(n, "unexpected text after %q", string(close))
			}
			return strings.TrimSpace(b.String()), nil
		default:
			b.WriteByte(c)
			i++
		}
	}
	return "", errf(n, "missing closing %q", string(close))
}
```

- [ ] **Step 4: Verify green + hygiene**

`gofmt -l . && go vet ./... && go test ./...` — expected: all green.

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "Add parser for processes, flow arrows, and comments"
```

---

### Task 4: parse — datastore binding

**Files:**
- Modify: `parse/parse.go` (already complete from Task 3 — this task exists to prove store binding with tests; if Task 3 was built hole-by-hole, `bindStore` may still be a hole and gets filled here)
- Test: `parse/parse_test.go`

**Interfaces:**
- Consumes: `ast.StoreLink`, `ast.Arrow`.
- Produces: store binding behavior relied on by layout tasks.

- [ ] **Step 1: Write failing/locking unit tests**

Append to `parse/parse_test.go`:

```go
func TestParseStoreBinding(t *testing.T) {
	src := `[Initial step]
> input
[Store in database]
    > input
    < record id
    |Somethings|
> record id
[Return to user]
`
	d := mustParse(t, src)
	if len(d.Steps) != 3 {
		t.Fatalf("got %d steps, want 3", len(d.Steps))
	}
	s := d.Steps[1]
	if len(s.Stores) != 1 {
		t.Fatalf("got %d stores, want 1", len(s.Stores))
	}
	l := s.Stores[0]
	if l.Name != "Somethings" {
		t.Errorf("name = %q", l.Name)
	}
	if l.Put == nil || l.Put.Label != "input" {
		t.Errorf("put = %+v, want label %q", l.Put, "input")
	}
	if l.Get == nil || l.Get.Label != "record id" {
		t.Errorf("get = %+v, want label %q", l.Get, "record id")
	}
	if d.Steps[2].In != "record id" {
		t.Errorf("flow into last = %q", d.Steps[2].In)
	}
}

func TestParseStoreVariants(t *testing.T) {
	cases := []struct {
		name           string
		src            string
		wantPut, wantGet bool
	}{
		{"write only", "[A]\n> save\n|S|\n", true, false},
		{"read only", "[A]\n< load\n|S|\n", false, true},
		{"reversed order", "[A]\n< load\n> save\n|S|\n", true, true},
		{"unlabeled both", "[A]\n>\n<\n|S|\n", true, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := mustParse(t, c.src)
			l := d.Steps[0].Stores[0]
			if (l.Put != nil) != c.wantPut || (l.Get != nil) != c.wantGet {
				t.Fatalf("put=%v get=%v, want %v/%v", l.Put != nil, l.Get != nil, c.wantPut, c.wantGet)
			}
		})
	}
}

func TestParseMultipleStoresPerStep(t *testing.T) {
	d := mustParse(t, "[A]\n> x\n|S1|\n< y\n|S2|\n[B]\n")
	if n := len(d.Steps[0].Stores); n != 2 {
		t.Fatalf("got %d stores, want 2", n)
	}
	if d.Steps[0].Stores[1].Name != "S2" {
		t.Errorf("second store = %q", d.Steps[0].Stores[1].Name)
	}
}
```

- [ ] **Step 2: Run tests**

`go test ./parse` — expected: PASS if Task 3 landed the full file, otherwise fill the `bindStore` hole until green.

- [ ] **Step 3: Hygiene + commit**

```bash
gofmt -l . && go vet ./... && go test ./...
git add -A && git commit -m "Cover datastore binding in parser tests"
```

---

### Task 5: parse — error cases and escapes

**Files:**
- Test: `parse/parse_test.go`
- Modify: `docs/superpowers/specs/2026-07-23-text-to-dfd-design.md` (add the "unrecognized line" row to the error table — it was missing from the spec)

**Interfaces:**
- Produces: exact error messages relied on by Task 11's acceptance scripts.

- [ ] **Step 1: Write failing/locking unit tests**

Append to `parse/parse_test.go`:

```go
func TestParseErrors(t *testing.T) {
	cases := []struct {
		name, src, want string
	}{
		{"empty document", "", "test.dfd: no processes found"},
		{"comments only", "# hi\n\n", "test.dfd: no processes found"},
		{"arrow before first process", "> x\n[A]\n", "test.dfd:1: arrow has no source process"},
		{"arrow at eof", "[A]\n> x\n", "test.dfd:2: arrow has no target"},
		{"back arrow at process", "[A]\n< x\n[B]\n", "test.dfd:2: '<' cannot point at a process; return arrows only precede a |store| line"},
		{"double flow arrow", "[A]\n> x\n> y\n[B]\n", "test.dfd:3: multiple flow arrows between processes"},
		{"duplicate put", "[A]\n> x\n> y\n|S|\n", "test.dfd:3: duplicate '>' arrow for datastore \"S\""},
		{"duplicate get", "[A]\n< x\n< y\n|S|\n", "test.dfd:3: duplicate '<' arrow for datastore \"S\""},
		{"store without arrows", "[A]\n|S|\n", "test.dfd:2: datastore \"S\" has no arrows; add > and/or < lines before it"},
		{"store before process", "> x\n|S|\n", "test.dfd:2: datastore before any process"},
		{"missing close bracket", "[A\n", "test.dfd:1: missing closing \"]\""},
		{"missing close pipe", "[A]\n> x\n|S\n", "test.dfd:3: missing closing \"|\""},
		{"trailing text", "[A] tail\n", "test.dfd:1: unexpected text after \"]\""},
		{"empty process name", "[]\n", "test.dfd:1: empty process name"},
		{"empty store name", "[A]\n> x\n||\n", "test.dfd:3: empty datastore name"},
		{"unrecognized line", "[A]\nwat\n", "test.dfd:2: unrecognized line; expected [process], |store|, > or < arrow, or # comment"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := parse.Parse(strings.NewReader(c.src), "test.dfd")
			if err == nil {
				t.Fatal("want error, got nil")
			}
			if err.Error() != c.want {
				t.Fatalf("error = %q\nwant    %q", err.Error(), c.want)
			}
		})
	}
}

func TestParseEscapes(t *testing.T) {
	d := mustParse(t, `[Array a\]b]`+"\n> l\n"+`|Pipe c\|d|`+"\n")
	if got := d.Steps[0].Title; got != "Array a]b" {
		t.Errorf("title = %q", got)
	}
	if got := d.Steps[0].Stores[0].Name; got != "Pipe c|d" {
		t.Errorf("store = %q", got)
	}
}
```

Note the store-before-process case: the arrow run is consumed at the store line, whose "before any process" check fires first — line 2, matching the implementation order in Task 3.

- [ ] **Step 2: Run, fix any message drift, verify green**

`go test ./parse` — expected: PASS (messages already match Task 3's implementation; fix implementation if any case disagrees — the table above is normative).

- [ ] **Step 3: Add the missing spec row**

In the spec's parse-error table, append:

```markdown
| Unrecognizable line                                   | unrecognized line; expected [process], \|store\|, > or < arrow, or # comment |
```

- [ ] **Step 4: Hygiene + commit**

```bash
gofmt -l . && go vet ./... && go test ./...
git add -A && git commit -m "Lock parser error messages and escapes; add unrecognized-line error to spec"
```

---

### Task 6: typeface + layout — single row

**Files:**
- Create: `internal/typeface/typeface.go`, `layout/layout.go`, `layout/wrap.go`
- Test: `layout/layout_test.go`, `layout/wrap_test.go`

**Interfaces:**
- Consumes: `ast.Diagram`.
- Produces:
  - `typeface.New(size float64) (font.Face, error)` — embedded Go Regular at given px (DPI 72, no hinting).
  - `layout.Config{BoxW, BoxH, MaxWidth, PerRow, FontSize int; Face font.Face}`
  - `layout.Arrange(d *ast.Diagram, c Config) (*Scene, error)`
  - `layout.Scene{W, H, FontSize int; Items []Item}`; items `layout.Rect{X, Y, W, H int}`, `layout.Line{X1, Y1, X2, Y2 int; Head, Thick bool}` (Thick = 2px structure stroke, else 1.5px arrow; Head = arrowhead at X2,Y2), `layout.Text{X, Y int; S string; Anchor Anchor}` with `layout.Start/Middle/End`.
  - Exported constants: `HGap=90, VGap=90, Margin=40, StoreW=150, StoreH=36, StoreArrow=64, StoreGap=20, BoxPad=12, LabelGap=8, Inset=3, LineH=17`.

- [ ] **Step 1: Write failing unit tests**

`layout/wrap_test.go`:

```go
package layout_test

import (
	"testing"

	"github.com/bilus/dfd/internal/typeface"
	"github.com/bilus/dfd/layout"
)

func TestWrapTitle(t *testing.T) {
	face, err := typeface.New(13)
	if err != nil {
		t.Fatalf("typeface: %v", err)
	}
	cases := []struct {
		s    string
		maxW int
		want []string
	}{
		{"Start process", 136, []string{"Start process"}},
		{"Change something that doesn't work", 136, []string{"Change something", "that doesn't work"}},
		{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 136, nil}, // just: every line must fit
	}
	for _, c := range cases {
		got := layout.WrapText(c.s, c.maxW, face)
		if c.want != nil {
			if len(got) != len(c.want) {
				t.Fatalf("WrapText(%q) = %q, want %q", c.s, got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Fatalf("WrapText(%q) = %q, want %q", c.s, got, c.want)
				}
			}
		}
		for _, line := range got {
			if w := layout.TextWidth(line, face); w > c.maxW {
				t.Errorf("line %q is %dpx, exceeds %d", line, w, c.maxW)
			}
		}
	}
}
```

`layout/layout_test.go`:

```go
package layout_test

import (
	"strings"
	"testing"

	"github.com/bilus/dfd/ast"
	"github.com/bilus/dfd/layout"
	"github.com/bilus/dfd/parse"
	"github.com/bilus/dfd/internal/typeface"
)

func arrange(t *testing.T, src string, c layout.Config) *layout.Scene {
	t.Helper()
	d, err := parse.Parse(strings.NewReader(src), "test.dfd")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if c.Face == nil {
		face, err := typeface.New(13)
		if err != nil {
			t.Fatalf("typeface: %v", err)
		}
		c.Face = face
	}
	if c.BoxW == 0 {
		c.BoxW, c.BoxH, c.MaxWidth, c.FontSize = 160, 60, 1000, 13
	}
	s, err := layout.Arrange(d, c)
	if err != nil {
		t.Fatalf("Arrange: %v", err)
	}
	return s
}

func rects(s *layout.Scene) []layout.Rect {
	var out []layout.Rect
	for _, it := range s.Items {
		if r, ok := it.(layout.Rect); ok {
			out = append(out, r)
		}
	}
	return out
}

func TestSingleRowGeometry(t *testing.T) {
	s := arrange(t, "[A]\n> go\n[B]\n", layout.Config{})
	rs := rects(s)
	if len(rs) != 2 {
		t.Fatalf("got %d rects, want 2", len(rs))
	}
	if rs[0].X != layout.Margin || rs[0].Y != layout.Margin {
		t.Errorf("first box at (%d,%d), want (%d,%d)", rs[0].X, rs[0].Y, layout.Margin, layout.Margin)
	}
	if want := layout.Margin + 160 + layout.HGap; rs[1].X != want {
		t.Errorf("second box x = %d, want %d", rs[1].X, want)
	}
	if want := 2*layout.Margin + 2*160 + layout.HGap; s.W != want {
		t.Errorf("scene w = %d, want %d", s.W, want)
	}
	if want := 2*layout.Margin + 60; s.H != want {
		t.Errorf("scene h = %d, want %d", s.H, want)
	}
	var arrow *layout.Line
	for _, it := range s.Items {
		if l, ok := it.(layout.Line); ok && l.Head {
			arrow = &l
			break
		}
	}
	if arrow == nil {
		t.Fatal("no arrowed line in scene")
	}
	cy := layout.Margin + 30
	if arrow.Y1 != cy || arrow.Y2 != cy {
		t.Errorf("arrow y = %d/%d, want %d", arrow.Y1, arrow.Y2, cy)
	}
	if arrow.X1 != layout.Margin+160 || arrow.X2 != layout.Margin+160+layout.HGap-layout.Inset {
		t.Errorf("arrow x = %d..%d", arrow.X1, arrow.X2)
	}
	foundLabel := false
	for _, it := range s.Items {
		if tx, ok := it.(layout.Text); ok && tx.S == "go" {
			foundLabel = true
			if tx.Anchor != layout.Middle {
				t.Errorf("label anchor = %v, want Middle", tx.Anchor)
			}
			if wantX := layout.Margin + 160 + layout.HGap/2; tx.X != wantX {
				t.Errorf("label x = %d, want %d", tx.X, wantX)
			}
			if wantY := cy - layout.LabelGap; tx.Y != wantY {
				t.Errorf("label y = %d, want %d", tx.Y, wantY)
			}
		}
	}
	if !foundLabel {
		t.Error("flow label not in scene")
	}
}

func TestNoOverlappingRects(t *testing.T) {
	s := arrange(t, "[One]\n[Two]\n[Three]\n[Four]\n", layout.Config{})
	rs := rects(s)
	for i := 0; i < len(rs); i++ {
		for j := i + 1; j < len(rs); j++ {
			a, b := rs[i], rs[j]
			if a.X < b.X+b.W && b.X < a.X+a.W && a.Y < b.Y+b.H && b.Y < a.Y+a.H {
				t.Errorf("rects %d and %d overlap: %+v %+v", i, j, a, b)
			}
		}
	}
}
```

- [ ] **Step 2: Run to verify RED**

`go test ./layout` — expected: FAIL (packages missing).

- [ ] **Step 3: Implement typeface, WrapText/TextWidth, then Arrange (holes first)**

`internal/typeface/typeface.go`:

```go
// Package typeface provides the embedded font face used for all text
// measurement and PNG rendering.
package typeface

import (
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
)

func New(size float64) (font.Face, error) {
	f, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(f, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingNone,
	})
}
```

Run `go get github.com/fogleman/gg golang.org/x/image` (gg is used in Task 12; fetching x/image now).

`layout/wrap.go`:

```go
package layout

import (
	"strings"

	"golang.org/x/image/font"
)

// TextWidth is the advance width of s in whole pixels.
func TextWidth(s string, face font.Face) int {
	return font.MeasureString(face, s).Ceil()
}

// WrapText greedily wraps s at word boundaries so every line fits maxW.
// A single word wider than maxW is broken mid-word.
func WrapText(s string, maxW int, face font.Face) []string {
	var lines []string
	var cur string
	flush := func() {
		if cur != "" {
			lines = append(lines, cur)
			cur = ""
		}
	}
	for _, w := range strings.Fields(s) {
		for TextWidth(w, face) > maxW {
			flush()
			i := len(w)
			for i > 1 && TextWidth(w[:i], face) > maxW {
				i--
			}
			lines = append(lines, w[:i])
			w = w[i:]
		}
		switch {
		case cur == "":
			cur = w
		case TextWidth(cur+" "+w, face) <= maxW:
			cur += " " + w
		default:
			flush()
			cur = w
		}
	}
	flush()
	if lines == nil {
		lines = []string{""}
	}
	return lines
}
```

`layout/layout.go` — single-row subset (rows/stores are Task 8/9 holes; `Arrange` structure below is final, with row assignment trivially one row for now):

```go
// Package layout turns an ast.Diagram into a Scene: a flat display list
// of rectangles, lines, and anchored text with final pixel coordinates.
package layout

import (
	"fmt"

	"golang.org/x/image/font"

	"github.com/bilus/dfd/ast"
)

const (
	HGap       = 90
	VGap       = 90
	Margin     = 40
	StoreW     = 150
	StoreH     = 36
	StoreArrow = 64
	StoreGap   = 20
	BoxPad     = 12
	LabelGap   = 8
	Inset      = 3
	LineH      = 17
)

type Config struct {
	BoxW, BoxH int // BoxH is a minimum; grows for tall wrapped titles
	MaxWidth   int
	PerRow     int // 0 = derive from MaxWidth
	FontSize   int
	Face       font.Face
}

type Scene struct {
	W, H, FontSize int
	Items          []Item
}

type Item interface{ item() }

type Rect struct{ X, Y, W, H int }

// Line is a straight segment. Head draws an arrowhead at (X2,Y2).
// Thick marks 2px structural strokes (boxes use Rect; store glyph lines
// use Thick) versus 1.5px arrow strokes.
type Line struct {
	X1, Y1, X2, Y2 int
	Head, Thick    bool
}

type Anchor int

const (
	Start Anchor = iota
	Middle
	End
)

// Text is a single line of text; Y is the baseline.
type Text struct {
	X, Y   int
	S      string
	Anchor Anchor
}

func (Rect) item() {}
func (Line) item() {}
func (Text) item() {}

func Arrange(d *ast.Diagram, c Config) (*Scene, error) {
	if c.Face == nil {
		return nil, fmt.Errorf("layout: Config.Face is required")
	}
	titles := make([][]string, len(d.Steps))
	boxH := c.BoxH
	for i, st := range d.Steps {
		titles[i] = WrapText(st.Title, c.BoxW-2*BoxPad, c.Face)
		if h := len(titles[i])*LineH + 2*BoxPad; h > boxH {
			boxH = h
		}
	}
	colW := c.BoxW // widened by store groups in Task 9
	perRow := c.PerRow
	if perRow <= 0 {
		perRow = 1
		for 2*Margin+(perRow+1)*colW+perRow*HGap <= c.MaxWidth {
			perRow++
		}
	}
	g := grid{d: d, c: c, titles: titles, boxH: boxH, colW: colW, perRow: perRow}
	return g.arrange()
}

// grid carries the shared geometry while emitting items.
type grid struct {
	d      *ast.Diagram
	c      Config
	titles [][]string
	boxH   int
	colW   int
	perRow int
}

func (g *grid) arrange() (*Scene, error) {
	n := len(g.d.Steps)
	if g.perRow < len(g.d.Steps) {
		return nil, fmt.Errorf("layout: multiple rows not implemented yet")
	}
	s := &Scene{FontSize: g.c.FontSize}
	rowY := Margin
	cy := rowY + g.boxH/2
	for i := range g.d.Steps {
		x := Margin + i*(g.colW+HGap)
		g.emitBox(s, i, x, rowY)
		if i > 0 {
			from := Margin + (i-1)*(g.colW+HGap) + g.boxW()
			g.emitFlowArrow(s, from, x, cy, g.d.Steps[i].In)
		}
	}
	s.W = 2*Margin + n*g.colW + (n-1)*HGap
	s.H = rowY + g.boxH + Margin
	return s, nil
}

func (g *grid) boxW() int { return g.c.BoxW }

// emitBox draws step i's rectangle at (x, y) plus its wrapped title.
func (g *grid) emitBox(s *Scene, i, x, y int) {
	s.Items = append(s.Items, Rect{X: x, Y: y, W: g.boxW(), H: g.boxH})
	lines := g.titles[i]
	cx := x + g.boxW()/2
	first := y + g.boxH/2 + 5 - (len(lines)-1)*LineH/2
	for j, line := range lines {
		s.Items = append(s.Items, Text{X: cx, Y: first + j*LineH, S: line, Anchor: Middle})
	}
}

// emitFlowArrow draws a horizontal flow arrow from x-edge from to the box
// starting at to, with the optional label centered above the midpoint.
func (g *grid) emitFlowArrow(s *Scene, from, to, cy int, label string) {
	x2 := to - Inset
	if to < from { // right-to-left rows (Task 8)
		x2 = to + g.boxW() + Inset
		from = to + g.boxW() + HGap // unreachable until Task 8; kept simple here
	}
	s.Items = append(s.Items, Line{X1: from, Y1: cy, X2: x2, Y2: cy, Head: true})
	if label != "" {
		s.Items = append(s.Items, Text{X: (from + to) / 2, Y: cy - LabelGap, S: label, Anchor: Middle})
	}
}
```

Note: `emitFlowArrow`'s right-to-left branch is a visible hole — it is exercised and rewritten in Task 8; single-row tests only hit the left-to-right path. `TestSingleRowGeometry` pins midpoint math: for adjacent boxes `(from+to)/2` = gap center.

- [ ] **Step 4: Verify green + hygiene**

`gofmt -l . && go vet ./... && go test ./...` — expected: all green.

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "Add typeface and single-row layout with wrapped titles"
```

---

### Task 7: SVG renderer + CLI wiring — first acceptance greens

**Files:**
- Create: `render/svg.go`, Test: `render/svg_test.go`
- Modify: `internal/cli/cli.go` (fill the `run` hole for file→file SVG)
- Create: `cmd/dfd/testdata/script/two_steps.txtar`, `cmd/dfd/testdata/script/labeled_arrows.txtar`

**Interfaces:**
- Consumes: `layout.Scene` items, `parse.Parse`, `layout.Arrange`, `typeface.New`.
- Produces: `render.SVG(s *layout.Scene, w io.Writer) error`; working `dfd in.dfd -o out.svg`.

- [ ] **Step 1: Write the failing acceptance tests (RED first)**

`cmd/dfd/testdata/script/two_steps.txtar`:

```
dfd flow.dfd -o flow.svg
cmp flow.svg want.svg

-- flow.dfd --
[First]
[Second]
-- want.svg --
```

`cmd/dfd/testdata/script/labeled_arrows.txtar`:

```
dfd flow.dfd -o flow.svg
cmp flow.svg want.svg

-- flow.dfd --
[Start process]
> foo
[Continue running]
>
[Finish]
-- want.svg --
```

Run: `go test ./cmd/dfd` — expected: FAIL with `dfd: not implemented` (the `want.svg` sections are intentionally empty; `-update` seeds them in Step 4).

- [ ] **Step 2: Unit-test and implement render.SVG**

`render/svg_test.go`:

```go
package render_test

import (
	"strings"
	"testing"

	"github.com/bilus/dfd/layout"
	"github.com/bilus/dfd/render"
)

func TestSVGEmitsItems(t *testing.T) {
	s := &layout.Scene{W: 300, H: 100, FontSize: 13, Items: []layout.Item{
		layout.Rect{X: 10, Y: 20, W: 160, H: 60},
		layout.Line{X1: 170, Y1: 50, X2: 200, Y2: 50, Head: true},
		layout.Line{X1: 10, Y1: 90, X2: 170, Y2: 90, Thick: true},
		layout.Text{X: 90, Y: 55, S: "A & B", Anchor: layout.Middle},
	}}
	var b strings.Builder
	if err := render.SVG(s, &b); err != nil {
		t.Fatalf("SVG: %v", err)
	}
	out := b.String()
	for _, want := range []string{
		`viewBox="0 0 300 100"`,
		`<rect x="10" y="20" width="160" height="60"/>`,
		`marker-end="url(#ah)"`,
		`stroke-width="2"`,
		`text-anchor="middle"`,
		`A &amp; B`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n%s", want, out)
		}
	}
	if strings.Count(out, "\n") < 8 {
		t.Error("expected line-structured output")
	}
}
```

`render/svg.go`:

```go
// Package render draws a layout.Scene as SVG or PNG.
package render

import (
	"encoding/xml"
	"fmt"
	"io"

	"github.com/bilus/dfd/layout"
)

// SVG writes the scene as a standalone SVG document. Output is
// deterministic: items are emitted in scene order with fixed formatting.
func SVG(s *layout.Scene, w io.Writer) error {
	p := &printer{w: w}
	p.f(`<svg viewBox="0 0 %d %d" width="%d" height="%d" xmlns="http://www.w3.org/2000/svg" font-family="Helvetica, Arial, sans-serif" font-size="%d">`+"\n",
		s.W, s.H, s.W, s.H, s.FontSize)
	p.f("  <defs>\n    <marker id=\"ah\" viewBox=\"0 0 10 10\" refX=\"9\" refY=\"5\" markerWidth=\"8\" markerHeight=\"8\" orient=\"auto\">\n      <path d=\"M0,0L10,5L0,10z\" fill=\"#000\"/>\n    </marker>\n  </defs>\n")
	p.f(`  <rect x="0" y="0" width="%d" height="%d" fill="#fff"/>`+"\n", s.W, s.H)
	for _, it := range s.Items {
		switch v := it.(type) {
		case layout.Rect:
			p.f(`  <rect x="%d" y="%d" width="%d" height="%d" fill="#fff" stroke="#000" stroke-width="2"/>`+"\n", v.X, v.Y, v.W, v.H)
		case layout.Line:
			width, marker := "1.5", ""
			if v.Thick {
				width = "2"
			}
			if v.Head {
				marker = ` marker-end="url(#ah)"`
			}
			p.f(`  <line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#000" stroke-width="%s"%s/>`+"\n", v.X1, v.Y1, v.X2, v.Y2, width, marker)
		case layout.Text:
			anchor := ""
			switch v.Anchor {
			case layout.Middle:
				anchor = ` text-anchor="middle"`
			case layout.End:
				anchor = ` text-anchor="end"`
			}
			p.f(`  <text x="%d" y="%d" fill="#000"%s>%s</text>`+"\n", v.X, v.Y, anchor, escape(v.S))
		}
	}
	p.f("</svg>\n")
	return p.err
}

func escape(s string) string {
	var b []byte
	if err := xml.EscapeText(discard{&b}, []byte(s)); err != nil {
		return s // cannot fail for valid UTF-8; fall back to raw
	}
	return string(b)
}

type discard struct{ b *[]byte }

func (d discard) Write(p []byte) (int, error) {
	*d.b = append(*d.b, p...)
	return len(p), nil
}

// printer accumulates the first write error so call sites stay linear.
type printer struct {
	w   io.Writer
	err error
}

func (p *printer) f(format string, args ...any) {
	if p.err != nil {
		return
	}
	_, p.err = fmt.Fprintf(p.w, format, args...)
}
```

Run: `go test ./render` — RED then green.

- [ ] **Step 3: Fill the cli `run` hole (file input → SVG output)**

Replace `run` in `internal/cli/cli.go` and add imports (`os`, `path/filepath`, `strings`, plus the project packages):

```go
func run(o options, stdin io.Reader, stdout io.Writer) error {
	var src io.Reader
	name := o.input
	if name == "" {
		name = "<stdin>"
		src = stdin
	} else {
		f, err := os.Open(o.input)
		if err != nil {
			return err
		}
		defer func() {
			if cerr := f.Close(); cerr != nil && err == nil {
				err = cerr
			}
		}()
		src = f
	}
	_ = stdout // stdout output arrives in Task 10
	d, err := parse.Parse(src, name)
	if err != nil {
		return err
	}
	var boxW, boxH int
	if _, err := fmt.Sscanf(o.box, "%dx%d", &boxW, &boxH); err != nil {
		return fmt.Errorf("dfd: invalid --box %q (want WxH)", o.box)
	}
	face, err := typeface.New(float64(o.fontSize))
	if err != nil {
		return err
	}
	scene, err := layout.Arrange(d, layout.Config{
		BoxW: boxW, BoxH: boxH,
		MaxWidth: o.maxWidth, PerRow: o.perRow,
		FontSize: o.fontSize, Face: face,
	})
	if err != nil {
		return err
	}
	out := o.out
	if out == "" {
		if o.input == "" {
			return fmt.Errorf("dfd: stdout output not implemented yet")
		}
		out = strings.TrimSuffix(o.input, filepath.Ext(o.input)) + ".svg"
	}
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	if err := render.SVG(scene, f); err != nil {
		if cerr := f.Close(); cerr != nil {
			return fmt.Errorf("%v; close: %v", err, cerr)
		}
		return err
	}
	return f.Close()
}
```

Note the deferred-close pattern requires naming the return (`func run(...) (err error)`) — do that, or hoist the parse before defer as shown; when filling this hole choose the named-return form so no error is discarded.

- [ ] **Step 4: Turn acceptance green via -update, review, verify**

```bash
go test ./cmd/dfd -run TestScript -update
go test ./...
```

Expected: update run rewrites `want.svg` in both txtars; full suite PASS. Open both txtars, sanity-check the seeded SVG (two 160×60 rects at y=40, arrow with/without `foo` label, unlabeled second arrow in `labeled_arrows`). Render one to PNG for eyeball if desired:

```bash
go run ./cmd/dfd cmd/dfd/testdata/... # not needed; txtar diff review suffices
```

- [ ] **Step 5: Hygiene + commit**

```bash
gofmt -l . && go vet ./... && go test ./...
git add -A && git commit -m "Render single-row diagrams to SVG via CLI"
```

---

### Task 8: Snake layout — rows, turn arrows, flags

**Files:**
- Modify: `layout/layout.go` (replace `grid.arrange` single-row limitation; real multi-row)
- Test: `layout/layout_test.go`
- Create: `cmd/dfd/testdata/script/snake.txtar`, `cmd/dfd/testdata/script/per_row.txtar`

**Interfaces:**
- Consumes/Produces: unchanged signatures; `Arrange` now handles any step count; `--per-row`/`--max-width` become meaningful end-to-end.

- [ ] **Step 1: Write the failing acceptance tests (RED first)**

`cmd/dfd/testdata/script/snake.txtar` (5 boxes, `--max-width` forces 2 rows; covers turn arrow + reversed row):

```
dfd --max-width 700 flow.dfd -o flow.svg
cmp flow.svg want.svg

-- flow.dfd --
[One]
> a
[Two]
> b
[Three]
> down
[Four]
> c
[Five]
-- want.svg --
```

`cmd/dfd/testdata/script/per_row.txtar`:

```
dfd --per-row 2 flow.dfd -o flow.svg
cmp flow.svg want.svg

-- flow.dfd --
[One]
[Two]
[Three]
-- want.svg --
```

Run: `go test ./cmd/dfd` — expected: FAIL `layout: multiple rows not implemented yet`.

- [ ] **Step 2: Unit tests for snake geometry**

Append to `layout/layout_test.go`:

```go
func TestSnakeRows(t *testing.T) {
	s := arrange(t, "[One]\n[Two]\n[Three]\n[Four]\n[Five]\n", layout.Config{
		BoxW: 160, BoxH: 60, MaxWidth: 700, FontSize: 13,
	})
	rs := rects(s)
	if len(rs) != 5 {
		t.Fatalf("got %d rects, want 5", len(rs))
	}
	if rs[0].Y != rs[1].Y || rs[1].Y != rs[2].Y {
		t.Errorf("first row not aligned: %+v %+v %+v", rs[0], rs[1], rs[2])
	}
	if rs[3].Y == rs[0].Y || rs[3].Y != rs[4].Y {
		t.Errorf("second row wrong: %+v %+v", rs[3], rs[4])
	}
	if rs[3].X != rs[2].X {
		t.Errorf("row 2 must start under row 1's last box: %d vs %d", rs[3].X, rs[2].X)
	}
	if rs[4].X >= rs[3].X {
		t.Errorf("row 2 must run right-to-left: %+v then %+v", rs[3], rs[4])
	}
	if want := rs[0].Y + 60 + layout.VGap; rs[3].Y != want {
		t.Errorf("row 2 y = %d, want %d", rs[3].Y, want)
	}
	var vertical *layout.Line
	for _, it := range s.Items {
		if l, ok := it.(layout.Line); ok && l.Head && l.X1 == l.X2 {
			vertical = &l
			break
		}
	}
	if vertical == nil {
		t.Fatal("no vertical turn arrow")
	}
	if cx := rs[2].X + 80; vertical.X1 != cx {
		t.Errorf("turn arrow x = %d, want %d", vertical.X1, cx)
	}
	if vertical.Y1 != rs[2].Y+60 || vertical.Y2 != rs[3].Y-layout.Inset {
		t.Errorf("turn arrow y = %d..%d", vertical.Y1, vertical.Y2)
	}
}

func TestTurnArrowLabelPlacement(t *testing.T) {
	s := arrange(t, "[One]\n[Two]\n> down\n[Three]\n", layout.Config{
		BoxW: 160, BoxH: 60, MaxWidth: 100, PerRow: 2, FontSize: 13,
	})
	for _, it := range s.Items {
		if tx, ok := it.(layout.Text); ok && tx.S == "down" {
			if tx.Anchor != layout.Start {
				t.Errorf("turn label anchor = %v, want Start", tx.Anchor)
			}
			return
		}
	}
	t.Fatal("turn label not found")
}
```

Run: `go test ./layout` — expected: FAIL.

- [ ] **Step 3: Implement multi-row arrange**

Replace `grid.arrange` and `emitFlowArrow` in `layout/layout.go`:

```go
func (g *grid) arrange() (*Scene, error) {
	n := len(g.d.Steps)
	s := &Scene{FontSize: g.c.FontSize}
	nRows := (n + g.perRow - 1) / g.perRow

	// Column index for each step; odd rows mirror so each row starts
	// under the previous row's last box.
	col := make([]int, n)
	rowLen := make([]int, nRows)
	maxCols := 0
	for i := range g.d.Steps {
		r := i / g.perRow
		k := i % g.perRow
		rowLen[r]++
		if r%2 == 0 {
			col[i] = k
		} else {
			col[i] = g.perRow - 1 - k
		}
		if col[i]+1 > maxCols {
			maxCols = col[i] + 1
		}
	}

	rowY := make([]int, nRows)
	y := Margin
	for r := 0; r < nRows; r++ {
		rowY[r] = y
		y += g.boxH + VGap
	}

	xOf := func(c int) int { return Margin + c*(g.colW+HGap) }
	boxX := func(i int) int { return xOf(col[i]) + (g.colW-g.boxW())/2 }

	for i := range g.d.Steps {
		r := i / g.perRow
		g.emitBox(s, i, boxX(i), rowY[r])
	}
	for i := 1; i < n; i++ {
		r, pr := i/g.perRow, (i-1)/g.perRow
		label := g.d.Steps[i].In
		if r == pr {
			g.emitFlowArrow(s, i-1, i, boxX(i-1), boxX(i), rowY[r]+g.boxH/2, label)
		} else {
			g.emitTurnArrow(s, boxX(i)+g.boxW()/2, rowY[pr]+g.boxH, rowY[r], label)
		}
	}

	s.W = 2*Margin + maxCols*g.colW + (maxCols-1)*HGap
	s.H = rowY[nRows-1] + g.boxH + Margin
	return s, nil
}

// emitFlowArrow draws the horizontal arrow between two same-row boxes at
// center height cy; direction follows the row's flow.
func (g *grid) emitFlowArrow(s *Scene, from, to int, fromX, toX, cy int, label string) {
	var x1, x2 int
	if toX > fromX { // left-to-right row
		x1, x2 = fromX+g.boxW(), toX-Inset
	} else { // right-to-left row
		x1, x2 = fromX, toX+g.boxW()+Inset
	}
	s.Items = append(s.Items, Line{X1: x1, Y1: cy, X2: x2, Y2: cy, Head: true})
	if label != "" {
		mid := (fromX + toX + g.boxW()) / 2
		s.Items = append(s.Items, Text{X: mid, Y: cy - LabelGap, S: label, Anchor: Middle})
	}
}

// emitTurnArrow draws the vertical snake-turn arrow at x from the bottom
// of the source row to the top of the target row.
func (g *grid) emitTurnArrow(s *Scene, x, fromY, toY int, label string) {
	s.Items = append(s.Items, Line{X1: x, Y1: fromY, X2: x, Y2: toY - Inset, Head: true})
	if label != "" {
		s.Items = append(s.Items, Text{X: x + LabelGap, Y: (fromY+toY)/2 + 5, S: label, Anchor: Start})
	}
}
```

Also delete the now-dead single-row guard (`multiple rows not implemented yet`) and the old `emitFlowArrow`. The label midpoint `(fromX+toX+boxW)/2` equals the gap center in both directions — verify against `TestSingleRowGeometry` (unchanged, must stay green).

- [ ] **Step 4: Turn everything green + seed fixtures**

```bash
go test ./layout ./parse ./ast ./render
go test ./cmd/dfd -run TestScript -update
go test ./...
```

Expected: all PASS; `snake.txtar`/`per_row.txtar` seeded. Review seeded SVGs: 2 rows, second row right-to-left, one vertical arrow with `down` label to its right.

- [ ] **Step 5: Hygiene + commit**

```bash
gofmt -l . && go vet ./... && go test ./...
git add -A && git commit -m "Add auto-snake multi-row layout with turn arrows"
```

---

### Task 9: Datastores end-to-end

**Files:**
- Modify: `layout/layout.go` (store sides, lanes, glyph emission, column widening)
- Test: `layout/layout_test.go`
- Create: `cmd/dfd/testdata/script/store_write.txtar`, `store_read.txtar`, `store_read_write.txtar`, `multi_store.txtar`

**Interfaces:**
- Consumes/Produces: unchanged signatures. Store glyph = 2 `Line{Thick:true}` + name `Text` + per-arrow `Line{Head:true}` + label `Text`s.

- [ ] **Step 1: Write the failing acceptance tests (RED first)**

`store_write.txtar`:

```
dfd flow.dfd -o flow.svg
cmp flow.svg want.svg

-- flow.dfd --
[Save it]
> something
|Database|
-- want.svg --
```

`store_read.txtar`:

```
dfd flow.dfd -o flow.svg
cmp flow.svg want.svg

-- flow.dfd --
[Load it]
< rows
|Database|
-- want.svg --
```

`store_read_write.txtar` (your original mini-example):

```
dfd flow.dfd -o flow.svg
cmp flow.svg want.svg

-- flow.dfd --
[Initial step]
> input
[Store in database]
    > input
    < record id
    |Somethings|
> record id
[Return to user]
-- want.svg --
```

`multi_store.txtar`:

```
dfd flow.dfd -o flow.svg
cmp flow.svg want.svg

-- flow.dfd --
[Fan out]
> a
|Cache|
> b
|Queue with long name|
-- want.svg --
```

Run: `go test ./cmd/dfd` — expected: FAIL (stores parsed but not drawn → cmp against empty want fails... they SEED empty and diagrams silently omit stores; RED comes from empty `want.svg` sections, which never match non-empty output).

- [ ] **Step 2: Unit tests for store geometry**

Append to `layout/layout_test.go`:

```go
func storeLines(s *layout.Scene) []layout.Line {
	var out []layout.Line
	for _, it := range s.Items {
		if l, ok := it.(layout.Line); ok && l.Thick {
			out = append(out, l)
		}
	}
	return out
}

func TestStoreAboveSingleRow(t *testing.T) {
	s := arrange(t, "[A]\n> save\n< load\n|S|\n", layout.Config{})
	lines := storeLines(s)
	if len(lines) != 2 {
		t.Fatalf("got %d thick lines, want 2 (store glyph)", len(lines))
	}
	boxTop := layout.Margin + layout.StoreH + layout.StoreArrow
	if lines[1].Y1 != boxTop-layout.StoreArrow {
		t.Errorf("lower store line y = %d, want %d", lines[1].Y1, boxTop-layout.StoreArrow)
	}
	if lines[0].Y1 != lines[1].Y1-layout.StoreH {
		t.Errorf("store line spacing = %d, want %d", lines[1].Y1-lines[0].Y1, layout.StoreH)
	}
	var vertical []layout.Line
	for _, it := range s.Items {
		if l, ok := it.(layout.Line); ok && l.Head && l.X1 == l.X2 {
			vertical = append(vertical, l)
		}
	}
	if len(vertical) != 2 {
		t.Fatalf("got %d store arrows, want 2", len(vertical))
	}
	put, get := vertical[0], vertical[1]
	if put.X1 >= get.X1 {
		t.Errorf("put arrow (x=%d) must sit left of get arrow (x=%d)", put.X1, get.X1)
	}
	if !(put.Y1 > put.Y2) {
		t.Errorf("put arrow must point up (into store): %+v", put)
	}
	if !(get.Y2 > get.Y1) {
		t.Errorf("get arrow must point down (into box): %+v", get)
	}
}

func TestStoreBelowOnLastRow(t *testing.T) {
	src := "[One]\n[Two]\n[Three]\n> x\n|S|\n[Four]\n"
	s := arrange(t, src, layout.Config{BoxW: 160, BoxH: 60, MaxWidth: 700, FontSize: 13})
	lines := storeLines(s)
	if len(lines) != 2 {
		t.Fatalf("got %d thick lines, want 2", len(lines))
	}
	rs := rects(s)
	lastRowBottom := rs[3].Y + 60
	if lines[0].Y1 != lastRowBottom+layout.StoreArrow {
		t.Errorf("store upper line y = %d, want %d", lines[0].Y1, lastRowBottom+layout.StoreArrow)
	}
}

func TestStoreNameWidensGlyph(t *testing.T) {
	s := arrange(t, "[A]\n> x\n|A very long datastore name indeed|\n", layout.Config{})
	lines := storeLines(s)
	if w := lines[0].X2 - lines[0].X1; w <= layout.StoreW {
		t.Errorf("glyph width = %d, want > %d for long name", w, layout.StoreW)
	}
}

func TestScenesHaveNoNegativeCoords(t *testing.T) {
	srcs := []string{
		"[A]\n> s\n|Store|\n",
		"[A]\n< g\n|Store|\n[B]\n> x\n|Other|\n",
	}
	for _, src := range srcs {
		s := arrange(t, src, layout.Config{})
		for _, it := range s.Items {
			switch v := it.(type) {
			case layout.Rect:
				if v.X < 0 || v.Y < 0 {
					t.Errorf("negative rect: %+v", v)
				}
			case layout.Line:
				if v.X1 < 0 || v.Y1 < 0 || v.X2 < 0 || v.Y2 < 0 {
					t.Errorf("negative line: %+v", v)
				}
			}
		}
	}
}
```

Run: `go test ./layout` — expected: FAIL.

- [ ] **Step 3: Implement stores in layout**

Modify `layout/layout.go` — the full store support:

```go
// storeGeom is the resolved geometry of one StoreLink.
type storeGeom struct {
	w int // glyph width: max(StoreW, name width + 20)
}

// In grid: precompute per-step store widths and widen colW.
```

Concretely, in `Arrange` before building `grid`, replace `colW := c.BoxW` with:

```go
	storeW := make([][]int, len(d.Steps))
	colW := c.BoxW
	for i, st := range d.Steps {
		storeW[i] = make([]int, len(st.Stores))
		group := 0
		for j, l := range st.Stores {
			w := TextWidth(l.Name, c.Face) + 20
			if w < StoreW {
				w = StoreW
			}
			storeW[i][j] = w
			group += w
			if j > 0 {
				group += StoreGap
			}
		}
		if group > colW {
			colW = group
		}
	}
```

and pass `storeW` into `grid`. Then extend `grid.arrange`:

```go
	// Store sides: prefer above for row 0 and middle rows, below for the
	// last row; flip when a turn arrow occupies the preferred anchor.
	side := make([]int, n) // +1 above, -1 below, 0 none
	for i, st := range g.d.Steps {
		if len(st.Stores) == 0 {
			continue
		}
		r := i / g.perRow
		pref := 1
		if r == nRows-1 && nRows > 1 {
			pref = -1
		}
		topBusy := r > 0 && col[i] == startCol(r, g.perRow) && rowLen[r] > 0 && i == r*g.perRow
		bottomBusy := i == (r+1)*g.perRow-1 && i != n-1
		if pref == 1 && topBusy {
			pref = -1
		} else if pref == -1 && bottomBusy {
			pref = 1
		}
		if (pref == 1 && topBusy) || (pref == -1 && bottomBusy) {
			return nil, fmt.Errorf("layout: step %q has no free side for its datastores; increase --max-width or --per-row", st.Title)
		}
		side[i] = pref
	}
```

where `startCol(r, perRow)` is the column of a row's first step (`0` for even rows, `perRow-1` for odd). Note `i == r*g.perRow` already identifies a row's first step, so the `col[i] ==` clause is redundant — keep the simpler `topBusy := r > 0 && i == r*g.perRow`.

Row Y assignment replaces the fixed `VGap` walk with lane-aware gaps:

```go
	lane := StoreArrow + StoreH
	above := make([]bool, nRows) // any above-side stores in row r
	below := make([]bool, nRows)
	for i := range g.d.Steps {
		r := i / g.perRow
		if side[i] == 1 {
			above[r] = true
		} else if side[i] == -1 {
			below[r] = true
		}
	}
	rowY := make([]int, nRows)
	y := Margin
	if above[0] {
		y += lane
	}
	for r := 0; r < nRows; r++ {
		rowY[r] = y
		y += g.boxH
		if r < nRows-1 {
			gap := VGap
			need := 0
			if below[r] {
				need += lane
			}
			if above[r+1] {
				need += lane
			}
			if need > gap {
				gap = need
			}
			y += gap
		}
	}
	// scene height: bottom margin plus a lane if the last row has below-stores
	bottom := Margin
	if below[nRows-1] {
		bottom += lane
	}
```

Store emission per step (after emitting its box):

```go
// emitStores draws step i's store glyphs and arrows on the given side.
func (g *grid) emitStores(s *Scene, i, bx, by int) {
	st := g.d.Steps[i]
	if len(st.Stores) == 0 {
		return
	}
	group := 0
	for j, w := range g.storeW[i] {
		group += w
		if j > 0 {
			group += StoreGap
		}
	}
	x := bx + g.boxW()/2 - group/2
	for j, l := range st.Stores {
		w := g.storeW[i][j]
		g.emitStore(s, l, x, w, bx, by, g.side[i])
		x += w + StoreGap
	}
}

// emitStore draws one store glyph plus its arrows and labels.
// side: +1 = above the box, -1 = below.
func (g *grid) emitStore(s *Scene, l ast.StoreLink, x, w, bx, by, side int) {
	cx := x + w/2
	boxTop, boxBottom := by, by+g.boxH
	var line1, line2 int // upper, lower glyph line
	if side == 1 {
		line2 = boxTop - StoreArrow
		line1 = line2 - StoreH
	} else {
		line1 = boxBottom + StoreArrow
		line2 = line1 + StoreH
	}
	s.Items = append(s.Items,
		Line{X1: x, Y1: line1, X2: x + w, Y2: line1, Thick: true},
		Line{X1: x, Y1: line2, X2: x + w, Y2: line2, Thick: true},
		Text{X: cx, Y: line1 + (StoreH+g.c.FontSize)/2 - 2, S: l.Name, Anchor: Middle},
	)
	// Arrow x positions: centered when single, ±20 when both; clamped
	// into the box's horizontal span so arrows always leave the box.
	clamp := func(v int) int {
		if min := bx + 20; v < min {
			v = min
		}
		if max := bx + g.boxW() - 20; v > max {
			v = max
		}
		return v
	}
	putX, getX := clamp(cx), clamp(cx)
	if l.Put != nil && l.Get != nil {
		putX, getX = clamp(cx-20), clamp(cx+20)
	}
	if side == 1 {
		if l.Put != nil {
			g.emitStoreArrow(s, putX, boxTop, line2+Inset, l.Put.Label, l.Get != nil)
		}
		if l.Get != nil {
			g.emitStoreArrow(s, getX, line2, boxTop-Inset, l.Get.Label, false)
		}
	} else {
		if l.Put != nil {
			g.emitStoreArrow(s, putX, boxBottom, line1-Inset, l.Put.Label, l.Get != nil)
		}
		if l.Get != nil {
			g.emitStoreArrow(s, getX, line1, boxBottom+Inset, l.Get.Label, false)
		}
	}
}

// emitStoreArrow draws one vertical store arrow from y1 to y2 (head at
// y2) with its label beside it: end-anchored left of the arrow when
// labelLeft, start-anchored right of it otherwise.
func (g *grid) emitStoreArrow(s *Scene, x, y1, y2 int, label string, labelLeft bool) {
	s.Items = append(s.Items, Line{X1: x, Y1: y1, X2: x, Y2: y2, Head: true})
	if label == "" {
		return
	}
	ly := (y1+y2)/2 + 5
	if labelLeft {
		s.Items = append(s.Items, Text{X: x - LabelGap, Y: ly, S: label, Anchor: End})
	} else {
		s.Items = append(s.Items, Text{X: x + LabelGap, Y: ly, S: label, Anchor: Start})
	}
}
```

Wire `storeW`, `side` into the `grid` struct, call `emitStores` from the box loop, and use the lane-aware `rowY`/`bottom` in the height calculation. Single-row diagrams with stores get `nRows == 1` and prefer above (matches `TestStoreAboveSingleRow`).

- [ ] **Step 4: Green + seed fixtures + review**

```bash
go test ./layout
go test ./cmd/dfd -run TestScript -update
go test ./...
```

Review the four seeded SVGs carefully — this is the visual heart of the tool: glyph above/below, put left pointing at store, get right pointing at box, labels beside arrows, long store name widening the glyph, two stores side by side.

- [ ] **Step 5: Hygiene + commit**

```bash
gofmt -l . && go vet ./... && go test ./...
git add -A && git commit -m "Draw datastores with directional labeled arrows"
```

---

### Task 10: CLI IO modes — stdin/stdout, --format, default output

**Files:**
- Modify: `internal/cli/cli.go`
- Create: `cmd/dfd/testdata/script/stdin_stdout.txtar`, `cmd/dfd/testdata/script/default_output.txtar`, `cmd/dfd/testdata/script/dash_output.txtar`

**Interfaces:**
- Produces: full CLI contract — `dfd` (stdin→stdout SVG), `dfd in.dfd` (→ `in.svg`), `-o -`, `--format` override; format detection errors.

- [ ] **Step 1: Write the failing acceptance tests (RED first)**

`stdin_stdout.txtar`:

```
stdin flow.dfd
dfd
cp stdout got.svg
cmp got.svg want.svg

-- flow.dfd --
[A]
> x
[B]
-- want.svg --
```

`default_output.txtar`:

```
dfd flow.dfd
cmp flow.svg want.svg

-- flow.dfd --
[A]
[B]
-- want.svg --
```

`dash_output.txtar`:

```
dfd flow.dfd -o -
cp stdout got.svg
cmp got.svg want.svg

! dfd --format png flow.dfd -o -
stderr 'refusing to write binary PNG to stdout'

! dfd flow.dfd -o out.bmp
stderr 'cannot infer format from ".bmp"'

-- flow.dfd --
[A]
-- want.svg --
```

Run: `go test ./cmd/dfd` — expected: FAIL (`stdout not implemented yet`; format checks missing).

- [ ] **Step 2: Implement output resolution**

In `internal/cli/cli.go`, replace the output block of `run` with:

```go
	format, out, err := resolveOutput(o)
	if err != nil {
		return err
	}
	switch format {
	case "png":
		return fmt.Errorf("dfd: png output not implemented yet")
	default:
		if out == "" {
			return render.SVG(scene, stdout)
		}
		f, err := os.Create(out)
		if err != nil {
			return err
		}
		if err := render.SVG(scene, f); err != nil {
			if cerr := f.Close(); cerr != nil {
				return fmt.Errorf("%v; close: %v", err, cerr)
			}
			return err
		}
		return f.Close()
	}
```

with:

```go
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
```

- [ ] **Step 3: Green + seed + hygiene + commit**

```bash
go test ./cmd/dfd -run TestScript -update
gofmt -l . && go vet ./... && go test ./...
git add -A && git commit -m "Support stdin/stdout and output format resolution"
```

---

### Task 11: Parse-error acceptance scripts

**Files:**
- Create: `cmd/dfd/testdata/script/errors.txtar`

**Interfaces:**
- Consumes: error strings locked in Task 5; `file:line:` prefixes from `parse.Error`.

- [ ] **Step 1: Write the acceptance test (messages are known — expect near-instant green)**

`cmd/dfd/testdata/script/errors.txtar`:

```
! dfd empty.dfd
stderr 'empty\.dfd: no processes found'

! dfd dangling.dfd
stderr 'dangling\.dfd:2: arrow has no target'

! dfd back.dfd
stderr 'back\.dfd:2: .< cannot point at a process'

! dfd nostore.dfd
stderr 'nostore\.dfd:2: datastore "S" has no arrows'

! dfd unrecognized.dfd
stderr 'unrecognized\.dfd:1: unrecognized line'

! dfd missing.dfd
stderr 'no such file or directory'

-- empty.dfd --
# only a comment
-- dangling.dfd --
[A]
> x
-- back.dfd --
[A]
< x
[B]
-- nostore.dfd --
[A]
|S|
-- unrecognized.dfd --
wat
```

Note `stderr` takes a regexp: quote metacharacters (`\.`); `.< ` sidesteps quoting the literal `'`.

- [ ] **Step 2: Run, fix drift, hygiene, commit**

```bash
go test ./cmd/dfd
gofmt -l . && go vet ./... && go test ./...
git add -A && git commit -m "Cover parse errors with acceptance scripts"
```

Expected: PASS without implementation changes (exit code 1 + message already wired). If any regexp fails, fix the regexp, not the message — messages are normative from Task 5.

---

### Task 12: PNG output

**Files:**
- Create: `render/png.go`, Test: `render/png_test.go`
- Modify: `internal/cli/cli.go` (replace the `png not implemented` branch)
- Create: `cmd/dfd/testdata/script/png_output.txtar`

**Interfaces:**
- Consumes: `layout.Scene`, `typeface.New`.
- Produces: `render.PNG(s *layout.Scene, scale float64, w io.Writer) error`.

- [ ] **Step 1: Write the failing acceptance test (RED first)**

`cmd/dfd/testdata/script/png_output.txtar`:

```
dfd flow.dfd -o flow.png
exists flow.png
dfd --scale 1 flow.dfd -o small.png
exists small.png

-- flow.dfd --
[A]
> x
[B]
```

Run: `go test ./cmd/dfd` — expected: FAIL `png output not implemented yet`.

- [ ] **Step 2: Unit test for PNG determinism and content**

`render/png_test.go`:

```go
package render_test

import (
	"bytes"
	"image/png"
	"testing"

	"github.com/bilus/dfd/layout"
	"github.com/bilus/dfd/render"
)

func scene() *layout.Scene {
	return &layout.Scene{W: 300, H: 100, FontSize: 13, Items: []layout.Item{
		layout.Rect{X: 10, Y: 20, W: 160, H: 60},
		layout.Line{X1: 170, Y1: 50, X2: 250, Y2: 50, Head: true},
		layout.Text{X: 90, Y: 55, S: "hello", Anchor: layout.Middle},
	}}
}

func TestPNGDeterministicAndScaled(t *testing.T) {
	var a, b bytes.Buffer
	if err := render.PNG(scene(), 2, &a); err != nil {
		t.Fatalf("PNG: %v", err)
	}
	if err := render.PNG(scene(), 2, &b); err != nil {
		t.Fatalf("PNG: %v", err)
	}
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Error("PNG output is not deterministic")
	}
	img, err := png.Decode(&a)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if w := img.Bounds().Dx(); w != 600 {
		t.Errorf("width = %d, want 600 (300 × scale 2)", w)
	}
	// Box stroke must land as a dark pixel at the scaled top-left corner.
	r, g, bl, _ := img.At(20, 40).RGBA()
	if r > 0x4000 && g > 0x4000 && bl > 0x4000 {
		t.Errorf("expected dark stroke pixel at (20,40), got r=%x g=%x b=%x", r, g, bl)
	}
}
```

Run: `go test ./render` — expected: FAIL (PNG undefined).

- [ ] **Step 3: Implement render.PNG**

`render/png.go`:

```go
package render

import (
	"io"
	"math"

	"github.com/fogleman/gg"

	"github.com/bilus/dfd/internal/typeface"
	"github.com/bilus/dfd/layout"
)

// PNG rasterizes the scene at the given scale. All geometry is drawn in
// scaled coordinates with a face sized scale× so text stays crisp.
func PNG(s *layout.Scene, scale float64, w io.Writer) error {
	face, err := typeface.New(float64(s.FontSize) * scale)
	if err != nil {
		return err
	}
	dc := gg.NewContext(int(math.Ceil(float64(s.W)*scale)), int(math.Ceil(float64(s.H)*scale)))
	dc.SetFontFace(face)
	dc.SetRGB(1, 1, 1)
	dc.Clear()
	px := func(v int) float64 { return float64(v) * scale }
	for _, it := range s.Items {
		dc.SetRGB(0, 0, 0)
		switch v := it.(type) {
		case layout.Rect:
			dc.SetRGB(1, 1, 1)
			dc.DrawRectangle(px(v.X), px(v.Y), px(v.W), px(v.H))
			dc.FillPreserve()
			dc.SetRGB(0, 0, 0)
			dc.SetLineWidth(2 * scale)
			dc.Stroke()
		case layout.Line:
			width := 1.5
			if v.Thick {
				width = 2
			}
			dc.SetLineWidth(width * scale)
			dc.DrawLine(px(v.X1), px(v.Y1), px(v.X2), px(v.Y2))
			dc.Stroke()
			if v.Head {
				drawHead(dc, px(v.X1), px(v.Y1), px(v.X2), px(v.Y2), 8*scale)
			}
		case layout.Text:
			ax := 0.0
			switch v.Anchor {
			case layout.Middle:
				ax = 0.5
			case layout.End:
				ax = 1
			}
			dc.DrawStringAnchored(v.S, px(v.X), px(v.Y), ax, 0)
		}
	}
	return dc.EncodePNG(w)
}

// drawHead draws a filled triangular arrowhead of the given length with
// its tip at (x2, y2), oriented along the segment direction.
func drawHead(dc *gg.Context, x1, y1, x2, y2, size float64) {
	dx, dy := x2-x1, y2-y1
	l := math.Hypot(dx, dy)
	if l == 0 {
		return
	}
	ux, uy := dx/l, dy/l
	bx, by := x2-size*ux, y2-size*uy // base center
	wx, wy := -uy*size*0.4, ux*size*0.4
	dc.MoveTo(x2, y2)
	dc.LineTo(bx+wx, by+wy)
	dc.LineTo(bx-wx, by-wy)
	dc.ClosePath()
	dc.Fill()
}
```

Wire the cli branch:

```go
	case "png":
		f, err := os.Create(out)
		if err != nil {
			return err
		}
		if err := render.PNG(scene, o.scale, f); err != nil {
			if cerr := f.Close(); cerr != nil {
				return fmt.Errorf("%v; close: %v", err, cerr)
			}
			return err
		}
		return f.Close()
```

(`resolveOutput` already guarantees `out != ""` for PNG.)

- [ ] **Step 4: Green + hygiene + commit**

```bash
go test ./render ./cmd/dfd
gofmt -l . && go vet ./... && go test ./...
git add -A && git commit -m "Add PNG rendering via embedded font rasterizer"
```

---

### Task 13: Original example end-to-end + README

**Files:**
- Create: `cmd/dfd/testdata/script/original.txtar`, `README.md`
- Modify: `examples/original.svg` (regenerated by the tool), `examples/original.dfd` (only if drift found)

**Interfaces:**
- Consumes: everything.

- [ ] **Step 1: Acceptance test for the flagship example (RED first)**

`cmd/dfd/testdata/script/original.txtar` — inline copy of `examples/original.dfd`:

```
dfd --max-width 1100 flow.dfd -o flow.svg
cmp flow.svg want.svg

-- flow.dfd --
# The original whiteboard example, in dfd syntax.
[Start process]
> foo
[Continue running]
> foo
[Change something that doesn't work]
    > data
    < result
    |Somethnigs|
> foo bar
[Blah blah]
> decision
[Revert]
> something
[Store data in database]
    > something
    |Database|
> thing2
[Finish]
-- want.svg --
```

Run RED, then seed with `-update`, then **carefully review**: 4 + 3 snake, `Somethnigs` above row 1 with up `data` / down `result`, `Database` below row 2 with `something`, all six flow labels. This is the acceptance criterion for the whole project — compare against `examples/original.svg` (the hand-made reference) for structural equivalence.

- [ ] **Step 2: Regenerate the shipped example with the real tool**

```bash
go run ./cmd/dfd --max-width 1100 examples/original.dfd -o examples/original.svg
```

Open `examples/original.svg` and verify it matches the hand-made reference structurally (exact pixel positions may differ per the spec's "table wins" rule).

- [ ] **Step 3: Write README.md**

```markdown
# dfd

Turn condensed text into data-flow diagrams (SVG/PNG).

    [Start process]
    > foo
    [Continue running]
    > input
    [Store record]
        > input
        < record id
        |Database|
    > record id
    [Finish]

    dfd flow.dfd -o flow.svg

- `[Text]` — process box; document order is the flow
- `> label` / `< label` — arrows; before a process = flow arrow,
  before a `|store|` = that store's send/return arrows
- `|Text|` — datastore attached to the preceding process
- `#` — comment; indentation is cosmetic

See `docs/superpowers/specs/2026-07-23-text-to-dfd-design.md` for the full
syntax, CLI flags, and rendering rules. Example: `examples/original.dfd` →
`examples/original.svg`.

## Develop

    go test ./...                       # all tests
    go test ./cmd/dfd -run TestScript -update   # reseed txtar fixtures after
                                        # an intentional rendering change
```

- [ ] **Step 4: Full-suite hygiene + commit**

```bash
gofmt -l . && go vet ./... && go test ./...
git add -A && git commit -m "Render the original example end-to-end; add README"
```

---

## Self-review checklist (run after writing, before first execution)

1. **Spec coverage:** syntax line types (T3–T5), binding rule (T4), escapes (T5), all 16 parse errors (T5, T11), snake layout + turn arrows (T8), store sides/lanes/widening/side-by-side (T9), title wrap + box growth (T6), CLI contract incl. stdin/stdout/format/`-` (T1, T10), PNG + scale (T12), determinism (T7 fixtures, T12 test), `-update` workflow (harness T1, exercised T7+), example + README (T13). Store-side flip on turn boxes is covered by T9's implementation and guarded by the perRow=1 error path; add a dedicated txtar in T9 if review of seeded fixtures shows a gap.
2. **Placeholders:** none — every `want.svg` is intentionally empty *in the plan* because fixtures are tool-seeded via `-update` (Global Constraints), which is the spec's fixture-override workflow, not a plan placeholder.
3. **Type consistency:** `cli.Run(args, stdin, stdout, stderr) int` (T1=T7=T10=T12); `parse.Parse(r, name) (*ast.Diagram, error)` (T3+); `layout.Arrange(d, Config) (*Scene, error)`, `Scene{W,H,FontSize,Items}`, `Rect/Line{Head,Thick}/Text{Anchor}` (T6=T7=T8=T9=T12); `typeface.New(size float64) (font.Face, error)` (T6=T12); `render.SVG(s, w) error` (T7), `render.PNG(s, scale, w) error` (T12).
