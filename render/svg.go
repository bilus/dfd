// Package render draws a layout.Scene as SVG or PNG in a given theme.
package render

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/bilus/dfd/layout"
	"github.com/bilus/dfd/theme"
)

// SVG writes the scene as a standalone SVG document. Output is
// deterministic: items are emitted in scene order with fixed formatting.
func SVG(s *layout.Scene, th theme.Theme, w io.Writer) error {
	root := th.Style(theme.Title)
	p := &printer{w: w}
	p.f(`<svg viewBox="0 0 %d %d" width="%d" height="%d" xmlns="http://www.w3.org/2000/svg" font-family="%s" font-size="%d">`+"\n",
		s.W, s.H, s.W, s.H, root.Family, s.FontSize)
	p.f("  <defs>\n    <marker id=\"ah\" viewBox=\"0 0 10 10\" refX=\"9\" refY=\"5\" markerWidth=\"%d\" markerHeight=\"%d\" orient=\"auto\">\n      <path d=\"M0,0L10,5L0,10z\" fill=\"%s\"/>\n    </marker>\n  </defs>\n",
		th.ArrowHead, th.ArrowHead, th.ArrowColor)
	p.f(`  <rect x="0" y="0" width="%d" height="%d" fill="%s"/>`+"\n", s.W, s.H, th.Canvas)
	for _, it := range s.Items {
		switch v := it.(type) {
		case layout.Rect:
			p.f(`  <rect x="%d" y="%d" width="%d" height="%d"%s fill="%s" stroke="%s" stroke-width="%s"/>`+"\n",
				v.X, v.Y, v.W, v.H, radius(th), th.BoxFill, th.BoxStroke, num(th.BoxStrokeW))
			if th.AccentW > 0 {
				p.f(`  <rect x="%d" y="%d" width="%d" height="%d" fill="%s"/>`+"\n",
					v.X, v.Y, th.AccentW, v.H, th.AccentColor)
			}
		case layout.Line:
			stroke, width := th.ArrowColor, th.ArrowStrokeW
			if v.Structural {
				stroke, width = th.StructStroke, th.StructStrokeW
			}
			marker := ""
			if v.Head {
				marker = ` marker-end="url(#ah)"`
			}
			p.f(`  <line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="%s"%s/>`+"\n",
				v.X1, v.Y1, v.X2, v.Y2, stroke, num(width), marker)
		case layout.Text:
			st := th.Style(v.Role)
			if th.LabelChip && v.Role == theme.Label {
				x, y, cw, ch := chip(v, st)
				p.f(`  <rect x="%d" y="%d" width="%d" height="%d" rx="3" fill="%s"/>`+"\n", x, y, cw, ch, th.Canvas)
			}
			p.f(`  <text x="%d" y="%d" fill="%s"%s%s%s%s>%s</text>`+"\n",
				v.X, v.Y, st.Color, anchorAttr(v.Anchor), familyAttr(st, root), sizeAttr(st, s.FontSize), weightAttr(st), escape(v.S))
		}
	}
	p.f("</svg>\n")
	return p.err
}

// chip is the box drawn behind an arrow label so the line it crosses
// does not run through the text.
func chip(t layout.Text, st theme.Style) (x, y, w, h int) {
	const padX = 8
	w = layout.TextWidth(t.S, st.Face) + 2*padX
	h = int(st.Size) + 12
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

// num formats a stroke width the way it is written by hand: 2, not 2.0.
func num(f float64) string { return strconv.FormatFloat(f, 'g', -1, 64) }

func radius(th theme.Theme) string {
	if th.BoxRadius == 0 {
		return ""
	}
	return fmt.Sprintf(` rx="%d"`, th.BoxRadius)
}

func anchorAttr(a layout.Anchor) string {
	switch a {
	case layout.Middle:
		return ` text-anchor="middle"`
	case layout.End:
		return ` text-anchor="end"`
	}
	return ""
}

// The root <svg> carries the title family and the base size, so text
// only names its own when it differs.
func familyAttr(st, root theme.Style) string {
	if st.Family == root.Family {
		return ""
	}
	return fmt.Sprintf(` font-family="%s"`, st.Family)
}

func sizeAttr(st theme.Style, rootSize int) string {
	if st.Size == float64(rootSize) {
		return ""
	}
	return fmt.Sprintf(` font-size="%s"`, num(st.Size))
}

func weightAttr(st theme.Style) string {
	if st.Weight == 0 {
		return ""
	}
	return fmt.Sprintf(` font-weight="%d"`, st.Weight)
}

// escape makes s safe for SVG text content.
func escape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}

// printer accumulates the first write error so call sites stay linear.
type printer struct {
	w   io.Writer
	err error
}

func (p *printer) f(format string, args ...any) {
	if p.err != nil {
		return
	}
	_, p.err = fmt.Fprintf(p.w, format, args...)
}
