package render_test

import (
	"strings"
	"testing"

	"github.com/bilus/dfd/layout"
	"github.com/bilus/dfd/render"
)

func TestSVGEmptyScene(t *testing.T) {
	s := &layout.Scene{W: 100, H: 50, FontSize: 13}
	var b strings.Builder
	if err := render.SVG(s, &b); err != nil {
		t.Fatalf("SVG: %v", err)
	}
	want := `<svg viewBox="0 0 100 50" width="100" height="50" xmlns="http://www.w3.org/2000/svg" font-family="Helvetica, Arial, sans-serif" font-size="13">
  <defs>
    <marker id="ah" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="8" markerHeight="8" orient="auto">
      <path d="M0,0L10,5L0,10z" fill="#000"/>
    </marker>
  </defs>
  <rect x="0" y="0" width="100" height="50" fill="#fff"/>
</svg>
`
	if b.String() != want {
		t.Fatalf("empty scene output:\n%s\nwant:\n%s", b.String(), want)
	}
}

func TestSVGArrowLine(t *testing.T) {
	s := &layout.Scene{W: 300, H: 100, FontSize: 13, Items: []layout.Item{
		layout.Line{X1: 200, Y1: 70, X2: 287, Y2: 70, Head: true},
		layout.Line{X1: 10, Y1: 10, X2: 20, Y2: 10},
	}}
	var b strings.Builder
	if err := render.SVG(s, &b); err != nil {
		t.Fatalf("SVG: %v", err)
	}
	out := b.String()
	for _, want := range []string{
		`  <line x1="200" y1="70" x2="287" y2="70" stroke="#000" stroke-width="1.5" marker-end="url(#ah)"/>` + "\n",
		`  <line x1="10" y1="10" x2="20" y2="10" stroke="#000" stroke-width="1.5"/>` + "\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\ngot:\n%s", want, out)
		}
	}
}

func TestSVGThickLine(t *testing.T) {
	s := &layout.Scene{W: 300, H: 100, FontSize: 13, Items: []layout.Item{
		layout.Line{X1: 45, Y1: 40, X2: 195, Y2: 40, Thick: true},
	}}
	var b strings.Builder
	if err := render.SVG(s, &b); err != nil {
		t.Fatalf("SVG: %v", err)
	}
	out := b.String()
	for _, want := range []string{
		`  <line x1="45" y1="40" x2="195" y2="40" stroke="#000" stroke-width="2"/>` + "\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\ngot:\n%s", want, out)
		}
	}
}

func TestSVGRectAndText(t *testing.T) {
	s := &layout.Scene{W: 240, H: 140, FontSize: 13, Items: []layout.Item{
		layout.Rect{X: 40, Y: 40, W: 160, H: 60},
		layout.Text{X: 120, Y: 75, S: "Hello & <World>", Anchor: layout.Middle},
		layout.Text{X: 10, Y: 20, S: "left", Anchor: layout.Start},
		layout.Text{X: 230, Y: 20, S: "right", Anchor: layout.End},
	}}
	var b strings.Builder
	if err := render.SVG(s, &b); err != nil {
		t.Fatalf("SVG: %v", err)
	}
	out := b.String()
	for _, want := range []string{
		`  <rect x="40" y="40" width="160" height="60" fill="#fff" stroke="#000" stroke-width="2"/>` + "\n",
		`  <text x="120" y="75" fill="#000" text-anchor="middle">Hello &amp; &lt;World&gt;</text>` + "\n",
		`  <text x="10" y="20" fill="#000">left</text>` + "\n",
		`  <text x="230" y="20" fill="#000" text-anchor="end">right</text>` + "\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\ngot:\n%s", want, out)
		}
	}
}
