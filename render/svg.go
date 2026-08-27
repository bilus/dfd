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
	paint(s, th, &svgCanvas{p: p, root: root, rootSize: s.FontSize})
	p.f("</svg>\n")
	return p.err
}

type svgCanvas struct {
	p        *printer
	root     theme.Style
	rootSize int
}

func (c *svgCanvas) Rect(x, y, w, h, radius int, fill, stroke, dash string, strokeW float64) {
	outline := ""
	if stroke != "" {
		outline = fmt.Sprintf(` stroke="%s" stroke-width="%s"`, stroke, num(strokeW))
	}
	c.p.f(`  <rect x="%d" y="%d" width="%d" height="%d"%s fill="%s"%s%s/>`+"\n",
		x, y, w, h, radiusAttr(radius), fill, outline, dashAttr(dash))
}

func (c *svgCanvas) Line(x1, y1, x2, y2 int, stroke string, strokeW float64, head bool) {
	marker := ""
	if head {
		marker = ` marker-end="url(#ah)"`
	}
	c.p.f(`  <line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="%s"%s/>`+"\n",
		x1, y1, x2, y2, stroke, num(strokeW), marker)
}

func (c *svgCanvas) Text(x, y int, s string, anchor layout.Anchor, st theme.Style) {
	c.p.f(`  <text x="%d" y="%d" fill="%s"%s%s%s%s>%s</text>`+"\n",
		x, y, st.Color, anchorAttr(anchor), familyAttr(st, c.root), sizeAttr(st, c.rootSize), weightAttr(st), escape(s))
}

// num formats a stroke width the way it is written by hand: 2, not 2.0.
func num(f float64) string { return strconv.FormatFloat(f, 'g', -1, 64) }

func dashAttr(dash string) string {
	if dash == "" {
		return ""
	}
	return fmt.Sprintf(` stroke-dasharray="%s"`, dash)
}

func radiusAttr(radius int) string {
	if radius == 0 {
		return ""
	}
	return fmt.Sprintf(` rx="%d"`, radius)
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
