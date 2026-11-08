package chip8

// display holds the screen as one byte per pixel. Each byte holds the plane
// bits of that pixel (bit 0 = plane 1, bit 1 = plane 2 on XO-CHIP), so a
// value of 0 is off. The buffer is sized for the platform's largest
// resolution; width and height are the resolution in use.
type display struct {
	pix    []byte
	width  int
	height int
}

// reset sizes the buffer for maxW×maxH, reusing it when the size is
// unchanged, sets the resolution to w×h and clears it.
func (d *display) reset(maxW, maxH, w, h int) {
	if n := maxW * maxH; len(d.pix) == n {
		clear(d.pix)
	} else {
		d.pix = make([]byte, n)
	}

	d.width, d.height = w, h
}

// clear turns every pixel off.
func (d *display) clear() { clear(d.pix[:d.width*d.height]) }

// view returns a read-only view of the pixels in use.
func (d *display) view() DisplayView {
	return DisplayView{pix: d.pix[:d.width*d.height], width: d.width, height: d.height}
}

// drawSprite XORs an 8-pixel-wide sprite of len(rows) rows onto the screen at
// (x, y), on plane 1. The start position always wraps; pixels beyond the
// edge wrap or are clipped depending on the wrap quirk. It reports whether
// any pixel was turned off (a collision).
func (d *display) drawSprite(x, y int, rows []byte, wrap bool) bool {
	const spriteWidth = 8

	x %= d.width
	y %= d.height
	collided := false

	for r, bits := range rows {
		py := y + r
		if py >= d.height {
			if !wrap {
				break
			}

			py %= d.height
		}

		for c := range spriteWidth {
			if bits&(0x80>>c) == 0 {
				continue
			}

			px := x + c
			if px >= d.width {
				if !wrap {
					break
				}

				px %= d.width
			}

			p := &d.pix[py*d.width+px]
			if *p&1 != 0 {
				collided = true
			}

			*p ^= 1
		}
	}

	return collided
}

// DisplayView is a read-only view of the display.
type DisplayView struct {
	pix    []byte
	width  int
	height int
}

// Width returns the display width in pixels.
func (v DisplayView) Width() int { return v.width }

// Height returns the display height in pixels.
func (v DisplayView) Height() int { return v.height }

// Pixel returns the plane bits of the pixel at (x, y): 0 is off, 1 is plane
// 1, 2 is plane 2 and 3 is both. Coordinates outside the display return 0.
func (v DisplayView) Pixel(x, y int) byte {
	if x < 0 || y < 0 || x >= v.width || y >= v.height {
		return 0
	}

	return v.pix[y*v.width+x]
}
