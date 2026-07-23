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

// arrowLine is one pending `>`/`<` line awaiting its binding target.
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
