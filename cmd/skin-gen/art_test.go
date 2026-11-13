package main

import (
	"image"
	"image/color"
	"testing"

	"golang.org/x/image/font/gofont/gomedium"
)

var (
	grey  = color.NRGBA{R: 200, G: 200, B: 200, A: 0xFF}
	black = color.NRGBA{R: 5, G: 5, B: 5, A: 0xFF}
	white = color.NRGBA{R: 250, G: 250, B: 250, A: 0xFF}
)

func filled(w, h int, c color.NRGBA) *image.NRGBA {
	m := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(m.Pix); i += 4 {
		m.Pix[i], m.Pix[i+1], m.Pix[i+2], m.Pix[i+3] = c.R, c.G, c.B, c.A
	}

	return m
}

func fillRect(m *image.NRGBA, r image.Rectangle, c color.NRGBA) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			m.SetNRGBA(x, y, c)
		}
	}
}

func TestPaintOut(t *testing.T) {
	t.Parallel()

	m := filled(20, 10, grey)
	fillRect(m, image.Rect(8, 2, 12, 6), white) // the mark

	paintOut(m, image.Rect(7, 1, 13, 7))

	for y := range 10 {
		for x := range 20 {
			if c := m.NRGBAAt(x, y); c != grey {
				t.Fatalf("(%d,%d) = %v, want the background", x, y, c)
			}
		}
	}
}

func TestTubeMask(t *testing.T) {
	t.Parallel()

	m := filled(60, 40, grey)
	fillRect(m, image.Rect(10, 8, 50, 32), black) // the tube

	mask, box := tubeMask(m, image.Pt(30, 20))

	if box != image.Rect(10, 8, 50, 32) {
		t.Errorf("box = %v", box)
	}

	if y := mask.GrayAt(30, 20).Y; y != 0 {
		t.Errorf("centre = %d, want 0 (hole)", y)
	}

	if y := mask.GrayAt(10, 8).Y; y != 0xFF {
		t.Errorf("tube corner = %d, want 255: the hole is shrunk so no bezel shows", y)
	}

	if y := mask.GrayAt(5, 5).Y; y != 0xFF {
		t.Errorf("cabinet = %d, want 255", y)
	}
}

func TestFadeEdges(t *testing.T) {
	t.Parallel()

	m := filled(40, 40, black)
	fadeEdges(m, white, 10)

	if c := m.NRGBAAt(0, 20); c != white {
		t.Errorf("edge = %v, want the fill", c)
	}

	if c := m.NRGBAAt(20, 20); c != black {
		t.Errorf("middle = %v, want unchanged", c)
	}

	if c := m.NRGBAAt(5, 20); c.R <= black.R || c.R >= white.R {
		t.Errorf("halfway = %v, want a blend", c)
	}

	if c := borderColour(filled(5, 5, grey)); c != grey {
		t.Errorf("borderColour = %v", c)
	}
}

func TestRoundSprite(t *testing.T) {
	t.Parallel()

	s := roundSprite(filled(100, 100, grey), 50, 50, 20)

	if s.Bounds() != image.Rect(0, 0, 40, 40) {
		t.Fatalf("bounds = %v", s.Bounds())
	}

	if a := s.NRGBAAt(20, 20).A; a != 0xFF {
		t.Errorf("centre alpha = %d", a)
	}

	if a := s.NRGBAAt(0, 0).A; a != 0 {
		t.Errorf("corner alpha = %d, want transparent", a)
	}
}

func TestEraseRing(t *testing.T) {
	t.Parallel()

	s := filled(100, 100, black)
	fillRect(s, image.Rect(48, 10, 52, 16), white) // print in the band (r 34–40)
	fillRect(s, image.Rect(48, 80, 52, 86), white) // print inside a kept area

	eraseRing(s, 30, 45, []image.Rectangle{image.Rect(-5, 28, 5, 40)})

	if c := s.NRGBAAt(50, 12); c.R > black.R+5 {
		t.Errorf("print not erased: %v", c)
	}

	if c := s.NRGBAAt(50, 82); c != white {
		t.Errorf("kept area changed: %v", c)
	}
}

func TestDrawLabels(t *testing.T) {
	t.Parallel()

	face, err := newFace(gomedium.TTF, 12)
	if err != nil {
		t.Fatal(err)
	}

	s := filled(100, 100, black)
	drawLabels(s, []string{"U", "6"}, 40, face, white)

	lit := func(r image.Rectangle) bool {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				if s.NRGBAAt(x, y).R > 128 {
					return true
				}
			}
		}

		return false
	}

	if !lit(image.Rect(40, 0, 60, 20)) {
		t.Error("first label not at the top")
	}

	if !lit(image.Rect(40, 80, 60, 100)) {
		t.Error("second of two labels not at the bottom")
	}

	if lit(image.Rect(0, 40, 20, 60)) {
		t.Error("ink on the left, where no label goes")
	}
}
