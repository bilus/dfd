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
