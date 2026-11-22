package app

import (
	"image"
	"image/color"
	"testing"
)

func TestWithMask(t *testing.T) {
	t.Parallel()

	img := image.NewNRGBA(image.Rect(10, 20, 13, 25)) // 3×5, not at the origin

	for y := 20; y < 25; y++ {
		for x := 10; x < 13; x++ {
			img.Set(x, y, color.NRGBA{R: 200, G: 100, B: 50, A: 0xFF})
		}
	}

	mask := image.NewGray(image.Rect(0, 0, 3, 4)) // a row shorter than img
	mask.SetGray(0, 0, color.Gray{Y: 0})
	mask.SetGray(1, 0, color.Gray{Y: 128})
	mask.SetGray(2, 0, color.Gray{Y: 255})

	pauses := 0
	got := withMask(img, mask, 2, func() { pauses++ })

	if got.Rect != image.Rect(0, 0, 3, 5) {
		t.Fatalf("bounds %v, want 3×5 at the origin", got.Rect)
	}

	for _, tt := range []struct {
		x, y int
		want color.RGBA
	}{
		{0, 0, color.RGBA{}},                             // masked out
		{1, 0, color.RGBA{R: 100, G: 50, B: 25, A: 128}}, // half, premultiplied
		{2, 0, color.RGBA{R: 200, G: 100, B: 50, A: 255}},
		{0, 1, color.RGBA{}},                              // mask black
		{1, 4, color.RGBA{R: 200, G: 100, B: 50, A: 255}}, // below the mask: opaque
	} {
		if c := got.RGBAAt(tt.x, tt.y); c != tt.want {
			t.Errorf("(%d,%d) = %v, want %v", tt.x, tt.y, c, tt.want)
		}
	}

	if pauses != 3 { // rows 0-1, 2-3, 4
		t.Errorf("%d pauses for 5 rows in bands of 2, want 3", pauses)
	}
}

func TestWithMaskConvertsOtherMasks(t *testing.T) {
	t.Parallel()

	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.White)

	mask := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	mask.Set(0, 0, color.NRGBA{R: 255, G: 255, B: 255, A: 255})

	if c := withMask(img, mask, 8, func() {}).RGBAAt(0, 0); c.A != 255 || c.R != 255 {
		t.Errorf("white NRGBA mask: %v, want opaque white", c)
	}
}

func TestToRGBA(t *testing.T) {
	t.Parallel()

	src := image.NewNRGBA(image.Rect(5, 5, 7, 6))
	src.Set(5, 5, color.NRGBA{R: 255, A: 128})

	got := toRGBA(src)
	if got.Rect != image.Rect(0, 0, 2, 1) {
		t.Fatalf("bounds %v, want 2×1 at the origin", got.Rect)
	}

	if c := got.RGBAAt(0, 0); c != (color.RGBA{R: 128, A: 128}) {
		t.Errorf("premultiplied %v, want {128 0 0 128}", c)
	}

	same := image.NewRGBA(image.Rect(0, 0, 1, 1))
	if toRGBA(same) != same {
		t.Error("an RGBA image at the origin should be used as it is")
	}
}
