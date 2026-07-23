// Package render draws a layout.Scene as SVG.
package render

import (
	"fmt"
	"io"
	"strings"

	"github.com/bilus/dfd/layout"
)

// SVG writes the scene as a standalone SVG document. Output is
// deterministic: items are emitted in scene order with fixed formatting.
func SVG(s *layout.Scene, w io.Writer) error {
	p := &printer{w: w}
	p.f(`<svg viewBox="0 0 %d %d" width="%d" height="%d" xmlns="http://www.w3.org/2000/svg" font-family="Helvetica, Arial, sans-serif" font-size="%d">`+"\n",
		s.W, s.H, s.W, s.H, s.FontSize)
	p.f("  <defs>\n    <marker id=\"ah\" viewBox=\"0 0 10 10\" refX=\"9\" refY=\"5\" markerWidth=\"8\" markerHeight=\"8\" orient=\"auto\">\n      <path d=\"M0,0L10,5L0,10z\" fill=\"#000\"/>\n    </marker>\n  </defs>\n")
	p.f(`  <rect x="0" y="0" width="%d" height="%d" fill="#fff"/>`+"\n", s.W, s.H)
	for _, it := range s.Items {
		switch v := it.(type) {
		case layout.Rect:
			p.f(`  <rect x="%d" y="%d" width="%d" height="%d" fill="#fff" stroke="#000" stroke-width="2"/>`+"\n", v.X, v.Y, v.W, v.H)
		case layout.Line:
			width, marker := "1.5", ""
			if v.Thick {
				width = "2"
			}
			if v.Head {
				marker = ` marker-end="url(#ah)"`
			}
			p.f(`  <line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#000" stroke-width="%s"%s/>`+"\n", v.X1, v.Y1, v.X2, v.Y2, width, marker)
		case layout.Text:
			anchor := ""
			switch v.Anchor {
			case layout.Middle:
				anchor = ` text-anchor="middle"`
			case layout.End:
				anchor = ` text-anchor="end"`
			}
			p.f(`  <text x="%d" y="%d" fill="#000"%s>%s</text>`+"\n", v.X, v.Y, anchor, escape(v.S))
		}
	}
	p.f("</svg>\n")
	return p.err
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
