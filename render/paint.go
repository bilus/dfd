package render

import (
	"github.com/bilus/dfd/layout"
	"github.com/bilus/dfd/theme"
)

// canvas is the small surface a backend has to provide. Everything a
// theme decides (shadows, accent bars, chips, which stroke an arrow
// gets, how a role is styled) is settled once in paint, so the two
// backends cannot drift apart.
type canvas interface {
	// Rect draws a rectangle. An empty stroke means fill only; an empty
	// dash means a solid outline.
	Rect(x, y, w, h, radius int, fill, stroke, dash string, strokeW float64)
	Line(x1, y1, x2, y2 int, stroke string, strokeW float64, head bool)
	Text(x, y int, s string, anchor layout.Anchor, st theme.Style)
}

// paint walks the scene once, turning theme decisions into primitives.
func paint(s *layout.Scene, th theme.Theme, c canvas) {
	for _, it := range s.Items {
		switch v := it.(type) {
		case layout.Rect:
			fill, stroke, dash := th.BoxFill, th.BoxStroke, ""
			if v.Entity {
				fill, stroke, dash = th.EntityFill, th.EntityStroke, th.EntityDash
				if th.EntityShadow > 0 {
					c.Rect(v.X+th.EntityShadow, v.Y+th.EntityShadow, v.W, v.H, th.BoxRadius,
						fill, stroke, "", th.BoxStrokeW)
				}
			}
			c.Rect(v.X, v.Y, v.W, v.H, th.BoxRadius, fill, stroke, dash, th.BoxStrokeW)
			if th.AccentW > 0 && (!v.Entity || th.EntityAccent) {
				y, h := insetAccent(v, th)
				c.Rect(v.X, y, th.AccentW, h, 0, th.AccentColor, "", "", 0)
			}
		case layout.Line:
			stroke, width := th.ArrowColor, th.ArrowStrokeW
			if v.Structural {
				stroke, width = th.StructStroke, th.StructStrokeW
			}
			c.Line(v.X1, v.Y1, v.X2, v.Y2, stroke, width, v.Head)
		case layout.Text:
			st := th.Style(v.Role)
			if th.LabelChip && v.Role == theme.Label {
				x, y, w, h := chip(v, st)
				c.Rect(x, y, w, h, 3, th.Canvas, "", "", 0)
			}
			c.Text(v.X, v.Y, v.S, v.Anchor, st)
		}
	}
}

// chip is the box drawn behind an arrow label so the line it crosses
// does not run through the text.
func chip(t layout.Text, st theme.Style) (x, y, w, h int) {
	const padX = theme.ChipPadX
	w = layout.TextWidth(t.S, st.Face) + 2*padX
	// Exactly one line tall, so the chips of a wrapped label tile
	// instead of erasing the line above.
	h = st.LineH()
	switch t.Anchor {
	case layout.Middle:
		x = t.X - w/2
	case layout.End:
		x = t.X + padX - w
	default:
		x = t.X - padX
	}
	return x, t.Y - int(st.Size)/3 - h/2, w, h
}

// insetAccent shortens the accent bar by the corner radius so it stays
// inside the box outline instead of poking past the rounded corners.
func insetAccent(r layout.Rect, th theme.Theme) (y, h int) {
	return r.Y + th.BoxRadius, r.H - 2*th.BoxRadius
}
