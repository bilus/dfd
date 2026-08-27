// Package typeface provides the font faces used for text measurement
// and PNG rasterization. Every face is embedded in the binary so that
// output is identical on every machine, with no system font lookup.
package typeface

import (
	_ "embed"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
)

// IBM Plex is licensed under the SIL Open Font License 1.1; see
// fonts/OFL.txt, which ships alongside these files.
var (
	//go:embed fonts/IBMPlexSans-Regular.ttf
	plexSans []byte
	//go:embed fonts/IBMPlexSans-SemiBold.ttf
	plexSansSemiBold []byte
	//go:embed fonts/IBMPlexMono-Regular.ttf
	plexMono []byte
)

// GoRegular is the Go standard face, used by the default theme.
func GoRegular(size float64) (font.Face, error) { return parse(goregular.TTF, size) }

func PlexSans(size float64) (font.Face, error) { return parse(plexSans, size) }

func PlexSansSemiBold(size float64) (font.Face, error) { return parse(plexSansSemiBold, size) }

func PlexMono(size float64) (font.Face, error) { return parse(plexMono, size) }

// parse builds a face at the given pixel size (points at 72 DPI),
// unhinted so metrics do not vary with rasterization.
func parse(ttf []byte, size float64) (font.Face, error) {
	f, err := opentype.Parse(ttf)
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(f, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingNone,
	})
}
