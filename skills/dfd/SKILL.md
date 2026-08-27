---
name: dfd
description: Use when drawing a data flow diagram, a linear process or pipeline picture with boxes, labelled arrows, datastores and external entities, when asked for a whiteboard-style flow diagram, or when a .dfd file or the dfd command appears.
---

# dfd

Turns a condensed text description of a **linear** data flow into SVG or PNG.
One line per node, in flow order; layout is automatic.

## When to use

- A flow that runs start to finish: request handling, a pipeline, a job
- Boxes with labelled arrows, plus datastores and external systems
- The user wants a picture they can keep in git and diff

**Not for:** branching, conditionals, loops, or arbitrary graphs. dfd has no
branch syntax. Reach for Mermaid or Graphviz instead.

## Syntax

| Line | Means |
| --- | --- |
| `[Text]` | Process box. Document order is the flow. |
| `{Text}` | External entity: a source or sink outside the system. May sit anywhere in the flow, not only at the ends. |
| `\|Text\|` | Datastore, attached to the process above it. |
| `> label` | Forward arrow. Before a process it is the flow arrow; before a datastore it is the write. |
| `< label` | Return arrow. Only before a datastore: the read. |
| `Alias := Label` | Inside any bracket: shorthand, so a bare `Alias` later means that label. |
| `# ...` | Comment. |

Blank lines and indentation are cosmetic. Labels need no quoting. Escape a
literal bracket or marker with a backslash: `\]`, `\}`, `\|`, `\:=`, `\\`.

A bracket may span lines, and an arrow label continues on the next line; each
source line becomes one rendered line.

## Example

```dfd
{Client}
> request
[Authenticate]
    > lookup
    < profile
    |Users|
> claims
[Render page]
> html
{CDN}
```

```bash
dfd flow.dfd -o flow.svg
```

## Running it

`dfd in.dfd -o out.svg` or `-o out.png`; the extension picks the format.
Reads stdin and writes stdout when you leave them off. `dfd --help` prints
the flags with one dash; Go accepts either, and this page uses two.

| Flag | Does |
| --- | --- |
| `--theme default\|plex` | `default` is hand-drawn black on white; `plex` is a grey canvas with a violet accent. |
| `--number` | Numbers processes 1, 2, 3 and datastores D1, D2. Off unless asked. |
| `--number-prefix 2.` | Levels the process numbers to 2.1, 2.2 for a child diagram. |
| `--per-row N` | Boxes per row before the flow snakes back (default 4). |
| `--box WxH`, `--font-size N`, `--scale N` | Box size, text size, PNG resolution. |

## Identity

A node is identified by its label. The same datastore mentioned by two
processes is one store and keeps one number; the alias saves retyping:

```dfd
[Register user]
    > row
    |R := Registration|
> id
[Confirm]
    < row
    |R|
```

With `--number` both glyphs read `D1 Registration`.

Identity is about numbering, not drawing: a repeated node is drawn again
wherever it appears, which is how a DFD shows one store or one customer
touching several steps. Entities are drawn again too and never take a
number.

## In org-mode

```org
#+begin_src dfd :file flow.svg :theme plex :number yes
[Start]
> go
[Finish]
#+end_src
```

`:file` is required. Renderer flags have header arguments of the same name;
`:cmdline "..."` passes anything unmapped. Declaring a `:var` turns on `$name`
substitution in the body.

## Common mistakes

| Mistake | What happens |
| --- | --- |
| `<` before a process | Error. Return arrows only precede a datastore. |
| A datastore under `{Entity}` | Error. Data cannot flow straight between an entity and a store; put a process between them. |
| Expecting a branch | There is none. Split into several diagrams. |
| Expecting a reply from an external system | There is no return arrow from `{Entity}`; `<` only precedes a datastore. A call that answers back is drawn as the flow passing through the entity and on to the next process. Model a true round trip as two diagrams, or accept the pass-through reading. |
| A datastore name over two lines | Error. Only processes and entities span lines. |
| Very long arrow labels | They wrap at word boundaries. One unbreakable word widens every column gap, so keep labels short or break them yourself. |
