package app

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// The remote control: a 4×4 hex keypad for touch screens, in the COSMAC
// VIP's layout, drawn in code (README §13). Every finger counts, so two
// keys can be held at once.

// remoteKeys is the keypad, row by row: the hex key at each position.
var remoteKeys = [4][4]uint8{ //nolint:gochecknoglobals // read-only table
	{0x1, 0x2, 0x3, 0xC},
	{0x4, 0x5, 0x6, 0xD},
	{0x7, 0x8, 0x9, 0xE},
	{0xA, 0x0, 0xB, 0xF},
}

// Remote colours.
var (
	remoteBody    = color.RGBA{R: 0x24, G: 0x20, B: 0x1C, A: 0xFF} //nolint:gochecknoglobals // read-only colour
	remoteKey     = color.RGBA{R: 0x3A, G: 0x34, B: 0x2D, A: 0xFF} //nolint:gochecknoglobals // read-only colour
	remotePressed = color.RGBA{R: 0x8A, G: 0x5A, B: 0x1E, A: 0xFF} //nolint:gochecknoglobals // read-only colour
	remoteHint    = color.RGBA{R: 0x4E, G: 0x42, B: 0x30, A: 0xFF} //nolint:gochecknoglobals // read-only colour
)

const remoteRound = 0.18 // of a key: corner radius

// remote tracks which keypad keys are held.
type remote struct {
	area    image.Rectangle // the whole remote
	keys    [16]image.Rectangle
	held    uint16 // bit n: hex key n is held
	touches []ebiten.TouchID

	// The keypad is drawn into img only when it changes: antialiased paths
	// are slow to draw every frame, especially without a GPU.
	img       *ebiten.Image
	drawnHeld uint16
	drawnHint uint16
	stale     bool
}

// layout places the keys in a square grid centred in area.
func (r *remote) layout(area image.Rectangle) {
	r.area = area
	r.stale = true

	if r.img != nil {
		r.img.Deallocate()
		r.img = nil
	}

	if area.Empty() {
		return
	}

	r.img = ebiten.NewImage(area.Dx(), area.Dy())

	size := min(area.Dx(), area.Dy()) / 4
	gap := int(float64(size) * remoteGap)
	x0 := area.Min.X + (area.Dx()-4*size)/2
	y0 := area.Min.Y + (area.Dy()-4*size)/2

	for row, keys := range remoteKeys {
		for col, hex := range keys {
			x, y := x0+col*size, y0+row*size
			r.keys[hex] = image.Rect(x+gap/2, y+gap/2, x+size-gap/2, y+size-gap/2)
		}
	}
}

// update reads every finger on the remote.
func (r *remote) update() {
	r.held = 0
	if r.area.Empty() {
		return
	}

	r.touches = ebiten.AppendTouchIDs(r.touches[:0])
	for _, id := range r.touches {
		r.held |= r.keyAt(image.Pt(ebiten.TouchPosition(id)))
	}
}

// keyAt is the bit of the key under p, or 0.
func (r *remote) keyAt(p image.Point) uint16 {
	for hex, k := range r.keys {
		if p.In(k) {
			return 1 << hex
		}
	}

	return 0
}

// draw draws the keypad; hint keys (the game's, from the database) are
// lighter. It redraws its image only when a key or the hint changed.
func (r *remote) draw(dst *ebiten.Image, face *text.GoTextFace, hint uint16) {
	if r.img == nil {
		return
	}

	if r.stale || r.held != r.drawnHeld || hint != r.drawnHint {
		r.img.Clear()
		r.paint(r.img, face, hint)
		r.stale, r.drawnHeld, r.drawnHint = false, r.held, hint
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(r.area.Min.X), float64(r.area.Min.Y))
	dst.DrawImage(r.img, op)
}

// paint draws the keypad into img, whose origin is the remote's corner.
func (r *remote) paint(img *ebiten.Image, face *text.GoTextFace, hint uint16) {
	off := r.area.Min
	dst := img

	fill := func(rc image.Rectangle, c color.Color, radius float32) {
		var p vector.Path

		rc = rc.Sub(off)
		x, y, w, h := float32(rc.Min.X), float32(rc.Min.Y), float32(rc.Dx()), float32(rc.Dy())
		p.MoveTo(x+radius, y)
		p.LineTo(x+w-radius, y)
		p.ArcTo(x+w, y, x+w, y+radius, radius)
		p.LineTo(x+w, y+h-radius)
		p.ArcTo(x+w, y+h, x+w-radius, y+h, radius)
		p.LineTo(x+radius, y+h)
		p.ArcTo(x, y+h, x, y+h-radius, radius)
		p.LineTo(x, y+radius)
		p.ArcTo(x, y, x+radius, y, radius)
		p.Close()

		op := &vector.FillOptions{}
		dop := &vector.DrawPathOptions{AntiAlias: true}
		dop.ColorScale.ScaleWithColor(c)
		vector.FillPath(dst, &p, op, dop)
	}

	all := r.keys[0x1].Union(r.keys[0xF])
	pad := all.Dx() / 20
	fill(all.Inset(-pad), remoteBody, float32(pad*2))

	op := &text.DrawOptions{}

	for hex, k := range r.keys {
		c := remoteKey

		switch {
		case r.held&(1<<hex) != 0:
			c = remotePressed
		case hint&(1<<hex) != 0:
			c = remoteHint
		}

		fill(k, c, float32(float64(k.Dx())*remoteRound))

		label := string("0123456789ABCDEF"[hex])
		w, h := text.Measure(label, face, 0)

		op.GeoM.Reset()
		op.GeoM.Translate(float64(k.Min.X-off.X)+(float64(k.Dx())-w)/2, float64(k.Min.Y-off.Y)+(float64(k.Dy())-h)/2)
		op.ColorScale.Reset()
		op.ColorScale.ScaleWithColor(inkColour)
		text.Draw(dst, label, face, op)
	}
}
