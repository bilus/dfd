// Package theme describes how a diagram is painted: colours, stroke
// weights, corner radii, and the typography of each kind of text.
// Layout measures with a theme's faces, so switching theme can change
// where things land, not just how they look.
package theme

import (
	"fmt"
	"strings"

	"golang.org/x/image/font"

	"github.com/bilus/dfd/internal/typeface"
)

// ChipPadX is the horizontal padding inside a label chip. Layout needs
// it to size column gaps; render needs it to draw the chip.
const ChipPadX = 8

// Role is the kind of text a scene item carries. A theme styles each
// role separately.
type Role int

const (
	Title     Role = iota // process box titles
	Label                 // arrow labels
	StoreName             // datastore names
	numRoles
)

// Style is the typography of one role.
type Style struct {
	Family string    // SVG font-family stack
	Size   float64   // pixels
	Weight int       // 0 leaves the weight unset
	Color  string    // hex, as written into the SVG
	Face   font.Face // measurement at 1x

	open func(size float64) (font.Face, error)
}

// FaceAt reopens this role's face scaled for rasterization.
func (s Style) FaceAt(scale float64) (font.Face, error) {
	return s.open(s.Size * scale)
}

// LineH is the baseline-to-baseline distance for stacked lines of this
// role. At the default 13px base this yields 17, the spacing the
// diagrams were originally drawn with.
func (s Style) LineH() int { return int(s.Size) * 4 / 3 }

// Theme is a complete painting recipe. The zero value is not usable;
// build one with Lookup.
type Theme struct {
	Name string

	Canvas      string // page background
	BoxFill     string
	BoxStroke   string
	BoxStrokeW  float64
	BoxRadius   int // rx on process boxes; 0 = square corners
	AccentW     int // left-edge accent bar width; 0 = no bar
	AccentColor string

	// Structural strokes are the datastore glyph lines.
	StructStroke  string
	StructStrokeW float64

	ArrowColor   string
	ArrowStrokeW float64
	ArrowHead    int // arrowhead marker size

	// LabelOnLine centres flow and turn labels on their arrow rather
	// than setting them beside it; LabelChip then masks the line where
	// the text crosses it.
	LabelOnLine bool
	LabelChip   bool

	styles [numRoles]Style
}

// Style returns the typography for one role.
func (t Theme) Style(r Role) Style { return t.styles[r] }

// Names lists the available themes, in the order they are offered.
func Names() []string { return []string{"default", "plex"} }

// Lookup builds the named theme with text sized relative to base.
func Lookup(name string, base int) (Theme, error) {
	switch name {
	case "default":
		return newDefault(base)
	case "plex":
		return newPlex(base)
	default:
		return Theme{}, fmt.Errorf("unknown theme %q (want %s)", name, strings.Join(Names(), " or "))
	}
}

// newDefault is the original hand-drawn look: black on white, square
// corners, one face throughout.
func newDefault(base int) (Theme, error) {
	face, err := typeface.GoRegular(float64(base))
	if err != nil {
		return Theme{}, err
	}
	const family = "Helvetica, Arial, sans-serif"
	st := Style{Family: family, Size: float64(base), Color: "#000", Face: face, open: typeface.GoRegular}
	t := Theme{
		Name:          "default",
		Canvas:        "#fff",
		BoxFill:       "#fff",
		BoxStroke:     "#000",
		BoxStrokeW:    2,
		StructStroke:  "#000",
		StructStrokeW: 2,
		ArrowColor:    "#000",
		ArrowStrokeW:  1.5,
		ArrowHead:     8,
	}
	t.styles = [numRoles]Style{Title: st, Label: st, StoreName: st}
	return t, nil
}

// newPlex is a cool-grey canvas with hairline boxes, a violet accent
// spine, and monospace arrow labels on chips.
func newPlex(base int) (Theme, error) {
	const (
		ink    = "#12161A"
		violet = "#63489E"
		paper  = "#F2F4F6"
		sans   = "IBM Plex Sans, Helvetica Neue, Arial, sans-serif"
		mono   = "IBM Plex Mono, SF Mono, Menlo, monospace"
	)
	titleSize, labelSize := base+2, base-1
	titleFace, err := typeface.PlexSansSemiBold(float64(titleSize))
	if err != nil {
		return Theme{}, err
	}
	labelFace, err := typeface.PlexMono(float64(labelSize))
	if err != nil {
		return Theme{}, err
	}
	storeFace, err := typeface.PlexSansSemiBold(float64(titleSize))
	if err != nil {
		return Theme{}, err
	}
	t := Theme{
		Name:          "plex",
		Canvas:        paper,
		BoxFill:       "#FFFFFF",
		BoxStroke:     ink,
		BoxStrokeW:    1.25,
		BoxRadius:     4,
		AccentW:       3,
		AccentColor:   violet,
		StructStroke:  ink,
		StructStrokeW: 1.25,
		ArrowColor:    violet,
		ArrowStrokeW:  1.4,
		ArrowHead:     7,
		LabelOnLine:   true,
		LabelChip:     true,
	}
	t.styles = [numRoles]Style{
		Title:     {Family: sans, Size: float64(titleSize), Weight: 600, Color: ink, Face: titleFace, open: typeface.PlexSansSemiBold},
		Label:     {Family: mono, Size: float64(labelSize), Color: violet, Face: labelFace, open: typeface.PlexMono},
		StoreName: {Family: sans, Size: float64(titleSize), Weight: 600, Color: ink, Face: storeFace, open: typeface.PlexSansSemiBold},
	}
	return t, nil
}
