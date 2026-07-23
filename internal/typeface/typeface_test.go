package typeface_test

import (
	"testing"

	"golang.org/x/image/font"

	"github.com/bilus/dfd/internal/typeface"
)

func TestNewFaceMeasuresText(t *testing.T) {
	face, err := typeface.New(13)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	short := font.MeasureString(face, "abc").Ceil()
	long := font.MeasureString(face, "abcdef").Ceil()
	if short <= 0 || long <= short {
		t.Fatalf("widths: short=%d long=%d, want 0 < short < long", short, long)
	}
}
