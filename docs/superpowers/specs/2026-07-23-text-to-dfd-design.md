# dfd — text-to-DFD diagram CLI: design

Date: 2026-07-23 (themes added 2026-08-27)
Status: implemented

## Overview

`dfd` turns a condensed text description of a linear data flow into an SVG or PNG
diagram: a sequence of process boxes connected by labeled arrows, laid out in a
snake pattern (left-to-right, down, right-to-left, ...), with datastores — drawn
as two horizontal lines with the name between them — attached to individual
processes.

The rendered output mimics a hand-drawn whiteboard flow; see
`examples/original.svg` for the visual target and `examples/original.dfd` for
its source.

## Goals

- Condensed, hand-typeable format: no quoting for labels with spaces, no node IDs.
- Deterministic SVG and PNG output from the same layout.
- Single static binary, pure Go (no cgo, no external tools at runtime).
- Parse errors that point at the offending line and say how to fix it.
- Feature-level golden tests with a one-flag fixture-update workflow.

## Non-goals (v1)

- Branching or arbitrary graphs. The flow is strictly one linear sequence.
- Arbitrary styling: user-defined colours or fonts. Themes are a closed
  set (see Themes below); `--theme` picks one, nothing is configurable
  past that.
- Layout hints in the syntax (store side, row breaks). Layout is fully automatic.
- Merging datastore symbols across different processes (same name = separate symbols).
- Config files, watch mode, editor integrations, Mermaid/DOT import or export.

## Syntax

A document is a sequence of lines. Leading and trailing whitespace on every line
is ignored — indentation is purely cosmetic. Encoding is UTF-8; a trailing `\r`
(CRLF input) is stripped.

### Line types

| Line (after trimming)   | Meaning                                            |
| ----------------------- | -------------------------------------------------- |
| `[Text]`                | Process box. Document order defines the flow.      |
| `\|Text\|`              | Datastore attached to the nearest preceding process. |
| `>` or `> label`        | Forward arrow, optional label.                     |
| `<` or `< label`        | Return arrow, optional label.                      |
| `# ...`                 | Comment, ignored.                                  |
| (empty)                 | Ignored.                                           |

Comments and blank lines are transparent: they may appear anywhere, including
between an arrow line and the line it binds to.

Process and store names are the bracketed text, trimmed (`[ Foo ]` → `Foo`).
A process or store line must contain nothing else after the closing bracket.

### Binding rule

Consecutive arrow lines form an *arrow run*. A run binds to the next
process-or-store line:

- **Next line is a process** — the run is the flow arrow from the previous
  process into it. The run must be exactly one `>` (its label labels the flow
  arrow). Adjacent processes with no run between them get an unlabeled arrow.
- **Next line is a store** — the run describes the connection between the
  nearest preceding process and that store: at most one `>` (process sends to
  store) and at most one `<` (store returns to process), in either order, each
  with its own optional label.

Every `|Store|` line produces one drawn symbol. A process may have any number
of store lines; each needs its own arrow run.

### Labels

An arrow label is everything after the `>`/`<` on that line, trimmed, taken
verbatim — spaces and punctuation need no quoting. Labels render on one line.

### Multi-line constructs

- A `[process]` may span source lines: the bracket stays open until an
  unescaped `]` ends a line, and each source line becomes one forced
  title line (trimmed). Lines inside an open bracket are content, even
  ones starting with `#` or blank ones. Unterminated at EOF reports
  `missing closing "]"` at the opening line.
- An arrow label continues onto following lines that are not blank and
  do not start a construct (`[`, `|`, `>`, `<`, `#`); each continuation
  line is one forced label line. A blank line or comment ends the
  continuation; a continuation with no preceding arrow is the
  unrecognized-line error.
- Datastore names cannot span lines (the glyph height is fixed); the
  missing-close error message says so.
- Rendering: forced breaks are kept; title and flow-label segments still
  auto-wrap individually if too wide.

### Escapes

