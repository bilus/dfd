package layout_test

import (
	"strings"
	"testing"

	"golang.org/x/image/font"

	"github.com/bilus/dfd/layout"
	"github.com/bilus/dfd/parse"
	"github.com/bilus/dfd/theme"
)

func arrange(t *testing.T, src string, c layout.Config) *layout.Scene {
	t.Helper()
	d, err := parse.Parse(strings.NewReader(src), "test.dfd")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if c.BoxW == 0 {
		c.BoxW, c.BoxH, c.FontSize = 160, 60, 13
	}
	if c.Theme.Name == "" {
		c.Theme = defaultTheme(t, c.FontSize)
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
	want := layout.Text{X: 245, Y: 62, S: "go", Anchor: layout.Middle, Role: theme.Label}
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
			if l.Structural {
				thick = append(thick, l)
			} else if l.Head {
				arrows = append(arrows, l)
			}
		}
	}
	if len(thick) != 2 {
		t.Fatalf("got %d thick lines, want 2 (store glyph)", len(thick))
	}
	upper := layout.Line{X1: 45, Y1: 40, X2: 195, Y2: 40, Structural: true}
	lower := layout.Line{X1: 45, Y1: 76, X2: 195, Y2: 76, Structural: true}
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
	wantName := layout.Text{X: 120, Y: 62, S: "Database", Anchor: layout.Middle, Role: theme.StoreName}
	wantLabel := layout.Text{X: 128, Y: 113, S: "something", Anchor: layout.Start, Role: theme.Label}
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
		if l, ok := it.(layout.Line); ok && l.Head && !l.Structural {
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
		{X: 92, Y: 113, S: "input", Anchor: layout.End, Role: theme.Label},
		{X: 148, Y: 113, S: "record id", Anchor: layout.Start, Role: theme.Label},
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
	if !sceneTexts(s)[layout.Text{X: 128, Y: 113, S: "rows", Anchor: layout.Start, Role: theme.Label}] {
		t.Error("missing get label right of centered arrow")
	}
}

func TestMultipleStoresSideBySideWithWidening(t *testing.T) {
	s := arrange(t, "[Fan out]\n> a\n|Cache|\n> b\n|Queue with long name|\n", layout.Config{})
	var thick []layout.Line
	for _, it := range s.Items {
		if l, ok := it.(layout.Line); ok && l.Structural {
			thick = append(thick, l)
		}
	}
	if len(thick) != 4 {
		t.Fatalf("got %d thick lines, want 4 (two glyphs)", len(thick))
	}
	cacheW := thick[0].X2 - thick[0].X1
	queueW := thick[2].X2 - thick[2].X1
	if cacheW != layout.StoreW {
		t.Errorf("cache glyph width = %d, want %d", cacheW, layout.StoreW)
	}
	if queueW <= layout.StoreW {
		t.Errorf("queue glyph width = %d, want > %d (long name widens)", queueW, layout.StoreW)
	}
	if gap := thick[2].X1 - thick[0].X2; gap != layout.StoreGap {
		t.Errorf("gap between glyphs = %d, want %d", gap, layout.StoreGap)
	}
	var box layout.Rect
	for _, it := range s.Items {
		if r, ok := it.(layout.Rect); ok {
			box = r
		}
	}
	groupL, groupR := thick[0].X1, thick[2].X2
	if got, want := groupL+groupR, 2*(box.X+80); got != want {
		t.Errorf("group not centered on box: group mid*2 = %d, box mid*2 = %d", got, want)
	}
	if groupL != layout.Margin {
		t.Errorf("group left = %d, want %d (column starts at margin)", groupL, layout.Margin)
	}
	if want := 2*layout.Margin + (groupR - groupL); s.W != want {
		t.Errorf("scene w = %d, want %d (column width = store group)", s.W, want)
	}
	for _, a := range headArrows(s) {
		if a.X1 < box.X+20 || a.X1 > box.X+160-20 {
			t.Errorf("arrow x = %d escapes box span [%d,%d]", a.X1, box.X+20, box.X+140)
		}
	}
}

func rects(s *layout.Scene) []layout.Rect {
	var out []layout.Rect
	for _, it := range s.Items {
		if r, ok := it.(layout.Rect); ok {
			out = append(out, r)
		}
	}
	return out
}

