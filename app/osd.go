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
)

var osdColour = color.RGBA{R: 0x4C, G: 0xFF, B: 0x6A, A: 0xFF} //nolint:gochecknoglobals // read-only colour

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

	if len(o.lines) == 0 {
		return
	}

	b := o.img.Bounds()
	x := float64(b.Dx()) * osdInset
	y := float64(b.Dy()) * osdInset
	lineH := o.face.Size * lineSpacing

	op := &text.DrawOptions{}

	for _, line := range o.lines {
		op.GeoM.Reset()
		op.GeoM.Translate(x, y)
		op.ColorScale.Reset()
		op.ColorScale.ScaleWithColor(osdColour)
		text.Draw(o.img, line, o.face, op)

		y += lineH
	}

	if o.bar >= 0 {
		step := o.face.Size * 0.6
		w := float32(step * (1 - osdBarSpacing))
		h := float32(o.face.Size * 0.8)

		for i := range osdBarSteps {
			c := osdColour
			if float64(i) >= o.bar*osdBarSteps-0.01 {
				c.A = 0x40 // unlit
			}

			vector.FillRect(o.img, float32(x+float64(i)*step), float32(y), w, h, c, false)
		}
	}
}