Inside process names: `\]` for a literal `]`. Inside store names: `\|` for a
literal `|`. `\\` is a literal backslash in both. Nothing else is escaped.

### Parse errors

All reported as `file:line: message` (see CLI section). The complete list:

| Condition                                             | Message sketch                                             |
| ----------------------------------------------------- | ---------------------------------------------------------- |
| Document has no process                               | no processes found                                         |
| Arrow run before the first process                    | arrow has no source process                                |
| Arrow run at end of file                              | arrow has no target                                        |
| `<` in a run that binds to a process                  | '<' cannot point at a process; return arrows only precede a \|store\| line |
| More than one `>` in a run binding to a process       | multiple flow arrows between processes                     |
| Duplicate `>` or `<` in a run binding to a store      | duplicate '>' arrow for store 'X'                          |
| Store line with an empty arrow run                    | datastore 'X' has no arrows; add > and/or < lines before it |
| Store before any process                              | datastore before any process                               |
| Missing closing `]` / `\|`                            | missing closing ']'                                        |
| Text after closing bracket                            | unexpected text after ']'                                  |
| Empty name (`[]`, `\|\|`)                             | empty process/datastore name                               |
| Unrecognizable line                                   | unrecognized line; expected [process], \|store\|, > or < arrow, or # comment |

### Examples

Minimal store round-trip:

```
[Initial step]
> input
[Store in database]
    > input
    < record id
    |Somethings|
> record id
[Return to user]
```

Full example: `examples/original.dfd` (renders to `examples/original.svg`).

## Rendering

### Visual constants

Normative defaults; several are CLI-configurable. The hand-made
`examples/original.svg` illustrates the style; where its ad-hoc numbers differ
from this table, the table wins, and txtar fixtures are seeded from the
implemented renderer (visually reviewed), not from the hand-made file.

| Constant       | Default | Flag          | Notes                                    |
| -------------- | ------- | ------------- | ---------------------------------------- |
| Box size       | 160×60  | `--box WxH`   | Height grows uniformly if any title needs more wrapped lines. |
| Box stroke     | 2px     | —             | Black, sharp corners, white fill.        |
| Arrow stroke   | 1.5px   | —             | Filled triangular head ~8×8.             |
| Font size      | 13px    | `--font-size` | SVG: `Helvetica, Arial, sans-serif`; PNG: embedded Go Regular. |
| Store symbol   | 150 wide, lines 36 apart | — | Widens to fit long names + 20px padding. |
| Store arrow length | 64px | —            | Vertical, between box edge and near store line. |
| Column gap     | 90px    | —             | Horizontal space between boxes; holds flow labels. Uniform, and grows only for an unwrappable word (see Flow label invariant). |
| Base row gap   | 90px    | —             | Grows when stores occupy the gap.        |
| Canvas margin  | 40px    | —             | White background, both formats. The canvas grows past the grid when an arrow label overhangs a column, keeping both margins equal. |
| PNG scale      | 2       | `--scale`     | Resolution multiplier.                   |

### Placement rules

- Flow arrows run horizontally between adjacent boxes in a row, arrowhead at the
  target edge; label centered above the arrow's midpoint. Labels wrap at word
  boundaries to fit the column gap (8px clearance each side), stacking upward;
  when a single word cannot fit, the column gap widens to that word instead —
  words are never broken mid-word in labels. The snake-turn arrow
  runs vertically from the row's last box down to the next row's first box;
  label to its right, vertically centered.
- A store with **one** arrow draws it at the box's horizontal center. A store
  with **two** arrows draws `>` at center−20 and `<` at center+20. Arrowheads
  point at the store line (for `>`) or the box edge (for `<`).
- Store arrow labels sit beside their arrow at half-height: the left arrow's
  label to its left (end-anchored), the right arrow's to its right
  (start-anchored). A single centered arrow labels to its right.
- Box titles wrap at word boundaries to fit the box interior (12px side
  padding); a single overlong word breaks mid-word. Store names and arrow
  labels never wrap.

## Themes

