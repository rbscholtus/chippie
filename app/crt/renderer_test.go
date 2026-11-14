package crt

import (
	"image"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// TestShadersCompile compiles both Kage shaders. Compilation happens on the
// CPU, so no window or GPU is needed.
func TestShadersCompile(t *testing.T) {
	t.Parallel()

	r, err := New(ModeCRT)
	if err != nil {
		t.Fatal(err)
	}

	if r.crt == nil || r.phosphor == nil {
		t.Error("shaders not set")
	}

	if _, ok := r.crtOpts.Uniforms["Tint"]; !ok {
		t.Error("CRT uniforms not prepared")
	}
}

func TestSetPowerClamps(t *testing.T) {
	t.Parallel()

	r, err := New(ModeClean)
	if err != nil {
		t.Fatal(err)
	}

	for in, want := range map[float32]float32{-1: 0, 0.5: 0.5, 2: 1} {
		r.SetPower(in)

		if r.power != want {
			t.Errorf("SetPower(%v) → %v, want %v", in, r.power, want)
		}
	}
}

func TestSetScreenIgnoresEmpty(t *testing.T) {
	t.Parallel()

	r, err := New(ModeClean)
	if err != nil {
		t.Fatal(err)
	}

	want := image.Rect(10, 20, 138, 84)
	r.SetScreen(want)
	r.SetScreen(image.Rectangle{})

	if r.Screen() != want {
		t.Errorf("Screen() = %v, want %v", r.Screen(), want)
	}
}

func TestPictureFallsBackToScreen(t *testing.T) {
	t.Parallel()

	r, err := New(ModeClean)
	if err != nil {
		t.Fatal(err)
	}

	tube := image.Rect(0, 0, 200, 160)
	r.SetScreen(tube)

	if r.Picture() != tube {
		t.Errorf("no picture set: Picture() = %v, want the screen", r.Picture())
	}

	pic := image.Rect(10, 30, 190, 120)
	r.SetPicture(pic)

	if r.Picture() != pic {
		t.Errorf("Picture() = %v, want %v", r.Picture(), pic)
	}

	r.SetPicture(image.Rect(-10, 0, 100, 50)) // sticks out of the tube

	if r.Picture() != tube {
		t.Errorf("picture outside the tube: Picture() = %v, want the screen", r.Picture())
	}
}

// TestPlaceFrame checks where the frame's corners land in the picture for
// each rotation (clockwise).
func TestPlaceFrame(t *testing.T) {
	t.Parallel()

	const w, h = 64, 32

	pic := image.Rect(10, 20, 110, 220)
	tl, tr := image.Pt(pic.Min.X, pic.Min.Y), image.Pt(pic.Max.X, pic.Min.Y)
	bl, br := image.Pt(pic.Min.X, pic.Max.Y), image.Pt(pic.Max.X, pic.Max.Y)

	for _, tc := range []struct {
		rot                       int
		frameTL, frameTR, frameBL image.Point // where the frame's corners go
	}{
		{0, tl, tr, bl},
		{90, tr, br, tl},
		{180, br, bl, tr},
		{270, bl, tl, br},
	} {
		var m ebiten.GeoM
		PlaceFrame(&m, w, h, pic, tc.rot)

		for _, c := range []struct {
			x, y float64
			want image.Point
		}{{0, 0, tc.frameTL}, {w, 0, tc.frameTR}, {0, h, tc.frameBL}} {
			x, y := m.Apply(c.x, c.y)
			if math.Abs(x-float64(c.want.X)) > 1e-9 || math.Abs(y-float64(c.want.Y)) > 1e-9 {
				t.Errorf("rotation %d: frame (%v, %v) → (%.1f, %.1f), want %v", tc.rot, c.x, c.y, x, y, c.want)
			}
		}
	}
}
