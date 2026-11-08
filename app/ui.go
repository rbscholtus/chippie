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
	panelColour  = color.RGBA{R: 0x10, G: 0x0E, B: 0x0B, A: 0xF0} //nolint:gochecknoglobals // read-only colour
	inkColour    = color.RGBA{R: 0xEF, G: 0xE3, B: 0xCC, A: 0xFF} //nolint:gochecknoglobals // read-only colour
	mutedColour  = color.RGBA{R: 0x8A, G: 0x7C, B: 0x66, A: 0xFF} //nolint:gochecknoglobals // read-only colour
	accentColour = color.RGBA{R: 0xFF, G: 0xB4, B: 0x3C, A: 0xFF} //nolint:gochecknoglobals // read-only colour
	errorColour  = color.RGBA{R: 0xFF, G: 0x7A, B: 0x6B, A: 0xFF} //nolint:gochecknoglobals // read-only colour
)

// Sizes in CSS pixels; multiplied by the device scale.
const (
	textSize      = 16
	lineSpacing   = 1.5
	panelPad      = 16
	touchRowSize  = 44 // Apple's minimum touch target
	dragThreshold = 8
)

// ui draws text overlays (the picker, messages, the debug line) and owns
// the layout that input handling uses for hit testing.
type ui struct {
	source *text.GoTextFaceSource
	face   *text.GoTextFace
	scale  float64
	touch  bool // a touch was seen: bigger rows, touch wording

	layout layout
}

// layout holds the rectangles of the overlays, in real pixels.
type layout struct {
	window image.Rectangle
	panel  image.Rectangle // the picker's background
	list   image.Rectangle // the scrolling rows, inside the panel
	status image.Rectangle // below the picture: tap here for the list
	rowH   float64
}

func newUI() (*ui, error) {
	src, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		return nil, fmt.Errorf("load font: %w", err)
	}

	u := &ui{source: src}
	u.resize(image.Rect(0, 0, 1, 1), image.Rect(0, 0, 1, 1), 1)

	return u, nil
}

func (u *ui) lineHeight() float64 { return textSize * lineSpacing * u.scale }

// resize lays out the overlays for a window and the picture's rectangle.
func (u *ui) resize(window, screen image.Rectangle, scale float64) {
	u.scale = scale
	u.face = &text.GoTextFace{Source: u.source, Size: textSize * scale}
	u.layout = computeLayout(window, screen, scale, u.lineHeight(), u.touch)
}

// computeLayout places the picker panel over the whole window, leaving a
// line at the top (debug) and two at the bottom (status).
func computeLayout(window, screen image.Rectangle, scale, lineH float64, touch bool) layout {
	pad := int(panelPad * scale)
	statusH := int(2 * lineH)

	rowH := lineH
	if touch {
		rowH = max(rowH, touchRowSize*scale)
	}

	panel := image.Rect(window.Min.X+pad, window.Min.Y+int(lineH)+pad, window.Max.X-pad, window.Max.Y-statusH)
	if panel.Dy() < 3*int(rowH) {
		panel = window
	}

	return layout{
		window: window,
		panel:  panel,
		list:   panel.Inset(pad),
		status: image.Rect(window.Min.X, screen.Max.Y, window.Max.X, window.Max.Y),
		rowH:   rowH,
	}
}

