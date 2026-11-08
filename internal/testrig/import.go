package testrig

import (
	"errors"
	"fmt"
	"image"
	"image/color"
)

// Errors from importing reference pictures.
var (
	ErrNotScaled     = errors.New("testrig: picture is not a whole multiple of the frame size")
	ErrUnknownColour = errors.New("testrig: colour not in palette")
	ErrMixedBlock    = errors.New("testrig: block is not a single colour")
)

// Palette maps a picture's colours to plane bits. Several colours may map
// to the same value (some pictures use two shades for "off").
type Palette map[color.RGBA]byte

// ImportPicture converts a scaled-up reference picture into a w×h frame.
// The picture must be an exact multiple of w×h, every colour must be in the
// palette, and every block must map to a single value, so the conversion
// is lossless.
func ImportPicture(img image.Image, w, h int, pal Palette) (Frame, error) {
	b := img.Bounds()
	if w <= 0 || h <= 0 || b.Dx()%w != 0 || b.Dy()%h != 0 || b.Dx()/w != b.Dy()/h {
		return Frame{}, fmt.Errorf("%w: %d×%d into %d×%d", ErrNotScaled, b.Dx(), b.Dy(), w, h)
	}

	scale := b.Dx() / w
	f := Frame{Width: w, Height: h, Pix: make([]byte, w*h)}

	for fy := range h {
		for fx := range w {
			v, err := blockValue(img, b.Min.X+fx*scale, b.Min.Y+fy*scale, scale, pal)
			if err != nil {
				return Frame{}, fmt.Errorf("block (%d,%d): %w", fx, fy, err)
			}

			f.Pix[fy*w+fx] = v
		}
	}

	return f, nil
}

// blockValue returns the palette value of the scale×scale block at (x0, y0).
func blockValue(img image.Image, x0, y0, scale int, pal Palette) (byte, error) {
	var first byte

	for y := y0; y < y0+scale; y++ {
		for x := x0; x < x0+scale; x++ {
			c := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA) //nolint:forcetypeassert // RGBAModel always returns RGBA

			v, ok := pal[c]
			if !ok {
				return 0, fmt.Errorf("%w: %v at (%d,%d)", ErrUnknownColour, c, x, y)
			}

			if x == x0 && y == y0 {
				first = v
			} else if v != first {
				return 0, ErrMixedBlock
			}
		}
	}

	return first, nil
}
