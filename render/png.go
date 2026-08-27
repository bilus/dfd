package render

import (
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"

	"github.com/bilus/dfd/layout"
	"github.com/bilus/dfd/theme"
)

// PNG rasterizes the scene at the given scale. Geometry is drawn in
// scaled coordinates with faces sized scale x so text stays crisp.
func PNG(s *layout.Scene, th theme.Theme, scale float64, w io.Writer) error {
	faces, err := scaledFaces(th, scale)
	if err != nil {
		return err
	}
	dc := gg.NewContext(int(math.Ceil(float64(s.W)*scale)), int(math.Ceil(float64(s.H)*scale)))
	dc.SetHexColor(th.Canvas)
	dc.Clear()
	px := func(v int) float64 { return float64(v) * scale }
	for _, it := range s.Items {
		switch v := it.(type) {
		case layout.Rect:
			fill, stroke := th.BoxFill, th.BoxStroke
			if v.Entity {
				fill, stroke = th.EntityFill, th.EntityStroke
				if th.EntityShadow > 0 {
					fillStroke(dc, px(v.X+th.EntityShadow), px(v.Y+th.EntityShadow), px(v.W), px(v.H),
						float64(th.BoxRadius)*scale, fill, stroke, th.BoxStrokeW*scale)
				}
				dc.SetDash(dashPattern(th.EntityDash, scale)...)
			}
			fillStroke(dc, px(v.X), px(v.Y), px(v.W), px(v.H), float64(th.BoxRadius)*scale,
				fill, stroke, th.BoxStrokeW*scale)
			dc.SetDash()
			if th.AccentW > 0 && (!v.Entity || th.EntityAccent) {
				ay, ah := insetAccent(v, th)
				dc.SetHexColor(th.AccentColor)
				dc.DrawRectangle(px(v.X), px(ay), px(th.AccentW), px(ah))
				dc.Fill()
			}
		case layout.Line:
			stroke, width := th.ArrowColor, th.ArrowStrokeW
			if v.Structural {
				stroke, width = th.StructStroke, th.StructStrokeW
			}
			dc.SetHexColor(stroke)
			dc.SetLineWidth(width * scale)
			dc.DrawLine(px(v.X1), px(v.Y1), px(v.X2), px(v.Y2))
			dc.Stroke()
			if v.Head {
				drawHead(dc, px(v.X1), px(v.Y1), px(v.X2), px(v.Y2), float64(th.ArrowHead)*scale)
			}
		case layout.Text:
			st := th.Style(v.Role)
			if th.LabelChip && v.Role == theme.Label {
				x, y, cw, ch := chip(v, st)
				fillStroke(dc, px(x), px(y), px(cw), px(ch), 3*scale, th.Canvas, "", 0)
			}
			dc.SetHexColor(st.Color)
			dc.SetFontFace(faces[v.Role])
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

// scaledFaces reopens every role's face at the raster scale, since the
// theme's own faces are sized for layout at 1x.
func scaledFaces(th theme.Theme, scale float64) (map[theme.Role]font.Face, error) {
	out := make(map[theme.Role]font.Face, len(theme.Roles()))
	for _, r := range theme.Roles() {
		f, err := th.Style(r).FaceAt(scale)
		if err != nil {
			return nil, err
		}
		out[r] = f
	}
	return out, nil
}

// dashPattern turns an SVG stroke-dasharray into gg dash lengths.
func dashPattern(dash string, scale float64) []float64 {
	var out []float64
	for _, f := range strings.Fields(dash) {
		v, err := strconv.ParseFloat(f, 64)
		if err != nil {
			return nil
		}
		out = append(out, v*scale)
	}
	return out
}

// fillStroke paints a rectangle, rounding its corners when r > 0 and
// stroking it when a stroke colour is given.
func fillStroke(dc *gg.Context, x, y, w, h, r float64, fill, stroke string, width float64) {
	if r > 0 {
		dc.DrawRoundedRectangle(x, y, w, h, r)
	} else {
		dc.DrawRectangle(x, y, w, h)
	}
	if fill == "none" {
		dc.SetLineWidth(width)
		dc.SetHexColor(stroke)
		dc.Stroke()
		return
	}
	dc.SetHexColor(fill)
	if stroke == "" {
		dc.Fill()
		return
	}
	dc.FillPreserve()
	dc.SetHexColor(stroke)
	dc.SetLineWidth(width)
	dc.Stroke()
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
