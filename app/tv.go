package app

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/rbscholtus/chippie/app/skin"
)

// This file holds the TV: where the cabinet goes in the window, loading its
// art in the background, and drawing it with its controls. The controls'
// behaviour is in controls.go.

// Layout proportions.
const (
	tvMargin     = 0.03 // of the window's smaller side, around the TV
	statusLines  = 3    // under the TV: title, keys, a message
	pictureShare = 0.92 // of the tube's width (or height) the picture fills
	pictureShape = 2    // CHIP-8's width:height
	remoteShare  = 0.3  // of the width, beside the TV in landscape
	remoteMinKey = 52   // CSS pixels: the smallest remote key in portrait
	remoteGap    = 0.12 // of a key, between keys
	fadeStep     = 1.0 / 20
	lampRadius   = 4.5 // cabinet pixels: the power pilot lamp
	lampGap      = 16  // cabinet pixels left of the rocker
)

// Fixed colours of the TV.
var (
	lampColour = color.RGBA{R: 0xFF, G: 0x9A, B: 0x2E, A: 0xFF} //nolint:gochecknoglobals // read-only colour
	lampGlow   = color.RGBA{R: 0x7A, G: 0x3A, B: 0x08, A: 0x60} //nolint:gochecknoglobals // read-only colour
)

// ArtFetcher reads one of the TV's image files by name ("cabinet@2x.jpg").
// It may block; it is only called from a goroutine.
type ArtFetcher func(ctx context.Context, name string) ([]byte, error)

// tvLayout is where the TV and what surrounds it go, in real pixels.
type tvLayout struct {
	k       float64         // real pixels per cabinet pixel
	cabinet image.Rectangle // the cabinet image
	tube    image.Rectangle // the picture tube (the shader's area)
	picture image.Rectangle // the emulated picture, within the tube
	below   image.Rectangle // the status lines under the TV
	remote  image.Rectangle // the touch remote; empty when not shown
}

// toCabinet converts a window point to cabinet pixels.
func (l tvLayout) toCabinet(p image.Point) (float64, float64) {
	return float64(p.X-l.cabinet.Min.X) / l.k, float64(p.Y-l.cabinet.Min.Y) / l.k
}

// fromCabinet converts cabinet pixels to window pixels.
func (l tvLayout) fromCabinet(x, y float64) (float64, float64) {
	return float64(l.cabinet.Min.X) + x*l.k, float64(l.cabinet.Min.Y) + y*l.k
}

// computeTV places the TV in the window. In portrait it sits at the top,
// as wide as the window, with the status lines and (with remote) the remote
// below. In landscape it is centred and as large as fits; the remote, if
// shown, takes a column on the right.
func computeTV(sk *skin.Skin, window image.Rectangle, scale, lineH float64, remote bool) tvLayout {
	w, h := float64(window.Dx()), float64(window.Dy())
	margin := min(w, h) * tvMargin
	statusH := statusLines * lineH
	portrait := h > w

	area := image.Rect(window.Min.X, window.Min.Y, window.Max.X, window.Max.Y)

	var rem image.Rectangle

	if remote && !portrait {
		split := window.Max.X - int(w*remoteShare)
		area.Max.X = split
		rem = image.Rect(split, window.Min.Y, window.Max.X, window.Max.Y).Inset(int(margin))
	}

	availW := float64(area.Dx()) - 2*margin
	availH := float64(area.Dy()) - 2*margin - statusH

	if remote && portrait {
		// Keep room for a 4×4 keypad of usable keys under the status lines.
		availH -= 4 * remoteMinKey * scale * (1 + remoteGap)
	}

	k := max(min(availW/sk.Width, availH/sk.Height), 0.01)
	cw, ch := sk.Width*k, sk.Height*k
	x := float64(area.Min.X) + (float64(area.Dx())-cw)/2
	y := float64(area.Min.Y) + margin

	if !portrait {
		y += max(0, (float64(area.Dy())-2*margin-statusH-ch)/2)
	}

	l := tvLayout{k: k, cabinet: image.Rect(int(x), int(y), int(x+cw), int(y+ch))}
	l.tube = l.rect(sk.Screen)
	l.picture = pictureIn(l.tube)
	l.below = image.Rect(window.Min.X, l.cabinet.Max.Y, area.Max.X, l.cabinet.Max.Y+int(statusH))

	switch {
	case remote && portrait:
		l.remote = image.Rect(window.Min.X, l.below.Max.Y, window.Max.X, window.Max.Y).Inset(int(margin))
	case remote:
		l.remote = rem
	}

	return l
}

