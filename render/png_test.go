package render_test

import (
	"bytes"
	"image/png"
	"testing"

	"github.com/bilus/dfd/layout"
	"github.com/bilus/dfd/render"
)

func pngScene() *layout.Scene {
	return &layout.Scene{W: 300, H: 100, FontSize: 13, Items: []layout.Item{
		layout.Rect{X: 10, Y: 20, W: 160, H: 60},
		layout.Line{X1: 170, Y1: 50, X2: 250, Y2: 50, Head: true},
		layout.Line{X1: 10, Y1: 90, X2: 170, Y2: 90, Thick: true},
		layout.Text{X: 90, Y: 55, S: "hello", Anchor: layout.Middle},
	}}
}

func TestPNGDeterministicAndScaled(t *testing.T) {
	var a, b bytes.Buffer
	if err := render.PNG(pngScene(), 2, &a); err != nil {
		t.Fatalf("PNG: %v", err)
	}
	if err := render.PNG(pngScene(), 2, &b); err != nil {
		t.Fatalf("PNG: %v", err)
	}
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Error("PNG output is not deterministic")
	}
	img, err := png.Decode(&a)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if w, h := img.Bounds().Dx(), img.Bounds().Dy(); w != 600 || h != 200 {
		t.Errorf("size = %dx%d, want 600x200 (scale 2)", w, h)
	}
	// Box stroke lands as a dark pixel at the scaled top-left corner.
	r, g, bl, _ := img.At(20, 40).RGBA()
	if r > 0x4000 && g > 0x4000 && bl > 0x4000 {
		t.Errorf("expected dark stroke pixel at (20,40), got r=%x g=%x b=%x", r, g, bl)
	}
	// Canvas background is white.
	r, g, bl, _ = img.At(590, 190).RGBA()
	if r < 0xc000 || g < 0xc000 || bl < 0xc000 {
		t.Errorf("expected white background pixel at (590,190), got r=%x g=%x b=%x", r, g, bl)
	}
}
