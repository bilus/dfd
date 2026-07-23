// Package layout turns an ast.Diagram into a Scene: a flat display list
// of primitives with final pixel coordinates.
package layout

import "github.com/bilus/dfd/ast"

// Geometry constants; normative values from the design spec.
const (
	HGap       = 90
	Margin     = 40
	Inset      = 3 // gap between an arrow tip and the edge it points at
	LabelGap   = 8 // distance between an arrow and its label
	StoreW     = 150
	StoreH     = 36 // distance between the two datastore glyph lines
	StoreArrow = 64 // length of a box<->datastore arrow
)

type Config struct {
	BoxW, BoxH int
	MaxWidth   int
	PerRow     int // 0 = derive from MaxWidth
	FontSize   int
}

// Arrange lays the diagram out on a grid and returns the display list.
func Arrange(d *ast.Diagram, c Config) (*Scene, error) {
	s := &Scene{FontSize: c.FontSize}
	n := len(d.Steps)
	rowY := Margin
	for _, st := range d.Steps {
		if len(st.Stores) > 0 {
			rowY = Margin + StoreArrow + StoreH // top lane holds the stores
			break
		}
	}
	cy := rowY + c.BoxH/2
	for i, st := range d.Steps {
		x := Margin + i*(c.BoxW+HGap)
		s.Items = append(s.Items,
			Rect{X: x, Y: rowY, W: c.BoxW, H: c.BoxH},
			Text{X: x + c.BoxW/2, Y: cy + 5, S: st.Title, Anchor: Middle},
		)
		if i > 0 {
			s.Items = append(s.Items, Line{X1: x - HGap, Y1: cy, X2: x - Inset, Y2: cy, Head: true})
			if st.In != "" {
				s.Items = append(s.Items, Text{X: x - HGap/2, Y: cy - LabelGap, S: st.In, Anchor: Middle})
			}
		}
		for _, l := range st.Stores {
			emitStore(s, l, x, rowY, c)
		}
	}
	s.W = 2*Margin + n*c.BoxW + (n-1)*HGap
	s.H = rowY + c.BoxH + Margin
	return s, nil
}

// emitStore draws one datastore glyph above the box at bx plus its
// arrows and their labels.
func emitStore(s *Scene, l ast.StoreLink, bx, by int, c Config) {
	cx := bx + c.BoxW/2
	sx := cx - StoreW/2
	line2 := by - StoreArrow // glyph line nearest the box
	line1 := line2 - StoreH
	s.Items = append(s.Items,
		Line{X1: sx, Y1: line1, X2: sx + StoreW, Y2: line1, Thick: true},
		Line{X1: sx, Y1: line2, X2: sx + StoreW, Y2: line2, Thick: true},
		Text{X: cx, Y: line1 + (StoreH+c.FontSize)/2 - 2, S: l.Name, Anchor: Middle},
	)
	// One arrow sits at the box center; a put/get pair sits at ±20.
	putX, getX := cx, cx
	if l.Put != nil && l.Get != nil {
		putX, getX = cx-20, cx+20
	}
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