func TestSnakeRowsAndTurns(t *testing.T) {
	s := arrange(t, "[One]\n[Two]\n[Three]\n[Four]\n[Five]\n", layout.Config{
		BoxW: 160, BoxH: 60, PerRow: 2, FontSize: 13,
	})
	rs := rects(s)
	if len(rs) != 5 {
		t.Fatalf("got %d rects, want 5", len(rs))
	}
	for i, wantY := range []int{40, 40, 190, 190, 340} {
		if rs[i].Y != wantY {
			t.Errorf("box %d y = %d, want %d", i, rs[i].Y, wantY)
		}
	}
	for i, wantX := range []int{40, 290, 290, 40, 40} {
		if rs[i].X != wantX {
			t.Errorf("box %d x = %d, want %d (snake mirror)", i, rs[i].X, wantX)
		}
	}
	var turns, horiz []layout.Line
	for _, a := range headArrows(s) {
		if a.X1 == a.X2 {
			turns = append(turns, a)
		} else {
			horiz = append(horiz, a)
		}
	}
	if len(turns) != 2 || len(horiz) != 2 {
		t.Fatalf("got %d turns / %d horizontal, want 2/2", len(turns), len(horiz))
	}
	if want := (layout.Line{X1: 370, Y1: 100, X2: 370, Y2: 187, Head: true}); turns[0] != want {
		t.Errorf("first turn = %+v, want %+v", turns[0], want)
	}
	if want := (layout.Line{X1: 120, Y1: 250, X2: 120, Y2: 337, Head: true}); turns[1] != want {
		t.Errorf("second turn = %+v, want %+v", turns[1], want)
	}
	if want := (layout.Line{X1: 290, Y1: 220, X2: 203, Y2: 220, Head: true}); horiz[1] != want {
		t.Errorf("row-1 arrow = %+v, want %+v (right to left)", horiz[1], want)
	}
	if s.W != 490 || s.H != 440 {
		t.Errorf("scene = %dx%d, want 490x440", s.W, s.H)
	}
}

func TestTurnArrowLabel(t *testing.T) {
	s := arrange(t, "[One]\n[Two]\n> down\n[Three]\n", layout.Config{
		BoxW: 160, BoxH: 60, PerRow: 2, FontSize: 13,
	})
	if !sceneTexts(s)[layout.Text{X: 378, Y: 150, S: "down", Anchor: layout.Start, Role: theme.Label}] {
		t.Error("turn label not right of the vertical arrow")
	}
}

func TestStoreSidesAcrossRows(t *testing.T) {
	src := `[A]
[B]
> x
|SB|
[C]
> y
|SC|
[D]
[E]
> z
|SE|
[F]
`
	s := arrange(t, src, layout.Config{BoxW: 160, BoxH: 60, PerRow: 2, FontSize: 13})
	rs := rects(s)
	if len(rs) != 6 {
		t.Fatalf("got %d rects, want 6", len(rs))
	}
	for i, wantY := range []int{140, 140, 290, 290, 480, 480} {
		if rs[i].Y != wantY {
			t.Errorf("box %d y = %d, want %d (lanes grow gaps)", i, rs[i].Y, wantY)
		}
	}
	var thick []layout.Line
	for _, it := range s.Items {
		if l, ok := it.(layout.Line); ok && l.Structural {
			thick = append(thick, l)
		}
	}
	if len(thick) != 6 {
		t.Fatalf("got %d thick lines, want 6 (three glyphs)", len(thick))
	}
	// SB above row 0: lines at 40 and 76 over box B (col 1).
	if thick[0].Y1 != 40 || thick[1].Y1 != 76 {
		t.Errorf("SB glyph lines at %d/%d, want 40/76 (above row 0)", thick[0].Y1, thick[1].Y1)
	}
	// SC below row 1 (C is row-first, its top is taken by the turn arrow):
	// upper line at boxBottom+StoreArrow = 350+64 = 414.
	if thick[2].Y1 != 414 || thick[3].Y1 != 450 {
		t.Errorf("SC glyph lines at %d/%d, want 414/450 (flipped below)", thick[2].Y1, thick[3].Y1)
	}
	// SE below row 2 (last row): upper line at 480+60+64 = 604.
	if thick[4].Y1 != 604 || thick[5].Y1 != 640 {
		t.Errorf("SE glyph lines at %d/%d, want 604/640 (below last row)", thick[4].Y1, thick[5].Y1)
	}
	if s.H != 680 {
		t.Errorf("scene h = %d, want 680 (bottom lane)", s.H)
	}
}

func TestPerRowOneStoreHasNoFreeSide(t *testing.T) {
	d, err := parse.Parse(strings.NewReader("[A]\n[B]\n> x\n|S|\n[C]\n"), "test.dfd")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	_, err = layout.Arrange(d, layout.Config{BoxW: 160, BoxH: 60, PerRow: 1, FontSize: 13, Theme: defaultTheme(t, 13)})
	if err == nil || !strings.Contains(err.Error(), "no free side") {
		t.Fatalf("err = %v, want no-free-side error", err)
	}
}