`--theme` selects one of two painting recipes. `default` is the original
black-on-white look and is byte-for-byte unchanged by the theme layer.
`plex` uses a #F2F4F6 canvas, #FFFFFF boxes with a 1.25 #12161A outline
and 4px corners, a 3px #63489E accent bar inset by the corner radius on
each box's left edge, 1.4 violet arrows with 7px heads, IBM Plex Sans
SemiBold titles at base+2, and IBM Plex Mono arrow labels at base-1.

Scene text carries a role (title, arrow label, datastore name) and a
theme styles each one separately. Because layout measures with the
theme's faces, a theme can move things as well as recolour them:

- `LabelOnLine` (plex only) centres flow and turn labels on their arrow
  instead of setting them beside it, and `LabelChip` masks the line
  behind the text with a canvas-coloured rounded rect.
- Labels drawn on a line are never auto-wrapped. The column gap instead
  grows to `label width + 2*ChipPadX + 2*LabelStub` so the chip cannot
  swallow the arrowhead. Explicit line breaks remain the way to shorten
  a long label under that theme.

Every face is embedded in the binary (Go Regular for `default`, IBM Plex
for `plex`, the latter under the SIL Open Font License 1.1), so neither
measurement nor PNG output depends on installed system fonts. SVG names
the font stack and leaves substitution to the viewer.

## Flow label invariant

This rule holds for every theme, present and future, and is enforced by
tests parameterised over `theme.Names()`:

1. Every column gap in a diagram is **the same width**.
2. That width is `HGap` (90px), the standard gap.
3. It grows past `HGap` only when a **single word** cannot be wrapped to
   fit, and then only to exactly what that word needs:
   `widest word + LabelInset(theme)`.
4. Flow labels always wrap at word boundaries to `gap - LabelInset`.
   A theme may not opt out of wrapping.

`LabelInset` is the room a label needs beyond its own text and is the
single source of truth for both halves of the rule, so the gap and the
wrap width can never disagree: `2*LabelPad` for labels set above the
line, and `2*ChipPadX + 2*LabelStub` for labels centred on it, the stub
being the arrow left visible either side of the masking chip.

A label chip is exactly one line tall (`Style.LineH()`), so the chips of
a wrapped label tile instead of erasing the line above.

Explicit line breaks in a label are honoured on top of this: each
segment wraps independently.

## Layout algorithm

Input: `ast.Diagram` + config. Output: `layout.Scene`, a display list of
primitives (rects, lines with optional arrowheads, anchored text runs) with a
computed canvas size. All decisions are deterministic.

1. **Columns.** Uniform column width = max over steps of (box width, widest
   store group). Boxes-per-row = `--per-row`, default 4. It is a fixed count,
   not a pixel budget, so a theme that draws wider gaps does not change how
   many boxes a row holds.
2. **Snake.** Step i goes to row `i / perRow`. Even rows run left-to-right,
   odd rows right-to-left, so a row's first step sits directly under the
   previous row's last step; that pair is joined by the vertical turn arrow.
3. **Store sides.** Preferred side: above for row 0, below for the last row,
   above for middle rows. If the preferred side's anchor is occupied by a turn
   arrow (a row-first box's top, a row-last box's bottom), use the other side.
   If both are occupied (only possible when perRow is 1), fail with an error
   suggesting a larger `--per-row`.
4. **Vertical spacing.** Each inter-row gap and outer margin grows to fit what
   it contains: turn arrows need the base gap; each store lane adds store
   height + store arrow length. Stores in the same gap from adjacent rows
   stack without overlap.
5. **Emit.** Boxes, store glyphs (two lines + name), arrows, labels — as
   primitives with final coordinates.

Multiple stores on one step share its side, placed side by side 20px apart,
centered as a group on the box (their group width already sized the columns in
step 1).

## Architecture

