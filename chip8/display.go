package chip8

// legacyLoresScale is how many screen pixels wide and high a lo-res pixel is
// on legacy SUPER-CHIP.
const legacyLoresScale = 2

// display holds the screen as one byte per pixel. Each byte holds the plane
// bits of that pixel (bit 0 = plane 1, bit 1 = plane 2 on XO-CHIP), so a
// value of 0 is off. The buffer is sized for the platform's largest
// resolution; width and height are the part in use.
//
// Sprites are placed in logical pixels, each scale×scale screen pixels.
// Scale is 1, except in lo-res on legacy SUPER-CHIP, where the whole 128×64
// screen stays in use and scrolling can move half a lo-res pixel.
type display struct {
	pix    []byte
	width  int
	height int
	scale  int
}

// reset sizes the buffer for maxW×maxH, reusing it when the size is
// unchanged, sets the resolution to w×h and the scale, and clears it.
func (d *display) reset(maxW, maxH, w, h, scale int) {
	if n := maxW * maxH; len(d.pix) == n {
		clear(d.pix)
	} else {
		d.pix = make([]byte, n)
	}

	d.width, d.height, d.scale = w, h, scale
}

// setResolution changes the resolution and clears the whole screen.
func (d *display) setResolution(w, h int) {
	clear(d.pix)
	d.width, d.height = w, h
}

// clear turns off the given planes of every pixel.
func (d *display) clear(planes byte) {
	for i := range d.pix[:d.width*d.height] {
		d.pix[i] &^= planes
	}
}

// view returns a read-only view of the pixels in use.
func (d *display) view() DisplayView {
	return DisplayView{pix: d.pix[:d.width*d.height], width: d.width, height: d.height}
}

// drawSprite XORs a sprite onto the given planes at (x, y), in logical
// pixels. rows holds its rows, top bit first, and width is 8 or 16. The
// start position always wraps; pixels beyond the edge wrap or are clipped
// depending on the wrap quirk. It returns how many rows turned a pixel off
// (collided) and how many were clipped at the bottom edge.
func (d *display) drawSprite(x, y int, rows []uint16, width int, planes byte, wrap bool) (int, int) {
	w, h := d.width/d.scale, d.height/d.scale
	x %= w
	y %= h
	hit := 0

	for r, bits := range rows {
		py := y + r
		if py >= h {
			if !wrap {
				return hit, len(rows) - r
			}

			py %= h
		}

		if d.drawRow(x, py, bits, width, w, planes, wrap) {
			hit++
		}
	}

	return hit, 0
}

// drawRow draws one sprite row and reports whether it turned a pixel off.
// w is the logical screen width.
func (d *display) drawRow(x, y int, bits uint16, width, w int, planes byte, wrap bool) bool {
	collided := false

	for c := range width {
		if bits&(0x8000>>c) == 0 {
			continue
		}

		px := x + c
		if px >= w {
			if !wrap {
				break
			}

			px %= w
		}

		if d.flip(px, y, planes) {
			collided = true
		}
	}

	return collided
}

// flip XORs the planes of one logical pixel and reports whether a lit plane
// was turned off.
func (d *display) flip(x, y int, planes byte) bool {
	if d.scale == 1 {
		p := &d.pix[y*d.width+x]
		off := *p&planes != 0
		*p ^= planes

		return off
	}

	off := false

	for sy := y * d.scale; sy < (y+1)*d.scale; sy++ {
		for sx := x * d.scale; sx < (x+1)*d.scale; sx++ {
			p := &d.pix[sy*d.width+sx]
			off = off || *p&planes != 0
			*p ^= planes
		}
	}

	return off
}

// scroll moves the given planes by (dx, dy) screen pixels. Pixels that
// scroll in are off. Other planes stay where they are.
func (d *display) scroll(dx, dy int, planes byte) {
	// Walk away from the direction of travel, so that each source pixel is
	// read before it is overwritten.
	for row := range d.height {
		y := row
		if dy > 0 {
			y = d.height - 1 - row
		}

		for col := range d.width {
			x := col
			if dx > 0 {
				x = d.width - 1 - col
			}

			p := &d.pix[y*d.width+x]
			*p = *p&^planes | d.at(x-dx, y-dy)&planes
		}
	}
}

// at returns the pixel at (x, y) in screen pixels, 0 outside the screen.
func (d *display) at(x, y int) byte {
	if x < 0 || y < 0 || x >= d.width || y >= d.height {
		return 0
	}

	return d.pix[y*d.width+x]
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
