package layout_test

import (
	"strings"
	"testing"

	"github.com/bilus/dfd/layout"
	"github.com/bilus/dfd/parse"
)

func arrange(t *testing.T, src string, c layout.Config) *layout.Scene {
	t.Helper()
	d, err := parse.Parse(strings.NewReader(src), "test.dfd")
	if err != nil {
		t.Fatalf("parse: %v", err)
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

func TestTwoBoxesConnectedByArrow(t *testing.T) {
	s := arrange(t, "[First]\n[Second]\n", layout.Config{})
	var lines []layout.Line
	for _, it := range s.Items {
		if l, ok := it.(layout.Line); ok {
			lines = append(lines, l)
		}
	}
	if len(lines) != 1 {
		t.Fatalf("got %d lines, want 1 arrow", len(lines))
	}
	want := layout.Line{X1: 200, Y1: 70, X2: 287, Y2: 70, Head: true}
	if lines[0] != want {
		t.Errorf("arrow = %+v, want %+v", lines[0], want)
	}
	if s.W != 2*layout.Margin+2*160+layout.HGap {
		t.Errorf("scene w = %d", s.W)
	}
}

func TestFlowArrowLabel(t *testing.T) {
	s := arrange(t, "[A]\n> go\n[B]\n", layout.Config{})
	var got *layout.Text
	for _, it := range s.Items {
		if tx, ok := it.(layout.Text); ok && tx.S == "go" {
			got = &tx
			break
		}
	}
	if got == nil {
		t.Fatal("flow label not in scene")
	}
	want := layout.Text{X: 245, Y: 62, S: "go", Anchor: layout.Middle}
	if *got != want {
		t.Errorf("label = %+v, want %+v (gap midpoint, LabelGap above)", *got, want)
	}
}

func TestStoreWriteAboveBox(t *testing.T) {
	s := arrange(t, "[Save it]\n> something\n|Database|\n", layout.Config{})
	var thick []layout.Line
	var arrows []layout.Line
	for _, it := range s.Items {
		if l, ok := it.(layout.Line); ok {
			if l.Thick {
				thick = append(thick, l)
			} else if l.Head {
				arrows = append(arrows, l)
			}
		}
	}
	if len(thick) != 2 {
		t.Fatalf("got %d thick lines, want 2 (store glyph)", len(thick))
	}
	upper := layout.Line{X1: 45, Y1: 40, X2: 195, Y2: 40, Thick: true}
	lower := layout.Line{X1: 45, Y1: 76, X2: 195, Y2: 76, Thick: true}
	if thick[0] != upper || thick[1] != lower {
		t.Errorf("glyph lines = %+v %+v\nwant %+v %+v", thick[0], thick[1], upper, lower)
	}
	if len(arrows) != 1 {
		t.Fatalf("got %d arrows, want 1 (put)", len(arrows))
	}
	put := layout.Line{X1: 120, Y1: 140, X2: 120, Y2: 79, Head: true}
	if arrows[0] != put {
		t.Errorf("put arrow = %+v, want %+v (box top up to store)", arrows[0], put)
	}
	wantName := layout.Text{X: 120, Y: 62, S: "Database", Anchor: layout.Middle}
	wantLabel := layout.Text{X: 128, Y: 113, S: "something", Anchor: layout.Start}
	foundName, foundLabel := false, false
	for _, it := range s.Items {
		if tx, ok := it.(layout.Text); ok {
			if tx == wantName {
				foundName = true
			}
			if tx == wantLabel {
				foundLabel = true
			}
		}
	}
	if !foundName || !foundLabel {
		t.Errorf("missing store name (%v) or arrow label (%v)", foundName, foundLabel)
	}
	var box layout.Rect
	for _, it := range s.Items {
		if r, ok := it.(layout.Rect); ok {
			box = r
		}
	}
	if box.Y != 140 {
		t.Errorf("box y = %d, want 140 (top lane holds the store)", box.Y)
	}
	if s.H != 240 {
		t.Errorf("scene h = %d, want 240", s.H)
	}
}

func sceneTexts(s *layout.Scene) map[layout.Text]bool {
	out := map[layout.Text]bool{}
	for _, it := range s.Items {
		if tx, ok := it.(layout.Text); ok {
			out[tx] = true
		}
	}
	return out
}

func headArrows(s *layout.Scene) []layout.Line {
	var out []layout.Line
	for _, it := range s.Items {
		if l, ok := it.(layout.Line); ok && l.Head && !l.Thick {
			out = append(out, l)
		}
	}
	return out
}

func TestStoreReadWriteArrowPair(t *testing.T) {
	s := arrange(t, "[Store in database]\n> input\n< record id\n|Somethings|\n", layout.Config{})
	arrows := headArrows(s)
	if len(arrows) != 2 {
		t.Fatalf("got %d arrows, want put + get", len(arrows))
	}
	put := layout.Line{X1: 100, Y1: 140, X2: 100, Y2: 79, Head: true}
	get := layout.Line{X1: 140, Y1: 76, X2: 140, Y2: 137, Head: true}
	if arrows[0] != put || arrows[1] != get {
		t.Errorf("arrows = %+v %+v\nwant put %+v get %+v", arrows[0], arrows[1], put, get)
	}
	texts := sceneTexts(s)
	for _, want := range []layout.Text{
		{X: 92, Y: 113, S: "input", Anchor: layout.End},
		{X: 148, Y: 113, S: "record id", Anchor: layout.Start},
	} {
		if !texts[want] {
			t.Errorf("missing label %+v", want)
		}
	}
}

func TestStoreReadOnlyCenteredArrow(t *testing.T) {
	s := arrange(t, "[Load it]\n< rows\n|Database|\n", layout.Config{})
	arrows := headArrows(s)
	if len(arrows) != 1 {
		t.Fatalf("got %d arrows, want 1 (get)", len(arrows))
	}
	get := layout.Line{X1: 120, Y1: 76, X2: 120, Y2: 137, Head: true}
	if arrows[0] != get {
		t.Errorf("get arrow = %+v, want %+v (centered, store down to box)", arrows[0], get)
	}
	if !sceneTexts(s)[layout.Text{X: 128, Y: 113, S: "rows", Anchor: layout.Start}] {
		t.Error("missing get label right of centered arrow")
	}
}

func TestSingleBoxScene(t *testing.T) {
	s := arrange(t, "[Hello]\n", layout.Config{})
	if s.W != 240 || s.H != 140 || s.FontSize != 13 {
		t.Errorf("scene = %dx%d font %d, want 240x140 font 13", s.W, s.H, s.FontSize)
	}
	if len(s.Items) != 2 {
		t.Fatalf("got %d items, want rect + text", len(s.Items))
	}
	r, ok := s.Items[0].(layout.Rect)
	if !ok || r != (layout.Rect{X: 40, Y: 40, W: 160, H: 60}) {
		t.Errorf("rect = %+v, want {40 40 160 60}", s.Items[0])
	}
	tx, ok := s.Items[1].(layout.Text)
	if !ok || tx != (layout.Text{X: 120, Y: 75, S: "Hello", Anchor: layout.Middle}) {
		t.Errorf("text = %+v, want centered baseline at (120,75)", s.Items[1])
	}
}
