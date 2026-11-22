package app

import "image"

// pointer turns a press, moves and a release (mouse or one finger) into a
// tap or a vertical drag. A press that moves more than the threshold
// becomes a drag and is never a tap, so scrolling a list doesn't pick from
// it. It holds no Ebitengine state, so it can be tested directly.
type pointer struct {
	down      bool
	dragging  bool
	start     image.Point
	last      image.Point
	threshold int // pixels of movement before a press becomes a drag
}

// press starts tracking at pt.
func (p *pointer) press(pt image.Point) {
	p.down, p.dragging = true, false
	p.start, p.last = pt, pt
}

// move reports the vertical movement since the last call, once the press
// has become a drag; 0 before that.
func (p *pointer) move(pt image.Point) int {
	if !p.down {
		return 0
	}

	if !p.dragging {
		d := pt.Sub(p.start)
		if d.X*d.X+d.Y*d.Y <= p.threshold*p.threshold {
			return 0
		}

		p.dragging = true
	}

	dy := pt.Y - p.last.Y
	p.last = pt

	return dy
}

// release ends the press and reports whether it was a tap, and where.
func (p *pointer) release(pt image.Point) (image.Point, bool) {
	if !p.down {
		return image.Point{}, false
	}

	p.move(pt)
	tap := !p.dragging
	p.down, p.dragging = false, false

	return p.start, tap
}
