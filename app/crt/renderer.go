// Package crt draws the CHIP-8 display: the framebuffer is copied into a
// texture, scaled up, passed through a phosphor afterglow shader and then a
// CRT shader (curvature, scanlines, shadow mask, bloom, power animation).
// A "clean" mode draws the plain scaled picture, for tests.
package crt

import (
	_ "embed"
	"fmt"
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/rbscholtus/chippie/chip8"
)

var (
	//go:embed crt.kage
	crtSource []byte
	//go:embed phosphor.kage
	phosphorSource []byte
)

// Mode selects how the picture is drawn.
type Mode uint8

// Drawing modes.
const (
	ModeCRT   Mode = iota // full CRT effect
	ModeClean             // plain nearest-neighbour scaling
)

// PhosphorDecay is how much of the previous frame's glow remains each
// emulated frame.
const PhosphorDecay = 0.82

// framesPerSecond converts ticks to seconds for the flicker.
const framesPerSecond = 60

// Renderer draws the display. Images, shaders, options and uniforms are
// created once (or on resize) and reused every frame.
type Renderer struct {
	mode Mode

	crt      *ebiten.Shader
	phosphor *ebiten.Shader

	fb   *ebiten.Image // display resolution
	pix  []byte        // RGBA staging buffer for fb
	up   *ebiten.Image // screen resolution, nearest-neighbour upscale
	phos [2]*ebiten.Image
	cur  int // index of the current phosphor image

	screen  image.Rectangle // the picture tube: the area the shader draws
	picture image.Rectangle // the emulated picture within it; empty = all of it
	overlay *ebiten.Image   // the on-screen display, screen-sized; may be nil
	dirty   bool            // a new frame or overlay hasn't gone through the phosphor yet

	upOpts      ebiten.DrawImageOptions
	overOpts    ebiten.DrawImageOptions
	phosOpts    ebiten.DrawRectShaderOptions
	crtOpts     ebiten.DrawRectShaderOptions
	tint        []float32
	palette     Palette
	frameWidth  int
	frameHeight int
	power       float32
	rotation    int // degrees clockwise: 0, 90, 180 or 270
}

// New compiles the shaders. Call it after Ebitengine is set up (from the
// game, not from init).
func New(mode Mode) (*Renderer, error) {
	r := &Renderer{mode: mode, palette: DefaultPalette, power: 1}

	if mode == ModeCRT {
		var err error

		if r.crt, err = ebiten.NewShader(crtSource); err != nil {
			return nil, fmt.Errorf("compile CRT shader: %w", err)
		}

		if r.phosphor, err = ebiten.NewShader(phosphorSource); err != nil {
			return nil, fmt.Errorf("compile phosphor shader: %w", err)
		}
	}

	r.tint = []float32{0.95, 0.93, 0.85} // warm black-and-white tube
	r.phosOpts.Uniforms = map[string]any{"Decay": float32(PhosphorDecay)}
	r.crtOpts.Uniforms = map[string]any{"Time": float32(0), "Power": float32(1), "Tint": r.tint}

	return r, nil
}

// SetPalette sets the colour of each pixel value, from the next frame on.
func (r *Renderer) SetPalette(p Palette) { r.palette = p }

// SetPower sets the power animation: 0 is off, 1 is fully on.
func (r *Renderer) SetPower(p float32) { r.power = min(max(p, 0), 1) }

// SetScreen sets where the picture is drawn, in destination pixels. The
// screen-sized images are reallocated only when the size changes.
func (r *Renderer) SetScreen(rect image.Rectangle) {
	if rect.Dx() <= 0 || rect.Dy() <= 0 {
		return
	}

	sizeChanged := rect.Dx() != r.screen.Dx() || rect.Dy() != r.screen.Dy()
	r.screen = rect

	if !sizeChanged || r.mode != ModeCRT {
		return
	}

	for _, img := range []*ebiten.Image{r.up, r.phos[0], r.phos[1]} {
		if img != nil {
			img.Deallocate()
		}
	}

	r.up = ebiten.NewImage(rect.Dx(), rect.Dy())
	r.phos[0] = ebiten.NewImage(rect.Dx(), rect.Dy())
	r.phos[1] = ebiten.NewImage(rect.Dx(), rect.Dy())
	r.dirty = true
}

// Screen returns where the picture tube is drawn.
func (r *Renderer) Screen() image.Rectangle { return r.screen }

// SetPicture sets where the emulated picture goes, within the screen (in
// destination pixels). The rest of the tube stays black, as on a real TV
// showing a picture of another shape. Empty means the whole screen.
func (r *Renderer) SetPicture(rect image.Rectangle) {
	r.picture = rect
	r.dirty = true
}

// SetRotation turns the emulated picture clockwise by 0, 90, 180 or 270
// degrees, for games made for a sideways screen. The picture rectangle
// (SetPicture) should already have the turned shape.
func (r *Renderer) SetRotation(deg int) {
	r.rotation = deg
	r.dirty = true
}

