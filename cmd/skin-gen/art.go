package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"slices"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/f64"
	"golang.org/x/image/math/fixed"
)

// This file holds the image operations, kept free of file I/O so they can
// be tested on small synthetic images.

// Image thresholds: luminance (0–255) and distances in pixels.
const (
	tubeDark     = 40  // the switched-off picture tube is at most this bright
	maskErode    = 2   // pixels the tube mask is shrunk by, so no bezel shows
	ringBlend    = 2.0 // pixels over which an erased ring blends into its edges
	backdropBlur = 2   // backdrop pixels of blur radius
)

func lum(c color.NRGBA) int { return (int(c.R)*3 + int(c.G)*6 + int(c.B)) / 10 }

// toNRGBA copies any image into a fresh NRGBA starting at (0, 0).
func toNRGBA(src image.Image) *image.NRGBA {
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)

	return dst
}

// paintOut fills r with a horizontal blend of the columns just left and
// right of it: enough to remove a small mark from a smooth background.
func paintOut(m *image.NRGBA, r image.Rectangle) {
	r = r.Intersect(m.Bounds())
	if r.Empty() {
		return
	}

	left, right := max(r.Min.X-1, 0), min(r.Max.X, m.Bounds().Max.X-1)

	for y := r.Min.Y; y < r.Max.Y; y++ {
		a, b := m.NRGBAAt(left, y), m.NRGBAAt(right, y)

		for x := r.Min.X; x < r.Max.X; x++ {
			t := float64(x-left) / float64(right-left)
			m.SetNRGBA(x, y, mix(a, b, t))
		}
	}
}

func mix(a, b color.NRGBA, t float64) color.NRGBA {
	f := func(x, y uint8) uint8 { return uint8(math.Round(float64(x)*(1-t) + float64(y)*t)) }

	return color.NRGBA{R: f(a.R, b.R), G: f(a.G, b.G), B: f(a.B, b.B), A: f(a.A, b.A)}
}

// tubeMask finds the dark picture tube by flooding from seed over pixels no
// brighter than tubeDark, then fills any holes in it (highlights on the
// glass). It returns a greyscale mask the size of m, black in the (slightly
// shrunk) tube with a one-pixel soft edge, white elsewhere, and the tube's
// bounding box.
func tubeMask(m *image.NRGBA, seed image.Point) (*image.Gray, image.Rectangle) {
	b := m.Bounds()
	in := make([]bool, b.Dx()*b.Dy())
	idx := func(p image.Point) int { return (p.Y-b.Min.Y)*b.Dx() + p.X - b.Min.X }
	box := image.Rectangle{Min: seed, Max: seed}

	stack := []image.Point{seed}
	for len(stack) > 0 {
		p := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if !p.In(b) || in[idx(p)] || lum(m.NRGBAAt(p.X, p.Y)) > tubeDark {
			continue
		}

		in[idx(p)] = true
		box = box.Union(image.Rectangle{Min: p, Max: p.Add(image.Pt(1, 1))})
		stack = append(stack, p.Add(image.Pt(1, 0)), p.Add(image.Pt(-1, 0)), p.Add(image.Pt(0, 1)), p.Add(image.Pt(0, -1)))
	}

	in = fillHoles(in, b.Dx(), b.Dy())

	for range maskErode {
		in = erode(in, b.Dx(), b.Dy())
	}

	mask := image.NewGray(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			// Soft edge: the share of the 3×3 neighbourhood outside the tube.
			out := 0

			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					p := image.Pt(x+dx, y+dy)
					if !p.In(b) || !in[idx(p)] {
						out++
					}
				}
			}

			mask.SetGray(x, y, color.Gray{Y: uint8(out * 255 / 9)})
		}
	}

	return mask, box
}

// fillHoles sets every unset cell that the unset cells along the edges
// can't reach: the holes inside a region.
func fillHoles(in []bool, w, h int) []bool {
	outside := make([]bool, len(in))

	var stack []int

	for x := range w {
		stack = append(stack, x, (h-1)*w+x)
	}

	for y := range h {
		stack = append(stack, y*w, y*w+w-1)
	}

	for len(stack) > 0 {
		i := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if in[i] || outside[i] {
			continue
		}

		outside[i] = true
		x, y := i%w, i/w

		if x > 0 {
			stack = append(stack, i-1)
		}

		if x < w-1 {
			stack = append(stack, i+1)
		}

		if y > 0 {
			stack = append(stack, i-w)
		}

		if y < h-1 {
			stack = append(stack, i+w)
		}
	}

	out := make([]bool, len(in))
	for i := range in {
		out[i] = in[i] || !outside[i]
	}

	return out
}