func TestFlowLabelWrapsAtWordBoundaries(t *testing.T) {
	s := arrange(t, "[Start container/process]\n> config, server node\n[Start workspace agent]\n", layout.Config{})
	var lines []layout.Text
	for _, it := range s.Items {
		if tx, ok := it.(layout.Text); ok && tx.X == 245 {
			lines = append(lines, tx)
		}
	}
	if len(lines) != 2 {
		t.Fatalf("got %d label lines at gap midpoint, want 2 (wrapped)", len(lines))
	}
	if got := lines[0].S + " " + lines[1].S; got != "config, server node" {
		t.Errorf("wrapped lines join to %q, want original label", got)
	}
	face := mustFace(t)
	for _, ln := range lines {
		if w := layout.TextWidth(ln.S, face); w > layout.HGap-2*layout.LabelPad {
			t.Errorf("label line %q is %dpx, exceeds %d", ln.S, w, layout.HGap-2*layout.LabelPad)
		}
	}
	if lines[0].Y != 62-lineH(t) || lines[1].Y != 62 {
		t.Errorf("label baselines = %d/%d, want %d/%d (stack grows upward)", lines[0].Y, lines[1].Y, 62-lineH(t), 62)
	}
	rs := rects(s)
	if gap := rs[1].X - (rs[0].X + 160); gap != layout.HGap {
		t.Errorf("gap = %d, want unchanged %d (all words fit)", gap, layout.HGap)
	}
}

func TestFlowLabelLongWordWidensGap(t *testing.T) {
	s := arrange(t, "[A]\n> internationalization\n[B]\n", layout.Config{})
	face := mustFace(t)
	wantGap := layout.TextWidth("internationalization", face) + 2*layout.LabelPad
	if wantGap <= layout.HGap {
		t.Fatalf("test word too narrow (%d) to force widening", wantGap)
	}
	rs := rects(s)
	if gap := rs[1].X - (rs[0].X + 160); gap != wantGap {
		t.Errorf("gap = %d, want %d (widened to longest word)", gap, wantGap)
	}
	found := false
	for _, it := range s.Items {
		if tx, ok := it.(layout.Text); ok && tx.S == "internationalization" {
			found = true
			if tx.Y != 62 {
				t.Errorf("single-line label y = %d, want 62", tx.Y)
			}
		}
	}
	if !found {
		t.Error("label missing or unexpectedly wrapped")
	}
	if want := 2*layout.Margin + 2*160 + wantGap; s.W != want {
		t.Errorf("scene w = %d, want %d", s.W, want)
	}
}

func TestExplicitTitleBreaks(t *testing.T) {
	s := arrange(t, "[This is a box\n line 2]\n", layout.Config{})
	texts := sceneTexts(s)
	for _, want := range []layout.Text{
		{X: 120, Y: 67, S: "This is a box", Anchor: layout.Middle, Role: theme.Title},
		{X: 120, Y: 84, S: "line 2", Anchor: layout.Middle, Role: theme.Title},
	} {
		if !texts[want] {
			t.Errorf("missing title line %+v", want)
		}
	}
}

func TestExplicitFlowLabelBreaks(t *testing.T) {
	s := arrange(t, "[A]\n> line 1\n  line 2\n[B]\n", layout.Config{})
	texts := sceneTexts(s)
	for _, want := range []layout.Text{
		{X: 245, Y: 45, S: "line 1", Anchor: layout.Middle, Role: theme.Label},
		{X: 245, Y: 62, S: "line 2", Anchor: layout.Middle, Role: theme.Label},
	} {
		if !texts[want] {
			t.Errorf("missing flow label line %+v", want)
		}
	}
}

func TestExplicitStoreLabelBreaks(t *testing.T) {
	s := arrange(t, "[A]\n> aaa\n  bbb\n< ccc\n  ddd\n|S|\n", layout.Config{})
	texts := sceneTexts(s)
	for _, want := range []layout.Text{
		{X: 92, Y: 105, S: "aaa", Anchor: layout.End, Role: theme.Label},
		{X: 92, Y: 122, S: "bbb", Anchor: layout.End, Role: theme.Label},
		{X: 148, Y: 105, S: "ccc", Anchor: layout.Start, Role: theme.Label},
		{X: 148, Y: 122, S: "ddd", Anchor: layout.Start, Role: theme.Label},
	} {
		if !texts[want] {
			t.Errorf("missing store label line %+v", want)
		}
	}
}

func TestExplicitTurnLabelBreaks(t *testing.T) {
	s := arrange(t, "[One]\n[Two]\n> down\n  more\n[Three]\n", layout.Config{
		BoxW: 160, BoxH: 60, PerRow: 2, FontSize: 13,
	})
	texts := sceneTexts(s)
	for _, want := range []layout.Text{
		{X: 378, Y: 142, S: "down", Anchor: layout.Start, Role: theme.Label},
		{X: 378, Y: 159, S: "more", Anchor: layout.Start, Role: theme.Label},
	} {
		if !texts[want] {
			t.Errorf("missing turn label line %+v", want)
		}
	}
}