func (u *ui) draw(dst *ebiten.Image, g *Game) {
	screen := g.renderer.Screen()
	lh := u.lineHeight()

	switch g.state {
	case stateOff:
		u.centred(dst, screen, "Tap, click or press a key to switch on", inkColour)
	case statePicker:
		u.drawPicker(dst, g)
	case stateLoading:
		u.centred(dst, screen, "Tuning in…", inkColour)
	case stateRunning:
	}

	// The status line: under the picture, or under the picker's panel.
	x := float64(screen.Min.X)
	y := float64(screen.Max.Y) + lh/2

	if g.state == statePicker {
		x = float64(u.layout.panel.Min.X)
		y = float64(u.layout.panel.Max.Y) + lh/2
	}

	maxW := float64(u.layout.window.Max.X) - x - panelPad*u.scale

	if g.message != "" {
		u.text(dst, u.fit(g.message, maxW), x, y, errorColour)
	} else if s := u.status(g); s != "" {
		u.text(dst, u.fit(s, maxW), x, y, mutedColour)
	}

	if g.debug {
		msg := fmt.Sprintf("TPS %.0f  FPS %.0f  scale %.1f×  %d×%d",
			ebiten.ActualTPS(), ebiten.ActualFPS(), g.scale, g.width, g.height)
		u.text(dst, msg, panelPad*u.scale, panelPad*u.scale/2, mutedColour)
	}
}

// status is the hint line: what's playing and how to get around, in touch
// or keyboard wording.
func (u *ui) status(g *Game) string {
	switch g.state {
	case stateRunning:
		if u.touch {
			return g.current.Title + "  ·  tap here for the ROM list"
		}

		return g.current.Title + "  ·  Esc or click here: ROM list  ·  Keys: 1–4, Q–R, A–F, Z–V  ·  F1: stats"
	case statePicker:
		hint := "↑↓ choose  ·  Enter or click: play"
		if u.touch {
			hint = "Tap a ROM to play  ·  drag to scroll"
		}

		if g.hasROM {
			if u.touch {
				hint += "  ·  tap here to go back"
			} else {
				hint += "  ·  Esc: back"
			}
		}

		return hint
	case stateOff, stateLoading:
		return ""
	default:
		return ""
	}
}

func (u *ui) drawPicker(dst *ebiten.Image, g *Game) {
	l := u.layout
	vector.FillRect(dst, float32(l.panel.Min.X), float32(l.panel.Min.Y),
		float32(l.panel.Dx()), float32(l.panel.Dy()), panelColour, false)

	// Draw the rows into the list area only, so partly visible rows are
	// clipped at its edges.
	list, ok := dst.SubImage(l.list).(*ebiten.Image)
	if !ok {
		return
	}

	p := g.picker
	textOffset := (l.rowH - u.lineHeight()) / 2
	maxW := float64(l.list.Dx())
	x := float64(l.list.Min.X)

	for r, row := range p.rows {
		y := float64(l.list.Min.Y) + float64(r)*l.rowH - p.offset
		if y+l.rowH < float64(l.list.Min.Y) || y > float64(l.list.Max.Y) {
			continue
		}

		if row.entry < 0 {
			u.text(list, u.fit(row.heading, maxW), x, y+textOffset, mutedColour)

			continue
		}

		e := p.entries[row.entry]
		prefix, col := "   ", inkColour

		if !e.Playable() {
			col = mutedColour
		}

		if row.entry == p.selected {
			prefix, col = "> ", accentColour
		}

		label := prefix + e.Title
		if !e.Playable() {
			label += "  —  soon: " + e.Platform.Name()
		}

		u.text(list, u.fit(label, maxW), x, y+textOffset, col)
	}
}

// fit shortens s with "…" until it is at most maxW pixels wide.
func (u *ui) fit(s string, maxW float64) string {
	if w, _ := text.Measure(s, u.face, 0); w <= maxW {
		return s
	}

	r := []rune(s)
	for len(r) > 0 {
		r = r[:len(r)-1]

		t := string(r) + "…"
		if w, _ := text.Measure(t, u.face, 0); w <= maxW {
			return t
		}
	}

	return ""
}

func (u *ui) centred(dst *ebiten.Image, r image.Rectangle, s string, c color.Color) {
	s = u.fit(s, float64(r.Dx()))
	w, h := text.Measure(s, u.face, 0)
	u.text(dst, s, float64(r.Min.X)+(float64(r.Dx())-w)/2, float64(r.Min.Y)+(float64(r.Dy())-h)/2, c)
}

func (u *ui) text(dst *ebiten.Image, s string, x, y float64, c color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	text.Draw(dst, s, u.face, op)
}