// rect converts a cabinet rectangle to window pixels.
func (l tvLayout) rect(r skin.Rect) image.Rectangle {
	x0, y0 := l.fromCabinet(r.X, r.Y)
	x1, y1 := l.fromCabinet(r.X+r.W, r.Y+r.H)

	return image.Rect(int(math.Round(x0)), int(math.Round(y0)), int(math.Round(x1)), int(math.Round(y1)))
}

// pictureIn is the largest 2:1 picture filling pictureShare of the tube,
// centred in it.
func pictureIn(tube image.Rectangle) image.Rectangle {
	pw := float64(tube.Dx()) * pictureShare
	ph := pw / pictureShape

	if limit := float64(tube.Dy()) * pictureShare; ph > limit {
		ph, pw = limit, limit*pictureShape
	}

	x := float64(tube.Min.X) + (float64(tube.Dx())-pw)/2
	y := float64(tube.Min.Y) + (float64(tube.Dy())-ph)/2

	return image.Rect(int(x), int(y), int(x+pw), int(y+ph))
}

// tvArt is the loaded images at one scale.
type tvArt struct {
	scale    int
	cabinet  *ebiten.Image
	backdrop *ebiten.Image
	sprites  map[string]*ebiten.Image
}

// artResult is a decoded set of images, from the loading goroutine.
type artResult struct {
	dir      string // the skin they are for
	scale    int
	cabinet  image.Image
	backdrop image.Image
	sprites  map[string]image.Image
	err      error
}

// loadArt fetches and decodes a skin's images (from its folder dir) at
// scale n in a goroutine. The cabinet comes as a JPEG and a separate
// greyscale mask, combined here into one image that is transparent where
// the tube shows through.
func loadArt(fetch ArtFetcher, dir string, sk *skin.Skin, n int) <-chan artResult {
	out := make(chan artResult, 1) // buffered: a result nobody wants doesn't block

	go func() {
		res := artResult{dir: dir, scale: n, sprites: map[string]image.Image{}}
		ctx := context.Background()

		get := func(name string, decode func([]byte) (image.Image, error)) (image.Image, error) {
			data, err := fetch(ctx, dir+"/"+name)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}

			m, err := decode(data)
			if err != nil {
				return nil, fmt.Errorf("decode %s: %w", name, err)
			}

			return m, nil
		}

		var err error

		res.backdrop, err = get(sk.Backdrop, decodeJPEG)

		var cab image.Image
		if err == nil {
			cab, err = get(skin.FileName(sk.Cabinet, n, "jpg"), decodeJPEG)
		}

		if err == nil {
			var mask image.Image
			if mask, err = get(skin.FileName(sk.Mask, n, "png"), decodePNG); err == nil {
				res.cabinet = withMask(cab, mask)
			}
		}

		for _, c := range sk.Controls {
			if err != nil {
				break
			}

			res.sprites[c.Image], err = get(skin.FileName(c.Image, n, "png"), decodePNG)
		}

		res.err = err
		out <- res
	}()

	return out
}

func decodeJPEG(b []byte) (image.Image, error) {
	return jpeg.Decode(bytes.NewReader(b)) //nolint:wrapcheck // get adds context
}

