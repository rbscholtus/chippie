package crt

import (
	"image"
	"testing"
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
