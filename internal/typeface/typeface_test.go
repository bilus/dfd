package typeface_test

import (
	"testing"

	"golang.org/x/image/font"

	"github.com/bilus/dfd/internal/typeface"
)

func TestFacesMeasureText(t *testing.T) {
	cases := []struct {
		name string
		open func(float64) (font.Face, error)
	}{
		{"go regular", typeface.GoRegular},
		{"plex sans", typeface.PlexSans},
		{"plex sans semibold", typeface.PlexSansSemiBold},
		{"plex mono", typeface.PlexMono},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			face, err := c.open(13)
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			short := font.MeasureString(face, "abc").Ceil()
			long := font.MeasureString(face, "abcdef").Ceil()
			if short <= 0 || long <= short {
				t.Fatalf("widths: short=%d long=%d, want 0 < short < long", short, long)
			}
		})
	}
}

func TestSemiBoldIsWiderThanRegular(t *testing.T) {
	reg, err := typeface.PlexSans(13)
	if err != nil {
		t.Fatalf("PlexSans: %v", err)
	}
	semi, err := typeface.PlexSansSemiBold(13)
	if err != nil {
		t.Fatalf("PlexSansSemiBold: %v", err)
	}
	const s = "Store data in database"
	if font.MeasureString(semi, s).Ceil() <= font.MeasureString(reg, s).Ceil() {
		t.Error("semibold must measure wider than regular; layout depends on the distinction")
	}
}

func TestMonoIsFixedPitch(t *testing.T) {
	face, err := typeface.PlexMono(13)
	if err != nil {
		t.Fatalf("PlexMono: %v", err)
	}
	i := font.MeasureString(face, "iiii").Ceil()
	m := font.MeasureString(face, "mmmm").Ceil()
	if i != m {
		t.Errorf("monospace widths differ: iiii=%d mmmm=%d", i, m)
	}
}
