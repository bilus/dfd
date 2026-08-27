// Package layout turns an ast.Diagram into a Scene: a flat display list
// of primitives with final pixel coordinates.
package layout

import (
	"fmt"
	"strings"

	"golang.org/x/image/font"

	"github.com/bilus/dfd/ast"
	"github.com/bilus/dfd/theme"
)

// Geometry constants; normative values from the design spec.
const (
	HGap       = 90
	VGap       = 90 // base vertical gap between rows
	Margin     = 40
	Inset      = 3  // gap between an arrow tip and the edge it points at
	LabelGap   = 8  // distance between an arrow and its label
	LabelPad   = 8  // clearance between a flow label and the boxes beside it
	LabelStub  = 20 // arrow left visible either side of an on-line label chip
	LanePad    = 30 // clearance between a store lane and the next row
	BoxPad     = 12 // padding between box border and title text
	StoreW     = 150
	StoreH     = 36 // distance between the two datastore glyph lines
	StoreArrow = 64 // length of a box<->datastore arrow
	StoreGap   = 20 // between side-by-side datastore glyphs
)

type Config struct {
	BoxW, BoxH int
	MaxWidth   int
	PerRow     int // 0 = derive from MaxWidth
	FontSize   int
	Theme      theme.Theme // paints the scene and measures its text; required
}

// TextWidth is the advance width of s in whole pixels.
func TextWidth(s string, face font.Face) int {
	return font.MeasureString(face, s).Ceil()
}

// storeWidth is a datastore glyph's width: wide enough for its name.
func storeWidth(l ast.StoreLink, face font.Face) int {
	if w := TextWidth(l.Name, face) + 20; w > StoreW {
		return w
	}
	return StoreW
}

