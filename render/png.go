package render

import (
	"io"
	"math"

	"github.com/fogleman/gg"

	"github.com/bilus/dfd/internal/typeface"
	"github.com/bilus/dfd/layout"
)

// PNG rasterizes the scene at the given scale. All geometry is drawn in
// scaled coordinates with a face sized scale x so text stays crisp.
func PNG(s *layout.Scene, scale float64, w io.Writer) error {
	face, err := typeface.New(float64(s.FontSize) * scale)
	if err != nil {
		return err
	}
	dc := gg.NewContext(int(math.Ceil(float64(s.W)*scale)), int(math.Ceil(float64(s.H)*scale)))
	dc.SetFontFace(face)
	dc.SetRGB(1, 1, 1)
	dc.Clear()
	px := func(v int) float64 { return float64(v) * scale }
	for _, it := range s.Items {
		dc.SetRGB(0, 0, 0)
		switch v := it.(type) {
		case layout.Rect:
			dc.SetRGB(1, 1, 1)
			dc.DrawRectangle(px(v.X), px(v.Y), px(v.W), px(v.H))
			dc.FillPreserve()
			dc.SetRGB(0, 0, 0)
			dc.SetLineWidth(2 * scale)
			dc.Stroke()
		case layout.Line:
			width := 1.5
			if v.Thick {
				width = 2
			}
			dc.SetLineWidth(width * scale)
			dc.DrawLine(px(v.X1), px(v.Y1), px(v.X2), px(v.Y2))
			dc.Stroke()
			if v.Head {
				drawHead(dc, px(v.X1), px(v.Y1), px(v.X2), px(v.Y2), 8*scale)
			}
		case layout.Text:
			ax := 0.0
			switch v.Anchor {
			case layout.Middle:
				ax = 0.5
			case layout.End:
				ax = 1
			}
			dc.DrawStringAnchored(v.S, px(v.X), px(v.Y), ax, 0)
		}
	}
	return dc.EncodePNG(w)
}

// drawHead draws a filled triangular arrowhead of the given length with
// its tip at (x2, y2), oriented along the segment direction.
func drawHead(dc *gg.Context, x1, y1, x2, y2, size float64) {
	dx, dy := x2-x1, y2-y1
	l := math.Hypot(dx, dy)
	if l == 0 {
		return
	}
	ux, uy := dx/l, dy/l
	bx, by := x2-size*ux, y2-size*uy // base center
	wx, wy := -uy*size*0.4, ux*size*0.4
	dc.MoveTo(x2, y2)
	dc.LineTo(bx+wx, by+wy)
	dc.LineTo(bx-wx, by-wy)
	dc.ClosePath()
	dc.Fill()
}