```
dfd/                         module github.com/bilus/dfd
  cmd/dfd/main.go            func main() { os.Exit(cli.Run(...)) } — nothing else
  internal/cli/              Run(args, stdin, stdout, stderr) int — flag parsing, IO, wiring
  ast/                       Diagram, Step, StoreLink, Arrow
  parse/                     Parse(r io.Reader, name string) (*ast.Diagram, error)
  layout/                    Arrange(*ast.Diagram, Config) (*Scene, error)
  render/                    SVG(*Scene, io.Writer) error; PNG(*Scene, Config, io.Writer) error
  examples/                  original.dfd, original.svg
```

### Key types

```go
// ast
type Diagram struct{ Steps []Step }            // New() enforces: ≥1 step, Steps[0].In == ""
type Step struct {
    Title  string
    In     string                              // label of the incoming flow arrow; "" = unlabeled; unused on the first step
    Stores []StoreLink
}
type StoreLink struct {
    Name     string
    Put, Get *Arrow                            // NewStoreLink() enforces: at least one non-nil
}
type Arrow struct{ Label string }

// layout
type Scene struct {
    W, H  float64
    Items []Item                               // Rect | Line{Head bool} | Text{Anchor}
}
```

Constructors validate invariants so a `Diagram` or `StoreLink` that exists is
well-formed regardless of who built it; the parser is not trusted to be the
only careful caller.

### Determinism

Both renderers draw the same `Scene`. Coordinates are computed in float64 from
integer constants and emitted with a fixed format, so SVG bytes are stable.
PNG uses `fogleman/gg` with the embedded `golang.org/x/image/font/gofont/goregular`
TTF — no system fonts, no timestamps — so PNG bytes are stable too. The same
goregular metrics drive title wrapping in layout, keeping SVG and PNG text
consistent (SVG viewers substitute Helvetica/Arial; boxes are sized from
metrics, so minor glyph-width differences cannot overflow them).

## CLI

```
dfd [flags] [input.dfd]

  (no input)        read stdin, write SVG to stdout
  input.dfd         write input.svg next to the input
  -o out.svg|.png   explicit output; format from extension; "-" = stdout
  --format svg|png  override format detection (required for PNG on stdout)
  --box WxH         box size (default 160x60)
  --per-row N       boxes per row (default 4)
  --font-size N     label font size (default 13)
  --scale N         PNG resolution multiplier (default 2)
  --version
```

Exit codes: 0 success; 1 for usage, parse, layout, or IO errors. Errors go to
stderr as `file:line: message` (stdin shows `<stdin>`), e.g.:

```
flow.dfd:7: '<' cannot point at a process; return arrows only precede a |store| line
```

## Testing

Per the repo's Go rules: all tests in external test packages, no discarded
errors.

### Feature tests — testscript + txtar

Every syntax and rendering feature gets one self-contained script under
`cmd/dfd/testdata/script/*.txtar`: labeled/unlabeled flow arrows, store
write-only / read-only / read+write, multiple stores per step, snake wrapping
at various `--per-row`, title wrapping, escapes, comments, stdin/stdout, and
every parse error (via `! dfd bad.dfd` + `stderr` checks).

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
<svg ...>
```

`cmd/dfd/script_test.go` (package `main_test`) registers the `dfd` command
with testscript backed by `cli.Run`, so scripts run in-process without
building a binary.

### Fixture override

A `-update` test flag maps to `testscript.Params.UpdateScripts`: after an
intentional renderer change, `go test ./cmd/dfd -update` rewrites the
`want.svg` sections inside the txtar archives in place, and the developer
reviews the git diff (plus a visual check of `examples/original.svg`,
regenerated the same way).

### Unit tests

- `parse`: table-driven, asserting diagrams, messages, and line numbers.
- `layout`: invariants on generated scenes — no overlapping rects, snake
  ordering, store side rules, gap growth.
- `render`: one PNG golden test comparing bytes (stable per the determinism
  notes); SVG is covered by the txtar fixtures.

## Build order

1. Module scaffold, `ast` + `parse` with unit tests.
2. `layout` with invariant tests.
3. `render.SVG` + `cli.Run` + testscript harness and feature fixtures.
4. `render.PNG`, remaining flags, `examples/` regeneration, README.
