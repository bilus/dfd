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
