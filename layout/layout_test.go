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
		c.BoxW, c.BoxH, c.MaxWidth, c.FontSize = 160, 60, 1000, 13
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
		BoxW: 160, BoxH: 60, MaxWidth: 700, FontSize: 13,
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
		BoxW: 160, BoxH: 60, MaxWidth: 1000, PerRow: 2, FontSize: 13,
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
	s := arrange(t, src, layout.Config{BoxW: 160, BoxH: 60, MaxWidth: 1000, PerRow: 2, FontSize: 13})
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
		BoxW: 160, BoxH: 60, MaxWidth: 1000, PerRow: 2, FontSize: 13,
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
		BoxW: 160, BoxH: 60, MaxWidth: 1000, FontSize: 13, Theme: th,
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
		BoxW: 160, BoxH: 60, MaxWidth: 1000, PerRow: 2, FontSize: 13, Theme: th,
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

func TestPlexGapLeavesArrowVisibleAroundTheChip(t *testing.T) {
	th := plexTheme(t, 13)
	s := arrange(t, "[A]\n> record id\n[B]\n", layout.Config{
		BoxW: 160, BoxH: 60, MaxWidth: 1000, FontSize: 13, Theme: th,
	})
	rs := rects(s)
	gap := rs[1].X - (rs[0].X + 160)
	chip := layout.TextWidth("record id", th.Style(theme.Label).Face) + 2*theme.ChipPadX
	if stub := (gap - chip) / 2; stub < layout.LabelStub {
		t.Errorf("only %dpx of arrow shows either side of the %dpx chip in a %dpx gap; want >= %d",
			stub, chip, gap, layout.LabelStub)
	}
}

func TestPlexKeepsALabelOnOneLine(t *testing.T) {
	th := plexTheme(t, 13)
	s := arrange(t, "[A]\n> config, server node\n[B]\n", layout.Config{
		BoxW: 160, BoxH: 60, MaxWidth: 1000, FontSize: 13, Theme: th,
	})
	n := 0
	for _, it := range s.Items {
		if tx, ok := it.(layout.Text); ok && tx.Role == theme.Label {
			n++
		}
	}
	if n != 1 {
		t.Errorf("label split into %d lines; on-line labels widen the gap instead of wrapping", n)
	}
}

func TestPlexHonoursExplicitLabelBreaks(t *testing.T) {
	th := plexTheme(t, 13)
	s := arrange(t, "[A]\n> one\n  two\n[B]\n", layout.Config{
		BoxW: 160, BoxH: 60, MaxWidth: 1000, FontSize: 13, Theme: th,
	})
	var got []string
	for _, it := range s.Items {
		if tx, ok := it.(layout.Text); ok && tx.Role == theme.Label {
			got = append(got, tx.S)
		}
	}
	if len(got) != 2 || got[0] != "one" || got[1] != "two" {
		t.Errorf("label lines = %q, want [one two]", got)
	}
}
