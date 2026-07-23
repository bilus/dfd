// Package layout turns an ast.Diagram into a Scene: a flat display list
// of primitives with final pixel coordinates.
package layout

import "github.com/bilus/dfd/ast"

// Geometry constants; normative values from the design spec.
const (
	HGap   = 90
	Margin = 40
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
	for i, st := range d.Steps {
		x := Margin + i*(c.BoxW+HGap)
		s.Items = append(s.Items,
			Rect{X: x, Y: Margin, W: c.BoxW, H: c.BoxH},
			Text{X: x + c.BoxW/2, Y: Margin + c.BoxH/2 + 5, S: st.Title, Anchor: Middle},
		)
	}
	s.W = 2*Margin + n*c.BoxW + (n-1)*HGap
	s.H = 2*Margin + c.BoxH
	return s, nil
}

// Scene is everything the renderers draw, in emit order.
type Scene struct {
	W, H, FontSize int
	Items          []Item
}

type Item interface{ item() }

type Rect struct{ X, Y, W, H int }

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
func (Text) item() {}
