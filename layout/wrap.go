package layout

import (
	"strings"

	"golang.org/x/image/font"
)

// WrapText greedily wraps s at word boundaries so every line fits maxW.
// A single word wider than maxW is broken mid-word.
func WrapText(s string, maxW int, face font.Face) []string {
	var lines []string
	var cur string
	flush := func() {
		if cur != "" {
			lines = append(lines, cur)
			cur = ""
		}
	}
	for _, w := range strings.Fields(s) {
		for TextWidth(w, face) > maxW {
			flush()
			i := len(w)
			for i > 1 && TextWidth(w[:i], face) > maxW {
				i--
			}
			lines = append(lines, w[:i])
			w = w[i:]
		}
		switch {
		case cur == "":
			cur = w
		case TextWidth(cur+" "+w, face) <= maxW:
			cur += " " + w
		default:
			flush()
			cur = w
		}
	}
	flush()
	if lines == nil {
		lines = []string{""}
	}
	return lines
}