func defaultTheme(t *testing.T, base int) theme.Theme {
	t.Helper()
	th, err := theme.Lookup("default", base)
	if err != nil {
		t.Fatalf("theme: %v", err)
	}
	return th
}

func mustFace(t *testing.T) font.Face {
	t.Helper()
	return defaultTheme(t, 13).Style(theme.Title).Face
}

// lineH is the default theme's baseline-to-baseline spacing at 13px.
func lineH(t *testing.T) int {
	t.Helper()
	return defaultTheme(t, 13).Style(theme.Label).LineH()
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
	if !ok || tx != (layout.Text{X: 120, Y: 75, S: "Hello", Anchor: layout.Middle, Role: theme.Title}) {
		t.Errorf("text = %+v, want centered baseline at (120,75)", s.Items[1])
	}
}

func plexTheme(t *testing.T, base int) theme.Theme {
	t.Helper()
	th, err := theme.Lookup("plex", base)
	if err != nil {
		t.Fatalf("theme: %v", err)
	}
	return th
}

func TestPlexCentresFlowLabelsOnTheArrow(t *testing.T) {
	th := plexTheme(t, 13)
	s := arrange(t, "[A]\n> go\n[B]\n", layout.Config{
		BoxW: 160, BoxH: 60, FontSize: 13, Theme: th,
	})
	var got *layout.Text
	for _, it := range s.Items {
		if tx, ok := it.(layout.Text); ok && tx.S == "go" {
			got = &tx
		}
	}
	if got == nil {
		t.Fatal("flow label not in scene")
	}
	var arrow layout.Line
	for _, l := range headArrows(s) {
		arrow = l
	}
	// The chip masks the line, so the text sits on it rather than above.
	if want := arrow.Y1 + int(th.Style(theme.Label).Size)/3; got.Y != want {
		t.Errorf("label baseline = %d, want %d (centred on the arrow at y=%d)", got.Y, want, arrow.Y1)
	}
	if got.Anchor != layout.Middle {
		t.Errorf("anchor = %v, want Middle", got.Anchor)
	}
}

func TestPlexCentresTurnLabelsOnTheArrow(t *testing.T) {
	th := plexTheme(t, 13)
	s := arrange(t, "[One]\n[Two]\n> down\n[Three]\n", layout.Config{
		BoxW: 160, BoxH: 60, PerRow: 2, FontSize: 13, Theme: th,
	})
	var turn layout.Line
	for _, l := range headArrows(s) {
		if l.X1 == l.X2 {
			turn = l
		}
	}
	if turn.X1 == 0 {
		t.Fatal("no turn arrow")
	}
	for _, it := range s.Items {
		if tx, ok := it.(layout.Text); ok && tx.S == "down" {
			if tx.X != turn.X1 || tx.Anchor != layout.Middle {
				t.Errorf("turn label at x=%d anchor=%v, want x=%d centred", tx.X, tx.Anchor, turn.X1)
			}
			return
		}
	}
	t.Fatal("turn label not in scene")
}

func TestDefaultKeepsLabelsBesideTheArrow(t *testing.T) {
	s := arrange(t, "[A]\n> go\n[B]\n", layout.Config{})
	for _, it := range s.Items {
		if tx, ok := it.(layout.Text); ok && tx.S == "go" {
			if tx.Y != 62 {
				t.Errorf("default label baseline = %d, want 62 (above the line, unchanged)", tx.Y)
			}
			return
		}
	}
	t.Fatal("flow label not in scene")
}

func labelExtent(tx layout.Text, th theme.Theme) (lo, hi int) {
	st := th.Style(tx.Role)
	w := layout.TextWidth(tx.S, st.Face)
	switch tx.Anchor {
	case layout.Middle:
		lo, hi = tx.X-w/2, tx.X+w/2
	case layout.End:
		lo, hi = tx.X-w, tx.X
	default:
		lo, hi = tx.X, tx.X+w
	}
	if th.LabelChip && tx.Role == theme.Label {
		lo, hi = lo-theme.ChipPadX, hi+theme.ChipPadX
	}
	return lo, hi
}

func assertNothingClipped(t *testing.T, s *layout.Scene, th theme.Theme) {
	t.Helper()
	for _, it := range s.Items {
		tx, ok := it.(layout.Text)
		if !ok {
			continue
		}
		lo, hi := labelExtent(tx, th)
		if lo < layout.Margin {
			t.Errorf("%q starts at x=%d, inside the left margin of %d", tx.S, lo, layout.Margin)
		}
		if hi > s.W-layout.Margin {
			t.Errorf("%q ends at x=%d, past the %d canvas less its %d margin", tx.S, hi, s.W, layout.Margin)
		}
	}
}

