package crt

import (
	"image/color"

	"github.com/rbscholtus/chippie/chip8"
)

// Palette gives the colour of each pixel value: index 0 is off, 1 is plane
// 1, 2 is plane 2 and 3 is both planes (XO-CHIP).
type Palette [4]color.RGBA

// DefaultPalette is white on black; the CRT shader tints it.
var DefaultPalette = Palette{ //nolint:gochecknoglobals // read-only default
	{R: 0x00, G: 0x00, B: 0x00, A: 0xFF},
	{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF},
	{R: 0xAA, G: 0xAA, B: 0xAA, A: 0xFF},
	{R: 0x55, G: 0x55, B: 0x55, A: 0xFF},
}

// bytesPerPixel is RGBA.
const bytesPerPixel = 4

// CopyFramebuffer writes the display into pix as RGBA, for WritePixels. pix
// must hold at least width×height×4 bytes; it returns false if it doesn't.
func CopyFramebuffer(v chip8.DisplayView, pal *Palette, pix []byte) bool {
	w, h := v.Width(), v.Height()
	if len(pix) < w*h*bytesPerPixel {
		return false
	}

	i := 0

	for y := range h {
		for x := range w {
			c := pal[v.Pixel(x, y)&3]
			pix[i], pix[i+1], pix[i+2], pix[i+3] = c.R, c.G, c.B, c.A
			i += bytesPerPixel
		}
	}

	return true
}
