// Package layout turns an ast.Diagram into a Scene: a flat display list
// of primitives with final pixel coordinates.
package layout

import (
	"fmt"

	"golang.org/x/image/font"

	"github.com/bilus/dfd/ast"
)

// Geometry constants; normative values from the design spec.
const (
	HGap       = 90
	VGap       = 90 // base vertical gap between rows
	Margin     = 40
	Inset      = 3  // gap between an arrow tip and the edge it points at
	LabelGap   = 8  // distance between an arrow and its label
	LanePad    = 30 // clearance between a store lane and the next row
	BoxPad     = 12 // padding between box border and title text
	LineH      = 17 // baseline-to-baseline distance for wrapped titles
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
	Face       font.Face // used for all text measurement; required
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
	if c.Face == nil {
		return nil, fmt.Errorf("layout: Config.Face is required")
	}
	s := &Scene{FontSize: c.FontSize}
	n := len(d.Steps)

	// Wrap titles up front; the tallest one sets the uniform box height.
	titles := make([][]string, n)
	boxH := c.BoxH
	for i, st := range d.Steps {
		titles[i] = WrapText(st.Title, c.BoxW-2*BoxPad, c.Face)
		if h := len(titles[i])*LineH + 2*BoxPad; h > boxH {
			boxH = h
		}
	}
	c.BoxH = boxH // c is a copy; the grown height flows to all consumers

	// Uniform column width: boxes, widened by the largest store group.
	colW := c.BoxW
	for _, st := range d.Steps {
		if g := groupWidth(st.Stores, c.Face); g > colW {
			colW = g
		}
	}

	perRow := c.PerRow
	if perRow <= 0 {
		perRow = 1
		for 2*Margin+(perRow+1)*colW+perRow*HGap <= c.MaxWidth {
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

	boxX := func(i int) int { return Margin + col[i]*(colW+HGap) + (colW-c.BoxW)/2 }

	for i, st := range d.Steps {
		r := i / perRow
		x, by := boxX(i), rowY[r]
		cy := by + c.BoxH/2
		s.Items = append(s.Items, Rect{X: x, Y: by, W: c.BoxW, H: c.BoxH})
		first := cy + 5 - (len(titles[i])-1)*LineH/2
		for j, ln := range titles[i] {
			s.Items = append(s.Items, Text{X: x + c.BoxW/2, Y: first + j*LineH, S: ln, Anchor: Middle})
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
					s.Items = append(s.Items, Text{X: (fromX + x + c.BoxW) / 2, Y: cy - LabelGap, S: st.In, Anchor: Middle})
				}
			} else { // snake turn: vertical arrow into this row's first box
				tx := x + c.BoxW/2
				fromY := rowY[pr] + c.BoxH
				s.Items = append(s.Items, Line{X1: tx, Y1: fromY, X2: tx, Y2: by - Inset, Head: true})
				if st.In != "" {
					s.Items = append(s.Items, Text{X: tx + LabelGap, Y: (fromY+by)/2 + 5, S: st.In, Anchor: Start})
				}
			}
		}
		if len(st.Stores) > 0 {
			sx := x + c.BoxW/2 - groupWidth(st.Stores, c.Face)/2
			for _, l := range st.Stores {
				sw := storeWidth(l, c.Face)
				emitStore(s, l, sx, sw, x, by, side[i], c)
				sx += sw + StoreGap
			}
		}
	}
	s.W = 2*Margin + maxCols*colW + (maxCols-1)*HGap
	s.H = rowY[nRows-1] + c.BoxH + bottom
	return s, nil
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
		Line{X1: sx, Y1: upper, X2: sx + sw, Y2: upper, Thick: true},
		Line{X1: sx, Y1: lower, X2: sx + sw, Y2: lower, Thick: true},
		Text{X: cx, Y: upper + (StoreH+c.FontSize)/2 - 2, S: l.Name, Anchor: Middle},
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
				s.Items = append(s.Items, Text{X: putX - LabelGap, Y: labelY, S: l.Put.Label, Anchor: End})
			} else {
				s.Items = append(s.Items, Text{X: putX + LabelGap, Y: labelY, S: l.Put.Label, Anchor: Start})
			}
		}
	}
	if l.Get != nil {
		s.Items = append(s.Items, Line{X1: getX, Y1: near, X2: getX, Y2: boxEdge - dir*Inset, Head: true})
		if l.Get.Label != "" {
			s.Items = append(s.Items, Text{X: getX + LabelGap, Y: labelY, S: l.Get.Label, Anchor: Start})
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
// Thick marks 2px structural strokes (datastore glyph lines) versus
// 1.5px arrow strokes.
type Line struct {
	X1, Y1, X2, Y2 int
	Head, Thick    bool
}

type Anchor int

const (
	Start Anchor = iota
	Middle
	End
)

// Text is a single line of text; Y is the baseline.
type Text struct {
	X, Y   int
	S      string
	Anchor Anchor
}

func (Rect) item() {}
func (Line) item() {}
func (Text) item() {}