func TestStoreLabelInLastColumnWidensTheCanvas(t *testing.T) {
	th := plexTheme(t, 13)
	s := arrange(t, "[Start]\n> go\n[Register live page]\n    > page id, mountFn, tree\n    |Registry|\n",
		layout.Config{BoxW: 160, BoxH: 60, FontSize: 13, Theme: th})
	assertNothingClipped(t, s, th)
}

func TestStoreLabelOverhangingTheLeftShiftsTheDiagram(t *testing.T) {
	th := plexTheme(t, 13)
	s := arrange(t, "[A]\n> a considerably long put label\n< out\n|S|\n",
		layout.Config{BoxW: 160, BoxH: 60, FontSize: 13, Theme: th})
	assertNothingClipped(t, s, th)
	if rs := rects(s); rs[0].X <= layout.Margin {
		t.Errorf("box x = %d; a label overhanging the left edge must push the diagram right", rs[0].X)
	}
}

func TestNothingIsClippedAcrossThemes(t *testing.T) {
	srcs := []string{
		"[Start]\n> go\n[Register live page]\n    > page id, mountFn, tree\n    |Registry|\n",
		"[A]\n> a considerably long put label\n< out\n|S|\n",
		"[One]\n[Two]\n> down\n[Three]\n    < a long read label here\n    |Disk|\n",
		"[Only]\n",
	}
	for _, name := range theme.Names() {
		for i, src := range srcs {
			th, err := theme.Lookup(name, 13)
			if err != nil {
				t.Fatalf("theme: %v", err)
			}
			s := arrange(t, src, layout.Config{BoxW: 160, BoxH: 60, FontSize: 13, Theme: th})
			t.Run(name+"/"+string(rune('a'+i)), func(t *testing.T) {
				assertNothingClipped(t, s, th)
			})
		}
	}
}

func TestMarginsStaySymmetricWhenTheCanvasGrows(t *testing.T) {
	th := plexTheme(t, 13)
	s := arrange(t, "[Start]\n> go\n[Register live page]\n    > page id, mountFn, tree\n    |Registry|\n",
		layout.Config{BoxW: 160, BoxH: 60, FontSize: 13, Theme: th})
	lo, hi := s.W, 0
	for _, it := range s.Items {
		if tx, ok := it.(layout.Text); ok {
			l, h := labelExtent(tx, th)
			lo, hi = min(lo, l), max(hi, h)
		}
		if r, ok := it.(layout.Rect); ok {
			lo, hi = min(lo, r.X), max(hi, r.X+r.W)
		}
	}
	if left, right := lo, s.W-hi; left != right {
		t.Errorf("margins are %d left and %d right; growing the canvas must keep them equal", left, right)
	}
}

func boxesInFirstRow(s *layout.Scene) int {
	rs := rects(s)
	if len(rs) == 0 {
		return 0
	}
	n := 0
	for _, r := range rs {
		if r.Y == rs[0].Y {
			n++
		}
	}
	return n
}

func TestDefaultIsFourColumnsWhateverTheTheme(t *testing.T) {
	// A label wide enough to stretch plex's column gaps: the column
	// count must not depend on how wide a theme draws things.
	const src = "[One]\n> page id, mountFn, tree\n[Two]\n[Three]\n[Four]\n[Five]\n[Six]\n"
	for _, name := range theme.Names() {
		th, err := theme.Lookup(name, 13)
		if err != nil {
			t.Fatalf("theme: %v", err)
		}
		s := arrange(t, src, layout.Config{BoxW: 160, BoxH: 60, FontSize: 13, Theme: th})
		if got := boxesInFirstRow(s); got != layout.DefaultPerRow {
			t.Errorf("theme %q put %d boxes in the first row, want %d", name, got, layout.DefaultPerRow)
		}
	}
}

func TestPerRowOverridesTheDefault(t *testing.T) {
	s := arrange(t, "[One]\n[Two]\n[Three]\n[Four]\n[Five]\n", layout.Config{
		BoxW: 160, BoxH: 60, FontSize: 13, PerRow: 3,
	})
	if got := boxesInFirstRow(s); got != 3 {
		t.Errorf("got %d boxes in the first row, want 3", got)
	}
}

