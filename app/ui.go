package app

import (
	"bytes"
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/goregular"
)

// Colours of the room and the overlays.
var (
	roomColour   = color.RGBA{R: 0x16, G: 0x13, B: 0x0F, A: 0xFF} //nolint:gochecknoglobals // read-only colour
	panelColour  = color.RGBA{R: 0x10, G: 0x0E, B: 0x0B, A: 0xE8} //nolint:gochecknoglobals // read-only colour
	inkColour    = color.RGBA{R: 0xEF, G: 0xE3, B: 0xCC, A: 0xFF} //nolint:gochecknoglobals // read-only colour
	mutedColour  = color.RGBA{R: 0x8A, G: 0x7C, B: 0x66, A: 0xFF} //nolint:gochecknoglobals // read-only colour
	accentColour = color.RGBA{R: 0xFF, G: 0xB4, B: 0x3C, A: 0xFF} //nolint:gochecknoglobals // read-only colour
	errorColour  = color.RGBA{R: 0xFF, G: 0x7A, B: 0x6B, A: 0xFF} //nolint:gochecknoglobals // read-only colour
)

// Text sizes in CSS pixels; multiplied by the device scale.
const (
	textSize    = 16
	lineSpacing = 1.5
	panelPad    = 16
)

// ui draws text overlays: the picker, messages and the debug line.
type ui struct {
	source *text.GoTextFaceSource
	face   *text.GoTextFace
	scale  float64

	// entryRects holds the picker rows drawn last frame, for clicks.
	entryRects []entryRect
}

type entryRect struct {
	rect  image.Rectangle
	entry int
}

func newUI() (*ui, error) {
	src, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		return nil, fmt.Errorf("load font: %w", err)
	}

	u := &ui{source: src}
	u.resize(1)

	return u, nil
}

// resize scales the font to the device scale factor.
func (u *ui) resize(scale float64) {
	u.scale = scale
	u.face = &text.GoTextFace{Source: u.source, Size: textSize * scale}
}

func (u *ui) lineHeight() float64 { return textSize * lineSpacing * u.scale }

// entryAt returns the picker entry drawn at pt.
func (u *ui) entryAt(pt image.Point) (int, bool) {
	for _, r := range u.entryRects {
		if pt.In(r.rect) {
			return r.entry, true
		}
	}

	return 0, false
}

func (u *ui) draw(dst *ebiten.Image, g *Game) {
	screen := g.renderer.Screen()

	switch g.state {
	case stateOff:
		u.centred(dst, screen, "Click or press any key to switch on", inkColour)
	case statePicker:
		u.drawPicker(dst, g, screen)
	case stateLoading:
		u.centred(dst, screen, "Tuning in…", inkColour)
	case stateRunning:
	}

	status := u.status(g)
	if g.message != "" {
		u.text(dst, g.message, float64(screen.Min.X), float64(screen.Max.Y)+u.lineHeight()/2, errorColour)
	} else if status != "" {
		u.text(dst, status, float64(screen.Min.X), float64(screen.Max.Y)+u.lineHeight()/2, mutedColour)
	}

	if g.debug {
		msg := fmt.Sprintf("TPS %.0f  FPS %.0f  scale %.1f×  %d×%d",
			ebiten.ActualTPS(), ebiten.ActualFPS(), g.scale, g.width, g.height)
		u.text(dst, msg, panelPad*u.scale, panelPad*u.scale/2, mutedColour)
	}
}

// status is the line under the picture: what's playing and the keys.
func (u *ui) status(g *Game) string {
	switch g.state {
	case stateRunning:
		return g.current.Title + "  ·  Esc: pick a ROM  ·  Keys: 1-4, Q-R, A-F, Z-V  ·  F1: stats"
	case statePicker:
		return "↑↓ choose  ·  Enter or click: play  ·  Esc: back"
	case stateOff, stateLoading:
		return ""
	default:
		return ""
	}
}

func (u *ui) drawPicker(dst *ebiten.Image, g *Game, screen image.Rectangle) {
	pad := panelPad * u.scale

	vector.FillRect(dst, float32(screen.Min.X), float32(screen.Min.Y),
		float32(screen.Dx()), float32(screen.Dy()), panelColour, false)

	rows := g.picker.rows()
	lh := u.lineHeight()
	maxRows := int((float64(screen.Dy()) - 2*pad) / lh)
	first := g.picker.visibleRows(rows, maxRows)

	u.entryRects = u.entryRects[:0]
	x := float64(screen.Min.X) + pad
	y := float64(screen.Min.Y) + pad

	for i := first; i < len(rows) && i < first+maxRows; i++ {
		r := rows[i]
		if r.entry < 0 {
			u.text(dst, r.heading, x, y, mutedColour)
			y += lh

			continue
		}

		e := g.picker.entries[r.entry]
		label, col := e.Title, inkColour

		if !e.Playable() {
			label += "  —  coming soon (" + e.Platform.Name() + ")"
			col = mutedColour
		}

		if r.entry == g.picker.selected {
			label = "> " + label
			col = accentColour
		} else {
			label = "   " + label
		}

		u.text(dst, label, x, y, col)
		u.entryRects = append(u.entryRects, entryRect{
			rect:  image.Rect(screen.Min.X, int(y), screen.Max.X, int(y+lh)),
			entry: r.entry,
		})
		y += lh
	}
}

func (u *ui) centred(dst *ebiten.Image, r image.Rectangle, s string, c color.Color) {
	w, h := text.Measure(s, u.face, 0)
	u.text(dst, s, float64(r.Min.X)+(float64(r.Dx())-w)/2, float64(r.Min.Y)+(float64(r.Dy())-h)/2, c)
}

func (u *ui) text(dst *ebiten.Image, s string, x, y float64, c color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	text.Draw(dst, s, u.face, op)
}