func decodePNG(b []byte) (image.Image, error) {
	return png.Decode(bytes.NewReader(b)) //nolint:wrapcheck // get adds context
}

// withMask returns img with its alpha taken from mask's brightness.
func withMask(img, mask image.Image) *image.NRGBA {
	b := img.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(out, out.Bounds(), img, b.Min, draw.Src)

	mb := mask.Bounds()

	for y := range min(b.Dy(), mb.Dy()) {
		for x := range min(b.Dx(), mb.Dx()) {
			r, _, _, _ := color.GrayModel.Convert(mask.At(mb.Min.X+x, mb.Min.Y+y)).RGBA()
			out.Pix[out.PixOffset(x, y)+3] = uint8(r >> 8) //nolint:gosec // G115: 16-bit colour to 8
		}
	}

	return out
}

// tv is the TV set: its skin, layout, art and the controls' positions.
type tv struct {
	skin  *skin.Skin
	dir   string     // the skin's folder: where its images are
	fetch ArtFetcher // nil: no art (tests), just the picture
	lay   tvLayout

	art     *tvArt
	loading <-chan artResult
	fade    float32 // the cabinet fading in once loaded, 0 to 1
	artErr  error

	controls controlState
	spriteOp ebiten.DrawImageOptions
}

func newTV(sk *skin.Skin, dir string, fetch ArtFetcher) *tv {
	return &tv{skin: sk, dir: dir, fetch: fetch, controls: defaultControls()}
}

// setSkin switches to another skin. Its art loads at the next relayout;
// a load still running for the old skin is ignored when it arrives.
func (t *tv) setSkin(sk *skin.Skin, dir string) {
	if t.art != nil {
		t.art.dispose()
	}

	t.skin, t.dir = sk, dir
	t.art, t.loading, t.fade, t.artErr = nil, nil, 0, nil
}

// relayout places the TV and starts loading art big enough for it.
func (t *tv) relayout(window image.Rectangle, scale, lineH float64, remote bool) {
	t.lay = computeTV(t.skin, window, scale, lineH, remote)

	n := t.skin.ScaleFor(float64(t.lay.cabinet.Dx()))
	if t.fetch == nil || t.loading != nil || (t.art != nil && t.art.scale >= n) {
		return
	}

	t.loading = loadArt(t.fetch, t.dir, t.skin, n)
}

// update collects loaded art and fades the cabinet in, reporting whether
// new art arrived. Creating the Ebitengine images happens here, on the
// game goroutine.
func (t *tv) update() bool {
	arrived := false

	select {
	case res := <-t.loading:
		t.loading = nil

		if res.err != nil {
			t.artErr = res.err

			break
		}

		art := &tvArt{
			scale: res.scale, cabinet: ebiten.NewImageFromImage(res.cabinet),
			backdrop: ebiten.NewImageFromImage(res.backdrop), sprites: map[string]*ebiten.Image{},
		}
		for name, m := range res.sprites {
			art.sprites[name] = ebiten.NewImageFromImage(m)
		}

		if t.art != nil {
			t.art.dispose()
		}

		t.art = art
		arrived = true
	default:
	}

	if t.art != nil && t.fade < 1 {
		t.fade = min(1, t.fade+fadeStep)
	}

	return arrived
}

func (a *tvArt) dispose() {
	a.cabinet.Deallocate()
	a.backdrop.Deallocate()

	for _, s := range a.sprites {
		s.Deallocate()
	}
}

// drawBackdrop draws the blurred studio behind where the cabinet goes,
// before the picture and the cabinet.
func (t *tv) drawBackdrop(dst *ebiten.Image) {
	if t.art == nil {
		return
	}

	op := &t.spriteOp
	op.GeoM.Reset()
	op.ColorScale.Reset()
	op.ColorScale.ScaleAlpha(t.fade)
	op.Filter = ebiten.FilterLinear

	r := t.lay.rect(t.skin.BackdropRect)
	b := t.art.backdrop.Bounds()
	op.GeoM.Scale(float64(r.Dx())/float64(b.Dx()), float64(r.Dy())/float64(b.Dy()))
	op.GeoM.Translate(float64(r.Min.X), float64(r.Min.Y))
	dst.DrawImage(t.art.backdrop, op)
}