// erode removes every set cell that has an unset 4-neighbour.
func erode(in []bool, w, h int) []bool {
	out := slices.Clone(in)

	for y := range h {
		for x := range w {
			i := y*w + x
			if !in[i] {
				continue
			}

			if x == 0 || y == 0 || x == w-1 || y == h-1 || !in[i-1] || !in[i+1] || !in[i-w] || !in[i+w] {
				out[i] = false
			}
		}
	}

	return out
}

// fadeMask makes the outer feather pixels of a mask fade to transparent
// (smoothly), so the cabinet's cut edges disappear into the backdrop.
func fadeMask(m *image.Gray, feather int) {
	b := m.Bounds()

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			d := min(x-b.Min.X, y-b.Min.Y, b.Max.X-1-x, b.Max.Y-1-y)
			if d >= feather {
				continue
			}

			t := float64(d) / float64(feather)
			t = t * t * (3 - 2*t) // smoothstep
			m.SetGray(x, y, color.Gray{Y: uint8(math.Round(float64(m.GrayAt(x, y).Y) * t))})
		}
	}
}

// makeBackdrop is a small, blurred picture of the studio behind the TV: the
// master image around crop, extended by pad on every side (beyond the
// master, its edge pixels continue), shrunk by shrink, and fading into fill
// towards its outer edges. The app draws it behind the cabinet, so the
// cabinet's faded edges meet the background they were cut from.
func makeBackdrop(master *image.NRGBA, crop image.Rectangle, pad, shrink int, fill color.NRGBA) *image.NRGBA {
	area := crop.Inset(-pad)
	w, h := area.Dx()/shrink, area.Dy()/shrink
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	mb := master.Bounds()

	for y := range h {
		for x := range w {
			var r, g, bl, n int

			for dy := range shrink {
				for dx := range shrink {
					px := min(max(area.Min.X+x*shrink+dx, mb.Min.X), mb.Max.X-1)
					py := min(max(area.Min.Y+y*shrink+dy, mb.Min.Y), mb.Max.Y-1)
					c := master.NRGBAAt(px, py)
					r, g, bl, n = r+int(c.R), g+int(c.G), bl+int(c.B), n+1
				}
			}

			avg := func(sum int) uint8 { return uint8(sum / n) } //nolint:gosec // G115: an average of bytes
			c := color.NRGBA{R: avg(r), G: avg(g), B: avg(bl), A: 0xFF}

			// Distance outside the crop, as a share of the padding.
			cx, cy := area.Min.X+x*shrink+shrink/2, area.Min.Y+y*shrink+shrink/2
			dx := max(crop.Min.X-cx, cx-crop.Max.X, 0)
			dy := max(crop.Min.Y-cy, cy-crop.Max.Y, 0)
			t := min(math.Hypot(float64(dx), float64(dy))/float64(pad), 1)
			out.SetNRGBA(x, y, mix(c, fill, t*t*(3-2*t)))
		}
	}

	return boxBlur(out, backdropBlur)
}

// boxBlur blurs m with a (2r+1)² box, twice (close to a Gaussian).
func boxBlur(m *image.NRGBA, r int) *image.NRGBA {
	b := m.Bounds()

	for range 2 {
		next := image.NewNRGBA(b)

		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				var sr, sg, sb, n int

				for dy := -r; dy <= r; dy++ {
					for dx := -r; dx <= r; dx++ {
						p := image.Pt(x+dx, y+dy)
						if !p.In(b) {
							continue
						}

						c := m.NRGBAAt(p.X, p.Y)
						sr, sg, sb, n = sr+int(c.R), sg+int(c.G), sb+int(c.B), n+1
					}
				}

				avg := func(sum int) uint8 { return uint8(sum / n) } //nolint:gosec // G115: an average of bytes
				next.SetNRGBA(x, y, color.NRGBA{R: avg(sr), G: avg(sg), B: avg(sb), A: 0xFF})
			}
		}

		m = next
	}

	return m
}

// borderColour is the average colour of m's outermost pixels.
func borderColour(m *image.NRGBA) color.NRGBA {
	b := m.Bounds()

	var r, g, bl, n int

	add := func(x, y int) {
		c := m.NRGBAAt(x, y)
		r, g, bl, n = r+int(c.R), g+int(c.G), bl+int(c.B), n+1
	}

	for x := b.Min.X; x < b.Max.X; x++ {
		add(x, b.Min.Y)
		add(x, b.Max.Y-1)
	}

	for y := b.Min.Y; y < b.Max.Y; y++ {
		add(b.Min.X, y)
		add(b.Max.X-1, y)
	}

	avg := func(sum int) uint8 { return uint8(sum / n) } //nolint:gosec // G115: an average of bytes

	return color.NRGBA{R: avg(r), G: avg(g), B: avg(bl), A: 0xFF}
}