// PlaceFrame sets m to map a w×h frame into pic, turned clockwise by rot
// degrees (a multiple of 90) and stretched to fill it.
func PlaceFrame(m *ebiten.GeoM, w, h int, pic image.Rectangle, rot int) {
	tw, th := float64(w), float64(h) // the frame's size once turned
	if rot == 90 || rot == 270 {
		tw, th = th, tw
	}

	m.Reset()
	m.Translate(-float64(w)/2, -float64(h)/2)
	m.Rotate(float64(rot) * math.Pi / 180)
	m.Scale(float64(pic.Dx())/tw, float64(pic.Dy())/th)
	m.Translate(float64(pic.Min.X)+float64(pic.Dx())/2, float64(pic.Min.Y)+float64(pic.Dy())/2)
}

// Picture returns where the emulated picture is drawn.
func (r *Renderer) Picture() image.Rectangle {
	if r.picture.Empty() || !r.picture.In(r.screen) {
		return r.screen
	}

	return r.picture
}

// SetOverlay sets an image drawn over the picture before the CRT effect, so
// it glows and curves with it: the on-screen display. It must be the
// screen's size. Call Redraw after changing its contents.
func (r *Renderer) SetOverlay(img *ebiten.Image) {
	r.overlay = img
	r.dirty = true
}

// Redraw makes the next Draw pick up a changed overlay, even while no new
// frames arrive.
func (r *Renderer) Redraw() { r.dirty = true }

// Update copies a new emulated frame into the texture. Call it once per
// RunFrame.
func (r *Renderer) Update(v chip8.DisplayView) {
	w, h := v.Width(), v.Height()
	if w != r.frameWidth || h != r.frameHeight {
		if r.fb != nil {
			r.fb.Deallocate()
		}

		r.fb = ebiten.NewImage(w, h)
		r.pix = make([]byte, w*h*bytesPerPixel)
		r.frameWidth, r.frameHeight = w, h
	}

	CopyFramebuffer(v, &r.palette, r.pix)
	r.fb.WritePixels(r.pix)
	r.dirty = true
}

// Static shows TV static (random grey noise) at the current resolution,
// while a ROM is tuning in. seed changes the pattern; pass the tick.
func (r *Renderer) Static(seed uint64) {
	if r.fb == nil {
		return
	}

	FillStatic(r.pix, seed)
	r.fb.WritePixels(r.pix)
	r.dirty = true
}

// Draw draws the picture into dst at the screen rectangle.
func (r *Renderer) Draw(dst *ebiten.Image, tick uint64) {
	if r.fb == nil || r.screen.Empty() {
		return
	}

	if r.mode == ModeClean {
		r.drawClean(dst)

		return
	}

	if r.dirty {
		r.up.Fill(r.palette[0]) // the whole tube shows the game's background
		PlaceFrame(&r.upOpts.GeoM, r.frameWidth, r.frameHeight, r.Picture().Sub(r.screen.Min), r.rotation)
		r.up.DrawImage(r.fb, &r.upOpts)

		if r.overlay != nil {
			r.overOpts.GeoM.Reset()
			r.up.DrawImage(r.overlay, &r.overOpts)
		}

		// Phosphor: new glow = max(current picture, previous glow × decay).
		prev := r.phos[r.cur]
		r.cur ^= 1
		r.phosOpts.Images[0], r.phosOpts.Images[1] = r.up, prev
		r.phos[r.cur].DrawRectShader(r.screen.Dx(), r.screen.Dy(), r.phosphor, &r.phosOpts)
		r.dirty = false
	}

	r.crtOpts.Uniforms["Time"] = float32(tick) / framesPerSecond
	r.crtOpts.Uniforms["Power"] = r.power
	r.crtOpts.Images[0] = r.phos[r.cur]
	r.crtOpts.GeoM.Reset()
	r.crtOpts.GeoM.Translate(float64(r.screen.Min.X), float64(r.screen.Min.Y))
	dst.DrawRectShader(r.screen.Dx(), r.screen.Dy(), r.crt, &r.crtOpts)
}

// drawClean draws the plain picture with nearest-neighbour scaling, on a
// black tube, with the overlay on top.
func (r *Renderer) drawClean(dst *ebiten.Image) {
	if tube, ok := dst.SubImage(r.screen).(*ebiten.Image); ok {
		tube.Fill(r.palette[0])
	}

	PlaceFrame(&r.upOpts.GeoM, r.frameWidth, r.frameHeight, r.Picture(), r.rotation)
	dst.DrawImage(r.fb, &r.upOpts)

	if r.overlay != nil {
		r.overOpts.GeoM.Reset()
		r.overOpts.GeoM.Translate(float64(r.screen.Min.X), float64(r.screen.Min.Y))
		dst.DrawImage(r.overlay, &r.overOpts)
	}
}
