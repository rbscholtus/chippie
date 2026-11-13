package main

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/math/f64"

	"github.com/rbscholtus/chippie/app/skin"
)

// Preview colours: the tube hole and the control outlines.
var (
	holeColour    = color.NRGBA{R: 0x20, G: 0x90, B: 0x40, A: 0xFF} //nolint:gochecknoglobals // read-only colour
	outlineColour = color.NRGBA{R: 0xFF, G: 0x20, B: 0x20, A: 0xFF} //nolint:gochecknoglobals // read-only colour
)

// writePreview composes the @3x outputs as the app would draw them, for
// checking alignment by eye: the tube hole shows green, every round control
// is turned by angle degrees, and each control and the screen are outlined.
func writePreview(path string, out *outputs, angle float64) error {
	m := toNRGBA(out.cabinet)
	sk, mask, sprites := &out.skin, out.mask, out.sprites

	for y := range m.Bounds().Dy() {
		for x := range m.Bounds().Dx() {
			t := 1 - float64(mask.GrayAt(x, y).Y)/0xFF
			m.SetNRGBA(x, y, mix(m.NRGBAAt(x, y), holeColour, t))
		}
	}

	for _, c := range sk.Controls {
		s := sprites[c.ID]
		b := c.Bounds()

		if c.Round() {
			sin, cos := math.Sincos(angle * math.Pi / 180)
			w, h := float64(s.Bounds().Dx()), float64(s.Bounds().Dy())
			mtx := f64.Aff3{
				cos, -sin, c.CX - cos*w/2 + sin*h/2,
				sin, cos, c.CY - sin*w/2 - cos*h/2,
			}
			xdraw.BiLinear.Transform(m, mtx, s, s.Bounds(), xdraw.Over, nil)
		} else {
			draw.Draw(m, image.Rect(int(b.X), int(b.Y), int(b.X+b.W), int(b.Y+b.H)), s, image.Point{}, draw.Over)
		}

		outline(m, b)
	}

	outline(m, sk.Screen)

	return writePNG(path, m)
}

func outline(m *image.NRGBA, r skin.Rect) {
	x0, y0, x1, y1 := int(r.X), int(r.Y), int(r.X+r.W)-1, int(r.Y+r.H)-1
	for x := x0; x <= x1; x++ {
		m.SetNRGBA(x, y0, outlineColour)
		m.SetNRGBA(x, y1, outlineColour)
	}

	for y := y0; y <= y1; y++ {
		m.SetNRGBA(x0, y, outlineColour)
		m.SetNRGBA(x1, y, outlineColour)
	}
}