// Arrange lays the diagram out on a grid and returns the display list.
func Arrange(d *ast.Diagram, c Config) (*Scene, error) {
	titleSt := c.Theme.Style(theme.Title)
	labelSt := c.Theme.Style(theme.Label)
	storeSt := c.Theme.Style(theme.StoreName)
	if titleSt.Face == nil || labelSt.Face == nil || storeSt.Face == nil {
		return nil, fmt.Errorf("layout: Config.Theme must supply a face for every role")
	}
	s := &Scene{FontSize: c.FontSize}
	n := len(d.Steps)

	// Wrap titles up front; the tallest one sets the uniform box height.
	titles := make([][]string, n)
	boxH := c.BoxH
	for i, st := range d.Steps {
		titles[i] = WrapSegments(st.Title, c.BoxW-2*BoxPad, titleSt.Face)
		if h := len(titles[i])*titleSt.LineH() + 2*BoxPad; h > boxH {
			boxH = h
		}
	}
	c.BoxH = boxH // c is a copy; the grown height flows to all consumers

	// Uniform column width: boxes, widened by the largest store group.
	colW := c.BoxW
	for _, st := range d.Steps {
		if g := groupWidth(st.Stores, storeSt.Face); g > colW {
			colW = g
		}
	}

	// Column gap: wide enough that no flow-label word overhangs a box.
	// Conservatively sized from every label (labels that end up on turn
	// arrows don't occupy a gap, but row assignment depends on the gap,
	// so the bound must be computed first).
	gapW := HGap
	for _, st := range d.Steps {
		if c.Theme.LabelOnLine {
			for _, seg := range strings.Split(st.In, "\n") {
				if need := TextWidth(seg, labelSt.Face) + 2*theme.ChipPadX + 2*LabelStub; need > gapW {
					gapW = need
				}
			}
			continue
		}
		for _, w := range strings.Fields(st.In) {
			if need := TextWidth(w, labelSt.Face) + 2*LabelPad; need > gapW {
				gapW = need
			}
		}
	}

	perRow := c.PerRow
	if perRow <= 0 {
		perRow = 1
		for 2*Margin+(perRow+1)*colW+perRow*gapW <= c.MaxWidth {
			perRow++
		}
	}
	nRows := (n + perRow - 1) / perRow

	// Columns: odd rows mirror so each row starts under the previous
	// row's last box.
	col := make([]int, n)
	maxCols := 0
	for i := range d.Steps {
		r, k := i/perRow, i%perRow
		if r%2 == 0 {
			col[i] = k
		} else {
			col[i] = perRow - 1 - k
		}
		if col[i]+1 > maxCols {
			maxCols = col[i] + 1
		}
	}

	// Store sides: +1 above, -1 below, 0 no stores. Preferred side is
	// above, except the last row of a multi-row diagram (below); flip
	// when a turn arrow occupies the preferred anchor.
	side := make([]int, n)
	for i, st := range d.Steps {
		if len(st.Stores) == 0 {
			continue
		}
		r := i / perRow
		pref := 1
		if r == nRows-1 && nRows > 1 {
			pref = -1
		}
		topBusy := r > 0 && i == r*perRow
		bottomBusy := i == (r+1)*perRow-1 && i != n-1
		if pref == 1 && topBusy {
			pref = -1
		} else if pref == -1 && bottomBusy {
			pref = 1
		}
		if (pref == 1 && topBusy) || (pref == -1 && bottomBusy) {
			return nil, fmt.Errorf("layout: step %q has no free side for its datastores; increase --max-width or --per-row", st.Title)
		}
		side[i] = pref
	}

	// Row positions: gaps grow to hold store lanes.
	lane := StoreArrow + StoreH
	above := make([]bool, nRows)
	below := make([]bool, nRows)
	for i := range d.Steps {
		r := i / perRow
		switch side[i] {
		case 1:
			above[r] = true
		case -1:
			below[r] = true
		}
	}
	rowY := make([]int, nRows)
	y := Margin
	if above[0] {
		y += lane
	}
	for r := 0; r < nRows; r++ {
		rowY[r] = y
		y += c.BoxH
		if r < nRows-1 {
			need := 0
			if below[r] {
				need += lane
			}
			if above[r+1] {
				need += lane
			}
			gap := VGap
			if need > 0 && need+LanePad > gap {
				gap = need + LanePad
			}
			y += gap
		}
	}
	bottom := Margin
	if below[nRows-1] {
		bottom += lane
	}

	boxX := func(i int) int { return Margin + col[i]*(colW+gapW) + (colW-c.BoxW)/2 }

	for i, st := range d.Steps {
		r := i / perRow
		x, by := boxX(i), rowY[r]
		cy := by + c.BoxH/2
		s.Items = append(s.Items, Rect{X: x, Y: by, W: c.BoxW, H: c.BoxH})
		first := cy + 5 - (len(titles[i])-1)*titleSt.LineH()/2
		for j, ln := range titles[i] {
			s.Items = append(s.Items, Text{X: x + c.BoxW/2, Y: first + j*titleSt.LineH(), S: ln, Anchor: Middle, Role: theme.Title})
		}
		if i > 0 {
			if pr := (i - 1) / perRow; pr == r {
				fromX := boxX(i - 1)
				var x1, x2 int
				if x > fromX { // left-to-right row
					x1, x2 = fromX+c.BoxW, x-Inset
				} else { // right-to-left row
					x1, x2 = fromX, x+c.BoxW+Inset
				}
				s.Items = append(s.Items, Line{X1: x1, Y1: cy, X2: x2, Y2: cy, Head: true})
				if st.In != "" {
					lines := WrapSegments(st.In, gapW-2*LabelPad, labelSt.Face)
					if c.Theme.LabelOnLine {
						lines = strings.Split(st.In, "\n")
					}
					mid := (fromX + x + c.BoxW) / 2
					for j, ln := range lines {
						var y int
						if c.Theme.LabelOnLine {
							y = baselineOn(cy, labelSt) + (j-(len(lines)-1)/2)*labelSt.LineH()
						} else {
							y = cy - LabelGap - (len(lines)-1-j)*labelSt.LineH()
						}
						s.Items = append(s.Items, Text{X: mid, Y: y, S: ln, Anchor: Middle, Role: theme.Label})
					}
				}
			} else { // snake turn: vertical arrow into this row's first box
				tx := x + c.BoxW/2
				fromY := rowY[pr] + c.BoxH
				s.Items = append(s.Items, Line{X1: tx, Y1: fromY, X2: tx, Y2: by - Inset, Head: true})
				if st.In != "" {
					mid := (fromY + by) / 2
					if c.Theme.LabelOnLine {
						emitLabelLines(s, st.In, tx, baselineOn(mid, labelSt), Middle, labelSt)
					} else {
						emitLabelLines(s, st.In, tx+LabelGap, mid+5, Start, labelSt)
					}
				}
			}
		}
		if len(st.Stores) > 0 {
			sx := x + c.BoxW/2 - groupWidth(st.Stores, storeSt.Face)/2
			for _, l := range st.Stores {
				sw := storeWidth(l, storeSt.Face)
				emitStore(s, l, sx, sw, x, by, side[i], c)
				sx += sw + StoreGap
			}
		}
	}
	s.W = 2*Margin + maxCols*colW + (maxCols-1)*gapW
	s.H = rowY[nRows-1] + c.BoxH + bottom
	return s, nil
}

