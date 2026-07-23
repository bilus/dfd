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
	Margin     = 40
	Inset      = 3 // gap between an arrow tip and the edge it points at
	LabelGap   = 8 // distance between an arrow and its label
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

	// Uniform column width: boxes, widened by the largest store group.
	colW := c.BoxW
	for _, st := range d.Steps {
		if g := groupWidth(st.Stores, c.Face); g > colW {
			colW = g
		}
	}

	rowY := Margin
	for _, st := range d.Steps {
		if len(st.Stores) > 0 {
			rowY = Margin + StoreArrow + StoreH // top lane holds the stores
			break
		}
	}
	cy := rowY + c.BoxH/2
	for i, st := range d.Steps {
		colX := Margin + i*(colW+HGap)
		x := colX + (colW-c.BoxW)/2
		s.Items = append(s.Items,
			Rect{X: x, Y: rowY, W: c.BoxW, H: c.BoxH},
			Text{X: x + c.BoxW/2, Y: cy + 5, S: st.Title, Anchor: Middle},
		)
		if i > 0 {
			prevRight := colX - HGap - (colW-c.BoxW)/2
			s.Items = append(s.Items, Line{X1: prevRight, Y1: cy, X2: x - Inset, Y2: cy, Head: true})
			if st.In != "" {
				s.Items = append(s.Items, Text{X: (prevRight + x) / 2, Y: cy - LabelGap, S: st.In, Anchor: Middle})
			}
		}
		if len(st.Stores) > 0 {
			sx := x + c.BoxW/2 - groupWidth(st.Stores, c.Face)/2
			for _, l := range st.Stores {
				sw := storeWidth(l, c.Face)
				emitStore(s, l, sx, sw, x, rowY, c)
				sx += sw + StoreGap
			}
		}
	}
	s.W = 2*Margin + n*colW + (n-1)*HGap
	s.H = rowY + c.BoxH + Margin
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

// emitStore draws one datastore glyph spanning [sx, sx+sw] above the
// box whose left edge is bx, plus its arrows and their labels. Arrow x
// positions follow the glyph center but are clamped into the box span
// so arrows always leave the box.
func emitStore(s *Scene, l ast.StoreLink, sx, sw, bx, by int, c Config) {
	cx := sx + sw/2
	line2 := by - StoreArrow // glyph line nearest the box
	line1 := line2 - StoreH
	s.Items = append(s.Items,
		Line{X1: sx, Y1: line1, X2: sx + sw, Y2: line1, Thick: true},
		Line{X1: sx, Y1: line2, X2: sx + sw, Y2: line2, Thick: true},
		Text{X: cx, Y: line1 + (StoreH+c.FontSize)/2 - 2, S: l.Name, Anchor: Middle},
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
	labelY := (by+line2)/2 + 5
	if l.Put != nil {
		s.Items = append(s.Items, Line{X1: putX, Y1: by, X2: putX, Y2: line2 + Inset, Head: true})
		if l.Put.Label != "" {
			if l.Get != nil { // get's label takes the right side
				s.Items = append(s.Items, Text{X: putX - LabelGap, Y: labelY, S: l.Put.Label, Anchor: End})
			} else {
				s.Items = append(s.Items, Text{X: putX + LabelGap, Y: labelY, S: l.Put.Label, Anchor: Start})
			}
		}
	}
	if l.Get != nil {
		s.Items = append(s.Items, Line{X1: getX, Y1: line2, X2: getX, Y2: by - Inset, Head: true})
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