// draw draws the cabinet over the picture, then the controls in their
// current positions and the pilot lamp.
func (t *tv) draw(dst *ebiten.Image) {
	if t.art == nil {
		return
	}

	op := &t.spriteOp
	op.GeoM.Reset()
	op.ColorScale.Reset()
	op.ColorScale.ScaleAlpha(t.fade)
	op.Filter = ebiten.FilterLinear

	cab := t.art.cabinet.Bounds()
	op.GeoM.Scale(float64(t.lay.cabinet.Dx())/float64(cab.Dx()), float64(t.lay.cabinet.Dy())/float64(cab.Dy()))
	op.GeoM.Translate(float64(t.lay.cabinet.Min.X), float64(t.lay.cabinet.Min.Y))
	dst.DrawImage(t.art.cabinet, op)

	for _, c := range t.skin.Controls {
		img, ok := t.art.sprites[c.Image]
		if !ok {
			continue
		}

		t.drawControl(dst, c, img)
	}

	if power, ok := t.skin.Control(controlPower); ok && t.controls.on {
		x, y := t.lay.fromCabinet(power.X-lampGap, power.Y+power.H/2)
		r := float32(lampRadius * t.lay.k)
		vector.FillCircle(dst, float32(x), float32(y), 2.2*r, lampGlow, true)
		vector.FillCircle(dst, float32(x), float32(y), r, lampColour, true)
	}
}

// drawControl draws one control's sprite, turned (round) or flipped
// (rocker, when on), scaled to its place.
func (t *tv) drawControl(dst *ebiten.Image, c skin.Control, img *ebiten.Image) {
	op := &t.spriteOp
	b := c.Bounds()
	sw, sh := float64(img.Bounds().Dx()), float64(img.Bounds().Dy())

	op.GeoM.Reset()
	op.GeoM.Translate(-sw/2, -sh/2)

	if c.Round() {
		op.GeoM.Rotate(t.controls.angle(c) * math.Pi / 180)
	} else if t.controls.on {
		op.GeoM.Scale(1, -1) // the rocker pressed the other way
	}

	op.GeoM.Scale(b.W*t.lay.k/sw, b.H*t.lay.k/sh)

	cx, cy := c.Centre()
	x, y := t.lay.fromCabinet(cx, cy)
	op.GeoM.Translate(x, y)
	dst.DrawImage(img, op)
}

// drawCalibration outlines every control's sprite and hit area and the
// screen, over the art (F3, or ?skin=debug).
func (t *tv) drawCalibration(dst *ebiten.Image, minR float64) {
	red := color.RGBA{R: 0xFF, G: 0x30, B: 0x30, A: 0xFF}
	cyan := color.RGBA{R: 0x30, G: 0xE0, B: 0xFF, A: 0xFF}

	strokeRect(dst, t.lay.tube, red)
	strokeRect(dst, t.lay.picture, cyan)

	for _, c := range t.skin.Controls {
		strokeRect(dst, t.lay.rect(c.Bounds()), red)

		cx, cy := c.Centre()
		x, y := t.lay.fromCabinet(cx, cy)
		r := max(c.R, minR) * t.lay.k

		if !c.Round() {
			r = max(c.W, c.H, 2*minR) / 2 * t.lay.k
		}

		vector.StrokeCircle(dst, float32(x), float32(y), float32(r), 1, cyan, true)
	}
}

func strokeRect(dst *ebiten.Image, r image.Rectangle, c color.Color) {
	vector.StrokeRect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), 1, c, false)
}
