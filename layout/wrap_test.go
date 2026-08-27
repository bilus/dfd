package layout_test

import (
	"strings"
	"testing"

	"github.com/bilus/dfd/layout"
	"github.com/bilus/dfd/theme"
)

func TestWrapTextSplitsAtWordBoundaries(t *testing.T) {
	face := mustFace(t)
	got := layout.WrapText("Change something that doesn't work", 136, face)
	want := []string{"Change something that", "doesn't work"}
	if len(got) != len(want) {
		t.Fatalf("WrapText = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("WrapText = %q, want %q", got, want)
		}
	}
}

func TestWrapTextEveryLineFits(t *testing.T) {
	face := mustFace(t)
	cases := []string{
		"short",
		"a few ordinary words wrapping around",
		"supercalifragilisticexpialidociousandthensome overlong",
		strings.Repeat("wide ", 30),
	}
	for _, s := range cases {
		for _, line := range layout.WrapText(s, 136, face) {
			if line == "" {
				t.Errorf("WrapText(%q) produced an empty line", s)
			}
			if w := layout.TextWidth(line, face); w > 136 {
				t.Errorf("line %q is %dpx, exceeds 136", line, w)
			}
		}
	}
}

func TestWrappedTitleBaselinesAndBoxGrowth(t *testing.T) {
	s := arrange(t, "[Change something that doesn't work]\n", layout.Config{})
	texts := sceneTexts(s)
	for _, want := range []layout.Text{
		{X: 120, Y: 67, S: "Change something that", Anchor: layout.Middle, Role: theme.Title},
		{X: 120, Y: 84, S: "doesn't work", Anchor: layout.Middle, Role: theme.Title},
	} {
		if !texts[want] {
			t.Errorf("missing title line %+v", want)
		}
	}
	rs := rects(s)
	if rs[0].H != 60 {
		t.Errorf("box h = %d, want 60 (two lines still fit)", rs[0].H)
	}

	small := arrange(t, "[Change something that doesn't work]\n", layout.Config{
		BoxW: 160, BoxH: 40, MaxWidth: 1000, FontSize: 13,
	})
	rs = rects(small)
	if want := 2*17 + 2*layout.BoxPad; rs[0].H != want {
		t.Errorf("box h = %d, want %d (grows for wrapped title)", rs[0].H, want)
	}
}
