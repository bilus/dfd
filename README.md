# dfd

Turn condensed text into data-flow diagrams (SVG/PNG).

![Data-flow diagram with seven process boxes snaking across two rows, a
datastore read and written above the third box, and a datastore written
below the sixth](examples/original.png)

That picture is this text ([examples/original.dfd](examples/original.dfd)):

```
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
```

rendered with:

```
dfd --max-width 1100 examples/original.dfd -o examples/original.png
```

Other ways to run it:

```
dfd flow.dfd -o flow.svg
dfd flow.dfd              # writes flow.svg next to the input
cat flow.dfd | dfd > flow.svg
```

## Themes

`--theme plex` swaps the hand-drawn look for a cool-grey canvas, hairline
boxes with a violet accent spine, and monospace arrow labels centred on
their connector:

![The same diagram in the plex theme](examples/original-plex.png)

```
dfd --theme plex --max-width 1100 examples/original.dfd -o out.png
```

Themes change measurement as well as paint, so the two layouts differ:
plex sizes column gaps to fit a whole label plus a visible stub of arrow
either side, and never auto-wraps a label sitting on a line. Use explicit
line breaks to shorten one. IBM Plex Sans and Mono are embedded in the
binary (SIL Open Font License 1.1, see `internal/typeface/fonts/OFL.txt`),
so PNG output is identical everywhere with no fonts installed.

## Syntax

- `[Text]` — process box; document order is the flow
- `> label` / `< label` — arrows, label optional; before a process it is
  the flow arrow into it, before a `|store|` line it is that store's
  send (`>`) / return (`<`) arrow
- `|Text|` — datastore (two horizontal lines) attached to the nearest
  preceding process
- `#` — comment; blank lines and indentation are cosmetic
- `\]`, `\|`, `\\` — literal brackets in names
- line breaks: keep a `[bracket` open across lines, or continue an
  arrow label on the next line — each source line renders as one line

Layout is automatic: boxes snake left-to-right then right-to-left across
rows, datastores sit above or below their process, long titles wrap.

## Flags

```
-o out.svg|.png   output file; format from extension; "-" = stdout
--format svg|png  override format detection
--box WxH         box size (default 160x60)
--max-width N     target canvas width for row wrapping (default 1000)
--per-row N       fixed boxes per row (overrides --max-width)
--font-size N     label font size (default 13)
--scale N         PNG resolution multiplier (default 2)
--theme NAME      default or plex
```

Full syntax, error catalogue, and rendering rules:
[docs/superpowers/specs/2026-07-23-text-to-dfd-design.md](docs/superpowers/specs/2026-07-23-text-to-dfd-design.md).
The examples above are also checked in as SVG:
[examples/original.svg](examples/original.svg) and
[examples/original-plex.svg](examples/original-plex.svg).

## Develop

```
go test ./...                                # all tests
go test ./cmd/dfd -run TestScript -update    # reseed txtar fixtures after an
                                             # intentional rendering change
```

Acceptance tests live in `cmd/dfd/testdata/script/*.txtar` (testscript);
each inlines a `.dfd` source and the expected SVG.