// gapOf is the uniform space between adjacent boxes in a row.
func gapOf(t *testing.T, s *layout.Scene) int {
	t.Helper()
	rs := rects(s)
	if len(rs) < 2 {
		t.Fatal("need two boxes to measure a gap")
	}
	gap := rs[1].X - (rs[0].X + rs[0].W)
	for i := 2; i < len(rs); i++ {
		if rs[i].Y != rs[i-1].Y {
			continue // a row turn, not a gap
		}
		if g := rs[i].X - (rs[i-1].X + rs[i-1].W); g != gap {
			t.Errorf("gaps differ: %d then %d; every column gap must be the same width", gap, g)
		}
	}
	return gap
}

func labelLines(s *layout.Scene) []string {
	var out []string
	for _, it := range s.Items {
		if tx, ok := it.(layout.Text); ok && tx.Role == theme.Label {
			out = append(out, tx.S)
		}
	}
	return out
}

// The agreed rule, for every theme: flow labels wrap at word
// boundaries; the column gap stays at HGap unless a single word cannot
// be wrapped, and then every gap grows to exactly what that word needs.
func TestFlowLabelGapIsHGapUnlessAWordCannotFit(t *testing.T) {
	labels := []string{
		"go",
		"config, server node",
		"page id, mountFn, tree",
		"internationalization",
		"a b c d e f g h i j k l m n o p",
	}
	for _, name := range theme.Names() {
		th, err := theme.Lookup(name, 13)
		if err != nil {
			t.Fatalf("theme: %v", err)
		}
		inset := layout.LabelInset(th)
		face := th.Style(theme.Label).Face
		for _, label := range labels {
			t.Run(name+"/"+label, func(t *testing.T) {
				s := arrange(t, "[A]\n> "+label+"\n[B]\n", layout.Config{
					BoxW: 160, BoxH: 60, FontSize: 13, Theme: th,
				})
				widest := 0
				for _, w := range strings.Fields(label) {
					if x := layout.TextWidth(w, face); x > widest {
						widest = x
					}
				}
				want := layout.HGap
				if need := widest + inset; need > want {
					want = need
				}
				gap := gapOf(t, s)
				if gap != want {
					t.Errorf("gap = %d, want %d (widest word %d + inset %d, floor %d)",
						gap, want, widest, inset, layout.HGap)
				}
				for _, ln := range labelLines(s) {
					if w := layout.TextWidth(ln, face); w > gap-inset {
						t.Errorf("label line %q is %dpx, over the %dpx a %dpx gap leaves", ln, w, gap-inset, gap)
					}
				}
			})
		}
	}
}

func TestFlowLabelsAreSplitAtWordBoundaries(t *testing.T) {
	for _, name := range theme.Names() {
		th, err := theme.Lookup(name, 13)
		if err != nil {
			t.Fatalf("theme: %v", err)
		}
		s := arrange(t, "[A]\n> page id, mountFn, tree\n[B]\n", layout.Config{
			BoxW: 160, BoxH: 60, FontSize: 13, Theme: th,
		})
		lines := labelLines(s)
		if len(lines) < 2 {
			t.Errorf("theme %q left the label on %d line(s); it must wrap rather than widen the gap", name, len(lines))
		}
		if got := strings.Join(lines, " "); got != "page id, mountFn, tree" {
			t.Errorf("theme %q wrapped to %q, losing the original words", name, got)
		}
	}
}

func TestOneLongWordWidensEveryGapEqually(t *testing.T) {
	for _, name := range theme.Names() {
		th, err := theme.Lookup(name, 13)
		if err != nil {
			t.Fatalf("theme: %v", err)
		}
		s := arrange(t, "[A]\n> internationalization\n[B]\n> x\n[C]\n", layout.Config{
			BoxW: 160, BoxH: 60, FontSize: 13, Theme: th,
		})
		gap := gapOf(t, s) // gapOf itself fails if the gaps differ
		want := layout.TextWidth("internationalization", th.Style(theme.Label).Face) + layout.LabelInset(th)
		if gap != want {
			t.Errorf("theme %q gap = %d, want %d", name, gap, want)
		}
	}
}

func numberTexts(s *layout.Scene) []string {
	var out []string
	for _, it := range s.Items {
		if tx, ok := it.(layout.Text); ok && tx.Role == theme.Number {
			out = append(out, tx.S)
		}
	}
	return out
}

func storeNames(s *layout.Scene) []string {
	var out []string
	for _, it := range s.Items {
		if tx, ok := it.(layout.Text); ok && tx.Role == theme.StoreName {
			out = append(out, tx.S)
		}
	}
	return out
}

const numberingSrc = "{Client}\n> a\n[One]\n    > x\n    |Users|\n> b\n[Two]\n    > y\n    |Pages|\n> c\n{CDN}\n"

