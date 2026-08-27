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
	dc := gg.NewContext(int(math.Ceil(float64(s.W)*scale)), int(math.Ceil(float64(s.H)*scale)))
	dc.SetHexColor(th.Canvas)
	dc.Clear()
	c := &pngCanvas{dc: dc, th: th, scale: scale, faces: map[faceKey]font.Face{}}
	paint(s, th, c)
	if c.err != nil {
		return c.err
	}
	return dc.EncodePNG(w)
}

// faceKey identifies a face by what a theme.Style says about it, so a
// role added later needs no list here to be rasterized.
type faceKey struct {
	family string
	size   float64
}

type pngCanvas struct {
	dc    *gg.Context
	th    theme.Theme
	scale float64
	faces map[faceKey]font.Face
	err   error
}

func (c *pngCanvas) px(v int) float64 { return float64(v) * c.scale }

func (c *pngCanvas) Rect(x, y, w, h, radius int, fill, stroke, dash string, strokeW float64) {
	if radius > 0 {
		c.dc.DrawRoundedRectangle(c.px(x), c.px(y), c.px(w), c.px(h), float64(radius)*c.scale)
	} else {
		c.dc.DrawRectangle(c.px(x), c.px(y), c.px(w), c.px(h))
	}
	if stroke == "" {
		c.dc.SetHexColor(fill)
		c.dc.Fill()
		return
	}
	if fill != "none" {
		c.dc.SetHexColor(fill)
		c.dc.FillPreserve()
	}
	if dash != "" {
		c.dc.SetDash(dashPattern(dash, c.scale)...)
		defer c.dc.SetDash()
	}
	c.dc.SetHexColor(stroke)
	c.dc.SetLineWidth(strokeW * c.scale)
	c.dc.Stroke()
}

func (c *pngCanvas) Line(x1, y1, x2, y2 int, stroke string, strokeW float64, head bool) {
	c.dc.SetHexColor(stroke)
	c.dc.SetLineWidth(strokeW * c.scale)
	c.dc.DrawLine(c.px(x1), c.px(y1), c.px(x2), c.px(y2))
	c.dc.Stroke()
	if head {
		c.drawHead(c.px(x1), c.px(y1), c.px(x2), c.px(y2), float64(c.th.ArrowHead)*c.scale)
	}
}

func (c *pngCanvas) Text(x, y int, s string, anchor layout.Anchor, st theme.Style) {
	face, err := c.face(st)
	if err != nil {
		if c.err == nil {
			c.err = err
		}
		return
	}
	c.dc.SetFontFace(face)
	c.dc.SetHexColor(st.Color)
	ax := 0.0
	switch anchor {
	case layout.Middle:
		ax = 0.5
	case layout.End:
		ax = 1
	}
	c.dc.DrawStringAnchored(s, c.px(x), c.px(y), ax, 0)
}

// face reopens a style's face at the raster scale, once per style.
func (c *pngCanvas) face(st theme.Style) (font.Face, error) {
	k := faceKey{st.Family, st.Size}
	if f, ok := c.faces[k]; ok {
		return f, nil
	}
	f, err := st.FaceAt(c.scale)
	if err != nil {
		return nil, err
	}
	c.faces[k] = f
	return f, nil
}

// drawHead draws a filled triangular arrowhead of the given length with
// its tip at (x2, y2), oriented along the segment direction.
func (c *pngCanvas) drawHead(x1, y1, x2, y2, size float64) {
	dx, dy := x2-x1, y2-y1
	l := math.Hypot(dx, dy)
	if l == 0 {
		return
	}
	ux, uy := dx/l, dy/l
	bx, by := x2-size*ux, y2-size*uy // base center
	wx, wy := -uy*size*0.4, ux*size*0.4
	c.dc.MoveTo(x2, y2)
	c.dc.LineTo(bx+wx, by+wy)
	c.dc.LineTo(bx-wx, by-wy)
	c.dc.ClosePath()
	c.dc.Fill()
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
