package app

import (
	"bytes"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/gomonobold"

	"github.com/rbscholtus/chippie/app/crt"
)

// The on-screen display: green text in the picture's top-left corner, as
// period sets showed the channel (README §11). It is drawn into an image
// the size of the tube, which the renderer puts through the CRT effect.
const (
	osdTicks      = 120  // two seconds
	osdTextShare  = 0.08 // of the tube's height: the text size
	osdInset      = 0.09 // of the tube's size: from the corner
	osdBarSteps   = 10
	osdBarSpacing = 0.25 // of a bar step, between bars
	osdBarStep    = 0.6  // of the text size: one bar and its gap
	osdBarHeight  = 0.8  // of the text size
	osdPanelPad   = 0.35 // of the text size, around the panel's contents
	osdUnlit      = 0x50 // opacity of an unlit bar
)

// The display's colours: green text and bars on a nearly opaque dark panel
// (pre-multiplied, as color.RGBA requires).
var (
	osdColour = color.RGBA{R: 0x4C, G: 0xFF, B: 0x6A, A: 0xFF} //nolint:gochecknoglobals // read-only colour
	osdPanel  = color.RGBA{R: 0x04, G: 0x08, B: 0x05, A: 0xE6} //nolint:gochecknoglobals // read-only colour
)

type osd struct {
	renderer *crt.Renderer
	source   *text.GoTextFaceSource
	face     *text.GoTextFace
	img      *ebiten.Image

	lines []string
	bar   float64 // with a bar: its value, 0 to 1; otherwise negative
	left  int     // ticks until it disappears
	dirty bool
}

func newOSD(r *crt.Renderer) (*osd, error) {
	src, err := text.NewGoTextFaceSource(bytes.NewReader(gomonobold.TTF))
	if err != nil {
		return nil, err //nolint:wrapcheck // only fails on a broken embedded font
	}

	return &osd{renderer: r, source: src, bar: -1}, nil
}

// resize makes the image the tube's size.
func (o *osd) resize(tube image.Rectangle) {
	if tube.Empty() || (o.img != nil && o.img.Bounds().Size() == tube.Size()) {
		return
	}

	if o.img != nil {
		o.img.Deallocate()
	}

	o.img = ebiten.NewImage(tube.Dx(), tube.Dy())
	o.face = &text.GoTextFace{Source: o.source, Size: float64(tube.Dy()) * osdTextShare}
	o.renderer.SetOverlay(o.img)
	o.dirty = true
}

// show displays lines for a while.
func (o *osd) show(lines ...string) {
	o.lines, o.bar, o.left, o.dirty = lines, -1, osdTicks, true
}

// showBar displays a label with a bar of value (0 to 1).
func (o *osd) showBar(label string, value float64) {
	o.lines, o.bar, o.left, o.dirty = []string{label}, value, osdTicks, true
}

// clear removes the display at once.
func (o *osd) clear() {
	if len(o.lines) > 0 {
		o.lines, o.left, o.dirty = nil, 0, true
	}
}

// update counts down and redraws the image when it changed.
func (o *osd) update() {
	if o.left > 0 {
		o.left--
		if o.left == 0 {
			o.lines, o.dirty = nil, true
		}
	}

	if !o.dirty || o.img == nil {
		return
	}

	o.dirty = false
	o.img.Clear()
	o.renderer.Redraw()

	if len(o.lines) > 0 {
		o.paint()
	}
}

// paint draws the lines, and the bar if there is one, on a dark panel in
// the picture's top-left corner.
func (o *osd) paint() {
	b := o.img.Bounds()
	x := float64(b.Dx()) * osdInset
	y := float64(b.Dy()) * osdInset
	lineH := o.face.Size * lineSpacing
	step := o.face.Size * osdBarStep

	// A dark panel behind the text and bar, so the game's pixels don't show
	// through, as period sets blanked the area behind their on-screen text.
	w := 0.0

	for _, line := range o.lines {
		lw, _ := text.Measure(line, o.face, 0)
		w = max(w, lw)
	}

	h := lineH * float64(len(o.lines))

	if o.bar >= 0 {
		w = max(w, step*osdBarSteps)
		h += o.face.Size * osdBarHeight
	}

	pad := o.face.Size * osdPanelPad
	vector.FillRect(o.img, float32(x-pad), float32(y-pad), float32(w+2*pad), float32(h+2*pad), osdPanel, false)

	op := &text.DrawOptions{}

	for _, line := range o.lines {
		op.GeoM.Reset()
		op.GeoM.Translate(x, y)
		op.ColorScale.Reset()
		op.ColorScale.ScaleWithColor(osdColour)
		text.Draw(o.img, line, o.face, op)

		y += lineH
	}

	if o.bar < 0 {
		return
	}

	bw := float32(step * (1 - osdBarSpacing))
	bh := float32(o.face.Size * osdBarHeight)

	for i := range osdBarSteps {
		c := osdColour
		if !barLit(i, o.bar) {
			c = dim(osdColour, osdUnlit)
		}

		vector.FillRect(o.img, float32(x+float64(i)*step), float32(y), bw, bh, c, false)
	}
}

// barLit reports whether bar i of osdBarSteps is lit at value (0 to 1).
func barLit(i int, value float64) bool { return float64(i) < value*osdBarSteps-0.01 }

// dim returns c at opacity a (0–255) as a valid colour: color.RGBA is
// pre-multiplied, so the red, green and blue scale with the opacity.
// Lowering only A would leave an invalid colour that draws bright.
func dim(c color.RGBA, a uint8) color.RGBA {
	scale := func(v uint8) uint8 { return uint8(uint16(v) * uint16(a) / 0xFF) } //nolint:gosec // G115: at most 255

	return color.RGBA{R: scale(c.R), G: scale(c.G), B: scale(c.B), A: a}
}
