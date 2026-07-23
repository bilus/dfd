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
		name             string
		src              string
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
		{"missing close pipe", "[A]\n> x\n|S\n", "test.dfd:3: missing closing \"|\" (datastore names cannot span lines)"},
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

func TestParseMultiLineTitle(t *testing.T) {
	d := mustParse(t, "[This is a box\n line 2]\n[Another box]\n")
	if got := d.Steps[0].Title; got != "This is a box\nline 2" {
		t.Errorf("title = %q, want lines joined with \\n and trimmed", got)
	}
}

func TestParseArrowLabelContinuation(t *testing.T) {
	d := mustParse(t, "[A]\n> line 1\n  line 2\n[B]\n")
	if got := d.Steps[1].In; got != "line 1\nline 2" {
		t.Errorf("flow label = %q, want %q", got, "line 1\nline 2")
	}
}

func TestParseStoreArrowLabelContinuations(t *testing.T) {
	d := mustParse(t, "[A]\n> aaa\n  bbb\n< ccc\n  ddd\n|S|\n")
	l := d.Steps[0].Stores[0]
	if l.Put == nil || l.Put.Label != "aaa\nbbb" {
		t.Errorf("put label = %+v, want aaa\\nbbb", l.Put)
	}
	if l.Get == nil || l.Get.Label != "ccc\nddd" {
		t.Errorf("get label = %+v, want ccc\\nddd", l.Get)
	}
}

func TestParseCommentInsideOpenBracketIsContent(t *testing.T) {
	d := mustParse(t, "[a\n# not a comment\nb]\n")
	if got := d.Steps[0].Title; got != "a\n# not a comment\nb" {
		t.Errorf("title = %q, want the hash line kept as content", got)
	}
}

func TestParseMultiLineErrors(t *testing.T) {
	cases := []struct {
		name, src, want string
	}{
		{"unterminated at eof", "[A\nstill open\n", "test.dfd:1: missing closing \"]\""},
		{"orphan continuation", "[A]\nwat\n", "test.dfd:2: unrecognized line; expected [process], |store|, > or < arrow, or # comment"},
		{"blank ends continuation", "[A]\n> x\n\ncont\n[B]\n", "test.dfd:4: unrecognized line; expected [process], |store|, > or < arrow, or # comment"},
		{"trailing text on closing line", "[A\nb] tail\n", "test.dfd:2: unexpected text after \"]\""},
		{"store name spans lines", "[A]\n> x\n|Store\nname|\n", "test.dfd:3: missing closing \"|\" (datastore names cannot span lines)"},
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