// baselineOn is the text baseline that visually centres a label on a
// line at y, matching where a masking chip is drawn.
func baselineOn(y int, st theme.Style) int { return y + int(st.Size)/3 }

// emitLabelLines draws a label's explicit "\n" segments as a stack of
// lines vertically centered on the baseline y.
func emitLabelLines(s *Scene, label string, x, y int, anchor Anchor, st theme.Style) {
	lines := strings.Split(label, "\n")
	first := y - (len(lines)-1)*st.LineH()/2
	for j, ln := range lines {
		s.Items = append(s.Items, Text{X: x, Y: first + j*st.LineH(), S: ln, Anchor: anchor, Role: theme.Label})
	}
}

// groupWidth is the total width of a step's side-by-side store glyphs.
func groupWidth(links []ast.StoreLink, face font.Face) int {
	w := 0
	for i, l := range links {
		w += storeWidth(l, face)
		if i > 0 {
			w += StoreGap
		}
	}
	return w
}

// emitStore draws one datastore glyph spanning [sx, sx+sw] beside the
// box whose left edge is bx and top edge by, on the given side (+1
// above, -1 below), plus its arrows and their labels. Arrow x positions
// follow the glyph center but are clamped into the box span so arrows
// always leave the box.
func emitStore(s *Scene, l ast.StoreLink, sx, sw, bx, by, side int, c Config) {
	nameSt := c.Theme.Style(theme.StoreName)
	labelSt := c.Theme.Style(theme.Label)
	cx := sx + sw/2
	// near = glyph line facing the box; upper = the glyph's top line.
	var near, upper, boxEdge int
	if side >= 0 { // above
		near = by - StoreArrow
		upper = near - StoreH
		boxEdge = by
	} else { // below
		boxEdge = by + c.BoxH
		near = boxEdge + StoreArrow
		upper = near
	}
	lower := upper + StoreH
	s.Items = append(s.Items,
		Line{X1: sx, Y1: upper, X2: sx + sw, Y2: upper, Structural: true},
		Line{X1: sx, Y1: lower, X2: sx + sw, Y2: lower, Structural: true},
		Text{X: cx, Y: upper + (StoreH+int(nameSt.Size))/2 - 2, S: l.Name, Anchor: Middle, Role: theme.StoreName},
	)
	clamp := func(v int) int {
		if min := bx + 20; v < min {
			v = min
		}
		if max := bx + c.BoxW - 20; v > max {
			v = max
		}
		return v
	}
	// One arrow sits at the glyph center; a put/get pair sits at ±20.
	putX, getX := cx, cx
	if l.Put != nil && l.Get != nil {
		putX, getX = cx-20, cx+20
	}
	putX, getX = clamp(putX), clamp(getX)
	labelY := (boxEdge+near)/2 + 5
	dir := 1 // sign from box toward store
	if side < 0 {
		dir = -1
	}
	if l.Put != nil {
		s.Items = append(s.Items, Line{X1: putX, Y1: boxEdge, X2: putX, Y2: near + dir*Inset, Head: true})
		if l.Put.Label != "" {
			if l.Get != nil { // get's label takes the right side
				emitLabelLines(s, l.Put.Label, putX-LabelGap, labelY, End, labelSt)
			} else {
				emitLabelLines(s, l.Put.Label, putX+LabelGap, labelY, Start, labelSt)
			}
		}
	}
	if l.Get != nil {
		s.Items = append(s.Items, Line{X1: getX, Y1: near, X2: getX, Y2: boxEdge - dir*Inset, Head: true})
		if l.Get.Label != "" {
			emitLabelLines(s, l.Get.Label, getX+LabelGap, labelY, Start, labelSt)
		}
	}
}

// Scene is everything the renderers draw, in emit order.
type Scene struct {
	W, H, FontSize int
	Items          []Item
}

type Item interface{ item() }

type Rect struct{ X, Y, W, H int }

// Line is a straight segment. Head draws an arrowhead at (X2,Y2).
// Structural marks the datastore glyph lines, which a theme strokes
// differently from arrows.
type Line struct {
	X1, Y1, X2, Y2   int
	Head, Structural bool
}

type Anchor int

const (
	Start Anchor = iota
	Middle
	End
)

// Text is a single line of text; Y is the baseline. Role selects the
// typography a theme paints it with.
type Text struct {
	X, Y   int
	S      string
	Anchor Anchor
	Role   theme.Role
}

func (Rect) item() {}
func (Line) item() {}
func (Text) item() {}