func TestNumberingCountsProcessesAndStoresOnly(t *testing.T) {
	for _, name := range theme.Names() {
		th, err := theme.Lookup(name, 13)
		if err != nil {
			t.Fatalf("theme: %v", err)
		}
		s := arrange(t, numberingSrc, layout.Config{
			BoxW: 160, BoxH: 60, FontSize: 13, Theme: th, Number: true,
		})
		if got := numberTexts(s); len(got) != 2 || got[0] != "1" || got[1] != "2" {
			t.Errorf("theme %q numbers = %q, want [1 2]; entities are not numbered", name, got)
		}
		want := []string{"D1 Users", "D2 Pages"}
		got := storeNames(s)
		if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
			t.Errorf("theme %q store names = %q, want %q", name, got, want)
		}
	}
}

func TestNumberingIsOffUnlessAsked(t *testing.T) {
	s := arrange(t, numberingSrc, layout.Config{BoxW: 160, BoxH: 60, FontSize: 13})
	if got := numberTexts(s); len(got) != 0 {
		t.Errorf("got numbers %q with numbering off", got)
	}
	if got := storeNames(s); got[0] != "Users" {
		t.Errorf("store name = %q, want the bare name", got[0])
	}
}

func TestNumberPrefixMakesLevelledNumbers(t *testing.T) {
	s := arrange(t, numberingSrc, layout.Config{
		BoxW: 160, BoxH: 60, FontSize: 13, Number: true, NumberPrefix: "2.",
	})
	if got := numberTexts(s); len(got) != 2 || got[0] != "2.1" || got[1] != "2.2" {
		t.Errorf("numbers = %q, want [2.1 2.2]", got)
	}
	// stores stay D1, D2 whatever the process level
	if got := storeNames(s); got[0] != "D1 Users" {
		t.Errorf("store name = %q, want D1 Users", got[0])
	}
}

func TestNumberSitsInsideTheBoxAboveTheTitle(t *testing.T) {
	th := plexTheme(t, 13)
	s := arrange(t, "[Only]\n", layout.Config{BoxW: 160, BoxH: 60, FontSize: 13, Theme: th, Number: true})
	var num, title layout.Text
	for _, it := range s.Items {
		if tx, ok := it.(layout.Text); ok {
			switch tx.Role {
			case theme.Number:
				num = tx
			case theme.Title:
				title = tx
			}
		}
	}
	box := rects(s)[0]
	if num.X != box.X+layout.BoxPad || num.Anchor != layout.Start {
		t.Errorf("number at x=%d anchor=%v, want %d and Start", num.X, num.Anchor, box.X+layout.BoxPad)
	}
	if num.Y >= title.Y {
		t.Errorf("number baseline %d is not above the title baseline %d", num.Y, title.Y)
	}
	if num.Y <= box.Y {
		t.Errorf("number baseline %d is outside the box top %d", num.Y, box.Y)
	}
}

func TestNumberBandKeepsAWrappedTitleClear(t *testing.T) {
	th := defaultTheme(t, 13)
	src := "[A title long enough that it has to wrap onto several lines inside the box]\n"
	plain := arrange(t, src, layout.Config{BoxW: 160, BoxH: 60, FontSize: 13, Theme: th})
	numbered := arrange(t, src, layout.Config{BoxW: 160, BoxH: 60, FontSize: 13, Theme: th, Number: true})
	if numbered.Items[0].(layout.Rect).H <= plain.Items[0].(layout.Rect).H {
		t.Error("a wrapped title must grow the box to make room for the number band")
	}
	var num, firstTitle layout.Text
	for _, it := range numbered.Items {
		if tx, ok := it.(layout.Text); ok {
			if tx.Role == theme.Number {
				num = tx
			} else if tx.Role == theme.Title && firstTitle.S == "" {
				firstTitle = tx
			}
		}
	}
	if firstTitle.Y <= num.Y {
		t.Errorf("first title line at %d collides with the number band ending at %d", firstTitle.Y, num.Y)
	}
}

func TestSameLabelIsTheSameNodeForNumbering(t *testing.T) {
	// Registration appears twice and Confirm twice; each keeps the
	// number it was given the first time.
	src := "[Register]\n    > row\n    |Registration|\n> id\n[Confirm]\n    < row\n    |Registration|\n" +
		"> ok\n[Notify]\n    > event\n    |Events|\n> again\n[Confirm]\n"
	s := arrange(t, src, layout.Config{BoxW: 160, BoxH: 60, FontSize: 13, PerRow: 4, Number: true})
	want := []string{"1", "2", "3", "2"}
	if got := numberTexts(s); len(got) != 4 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] || got[3] != want[3] {
		t.Errorf("process numbers = %q, want %q", numberTexts(s), want)
	}
	stores := storeNames(s)
	wantStores := []string{"D1 Registration", "D1 Registration", "D2 Events"}
	if len(stores) != 3 {
		t.Fatalf("store names = %q", stores)
	}
	for i := range wantStores {
		if stores[i] != wantStores[i] {
			t.Errorf("store names = %q, want %q", stores, wantStores)
			break
		}
	}
}