// roundSprite cuts a 2r×2r square centred on (cx, cy) out of m, transparent
// outside the circle of radius r, with an anti-aliased edge.
func roundSprite(m *image.NRGBA, cx, cy, r int) *image.NRGBA {
	s := image.NewNRGBA(image.Rect(0, 0, 2*r, 2*r))

	for y := range 2 * r {
		for x := range 2 * r {
			d := math.Hypot(float64(x)+0.5-float64(r), float64(y)+0.5-float64(r))
			a := min(max(float64(r)-d, 0), 1)

			c := m.NRGBAAt(cx-r+x, cy-r+y)
			c.A = uint8(math.Round(float64(c.A) * a))
			s.SetNRGBA(x, y, c)
		}
	}

	return s
}

// eraseRing repaints the annulus inner..outer of a round sprite (centre in
// its middle) in the ring's own colour at each radius, removing the print
// and its shadows. The colour is the median of the pixels at that radius,
// which ignores print covering less than half the circle. The band's edges
// blend over ringBlend pixels so no seam shows. Pixels within keep
// (relative to the centre) are left alone, e.g. a knob's bar crossing the
// band.
func eraseRing(s *image.NRGBA, inner, outer float64, keep []image.Rectangle) {
	b := s.Bounds()
	cx, cy := float64(b.Dx())/2, float64(b.Dy())/2
	radius := func(x, y int) float64 { return math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy) }

	at := map[int][]color.NRGBA{}

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if r := radius(x, y); r >= inner && r <= outer {
				at[int(r)] = append(at[int(r)], s.NRGBAAt(x, y))
			}
		}
	}

	ring := make(map[int]color.NRGBA, len(at))

	for r, cs := range at {
		slices.SortFunc(cs, func(a, b color.NRGBA) int { return lum(a) - lum(b) })
		ring[r] = cs[len(cs)/2]
	}

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r := radius(x, y)
			c, ok := ring[int(r)]

			if r < inner || r > outer || !ok || inAny(image.Pt(x-int(cx), y-int(cy)), keep) {
				continue
			}

			p := s.NRGBAAt(x, y)
			c.A = p.A
			s.SetNRGBA(x, y, mix(p, c, min(min(r-inner, outer-r)/ringBlend, 1)))
		}
	}
}

func inAny(p image.Point, rs []image.Rectangle) bool {
	return slices.ContainsFunc(rs, p.In)
}

// drawLabels prints labels around a round sprite's centre at radius r,
// evenly spaced clockwise from the top, each rotated so its top points
// outward (as printed on a dial).
func drawLabels(s *image.NRGBA, labels []string, r float64, face font.Face, ink color.NRGBA) {
	cx, cy := float64(s.Bounds().Dx())/2, float64(s.Bounds().Dy())/2

	for i, label := range labels {
		glyphs := renderText(label, face, ink)
		gw, gh := float64(glyphs.Bounds().Dx()), float64(glyphs.Bounds().Dy())

		theta := float64(i) * 2 * math.Pi / float64(len(labels))
		sin, cos := math.Sincos(theta)
		px, py := cx+r*sin, cy-r*cos

		// Map the label's pixels: centre on the origin, rotate by theta,
		// move to (px, py).
		m := f64.Aff3{
			cos, -sin, px - cos*gw/2 + sin*gh/2,
			sin, cos, py - sin*gw/2 - cos*gh/2,
		}
		xdraw.BiLinear.Transform(s, m, glyphs, glyphs.Bounds(), xdraw.Over, nil)
	}
}

// renderText draws s in ink on a transparent image just big enough for it.
func renderText(s string, face font.Face, ink color.NRGBA) *image.NRGBA {
	met := face.Metrics()
	w := font.MeasureString(face, s).Ceil()
	h := (met.Ascent + met.Descent).Ceil()

	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	d := font.Drawer{Dst: img, Src: image.NewUniform(ink), Face: face, Dot: fixed.Point26_6{Y: met.Ascent}}
	d.DrawString(s)

	return img
}

// newFace loads a TrueType font at size pixels.
func newFace(ttf []byte, size float64) (*opentype.Face, error) {
	f, err := opentype.Parse(ttf)
	if err != nil {
		return nil, fmt.Errorf("parse font: %w", err)
	}

	face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return nil, fmt.Errorf("font face: %w", err)
	}

	of, ok := face.(*opentype.Face)
	if !ok {
		return nil, fmt.Errorf("font face: %w: unexpected type %T", errRecipe, face)
	}

	return of, nil
}

// scaleTo resizes src to w×h with Catmull-Rom filtering.
func scaleTo(src image.Image, w, h int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, max(w, 1), max(h, 1)))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Src, nil)

	return dst
}

// scaleGray resizes a mask.
func scaleGray(src *image.Gray, w, h int) *image.Gray {
	dst := image.NewGray(image.Rect(0, 0, max(w, 1), max(h, 1)))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Src, nil)

	return dst
}
