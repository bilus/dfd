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