func TestDistinctLabelsStillCountUp(t *testing.T) {
	s := arrange(t, "[A]\n> x\n[B]\n> y\n[C]\n", layout.Config{
		BoxW: 160, BoxH: 60, FontSize: 13, Number: true,
	})
	if got := numberTexts(s); len(got) != 3 || got[2] != "3" {
		t.Errorf("numbers = %q, want [1 2 3]", got)
	}
}

func TestEntitiesRepeatWithoutTakingNumbers(t *testing.T) {
	s := arrange(t, "{Client}\n> a\n[Handle]\n> b\n{Client}\n", layout.Config{
		BoxW: 160, BoxH: 60, FontSize: 13, Number: true,
	})
	if got := numberTexts(s); len(got) != 1 || got[0] != "1" {
		t.Errorf("numbers = %q, want just [1] for the one process", got)
	}
}

// titleMid is the middle of the title block.
func titleMid(t *testing.T, s *layout.Scene) int {
	t.Helper()
	first, last := 1<<30, 0
	for _, it := range s.Items {
		if tx, ok := it.(layout.Text); ok && tx.Role == theme.Title {
			if tx.Y < first {
				first = tx.Y
			}
			if tx.Y > last {
				last = tx.Y
			}
		}
	}
	if last == 0 {
		t.Fatal("no title in the scene")
	}
	return (first + last) / 2
}

// Option C: the number gets its own compartment at the top of the box,
// divided by a rule, and the title centres in what is left. The number
// and the title cannot overlap however the title wraps.
func TestNumberBandSeparatesTheNumberFromTheTitle(t *testing.T) {
	const wide = "[Wrap component in container]\n" // wraps to the full box width
	for _, name := range theme.Names() {
		th, err := theme.Lookup(name, 13)
		if err != nil {
			t.Fatalf("theme: %v", err)
		}
		s := arrange(t, wide, layout.Config{
			BoxW: 160, BoxH: 60, FontSize: 13, Theme: th, Number: true, NumberPrefix: "1.",
		})
		box := rects(s)[0]
		if box.Band <= 0 {
			t.Fatalf("%s: numbered box has no band", name)
		}
		rule := box.Y + box.Band
		var num layout.Text
		firstTitle := 1 << 30
		for _, it := range s.Items {
			tx, ok := it.(layout.Text)
			if !ok {
				continue
			}
			switch tx.Role {
			case theme.Number:
				num = tx
			case theme.Title:
				if tx.Y < firstTitle {
					firstTitle = tx.Y
				}
			}
		}
		if num.S == "" {
			t.Fatalf("%s: no number drawn", name)
		}
		if num.Y <= box.Y || num.Y > rule {
			t.Errorf("%s: number baseline %d is not inside the band %d..%d", name, num.Y, box.Y, rule)
		}
		// The title's own line box has to start below the rule.
		if top := firstTitle - th.Style(theme.Title).LineH(); top < rule {
			t.Errorf("%s: title starts at %d, above the rule at %d; it would run through the number",
				name, top, rule)
		}
	}
}

func TestNoBandWhenNumberingIsOff(t *testing.T) {
	s := arrange(t, "[Wrap component in container]\n", layout.Config{})
	if box := rects(s)[0]; box.Band != 0 {
		t.Errorf("band = %d with numbering off, want none", box.Band)
	}
}

// Within its compartment the title is centred exactly as it is centred
// in a plain box, so numbering does not look like a nudge.
func TestTitleCentresInItsCompartment(t *testing.T) {
	for _, name := range theme.Names() {
		th, err := theme.Lookup(name, 13)
		if err != nil {
			t.Fatalf("theme: %v", err)
		}
		for _, src := range []string{"[Alpha]\n", "[Wrap component in container]\n"} {
			plain := arrange(t, src, layout.Config{BoxW: 160, BoxH: 60, FontSize: 13, Theme: th})
			numbered := arrange(t, src, layout.Config{BoxW: 160, BoxH: 60, FontSize: 13, Theme: th, Number: true})
			pb, nb := rects(plain)[0], rects(numbered)[0]
			want := titleMid(t, plain) - (pb.Y + pb.H/2)
			got := titleMid(t, numbered) - (nb.Y + nb.Band + (nb.H-nb.Band)/2)
			if got != want {
				t.Errorf("%s %q: title sits %d from its compartment centre, %d from a plain box centre",
					name, src, got, want)
			}
		}
	}
}
