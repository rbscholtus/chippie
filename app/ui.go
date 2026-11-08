package app

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/rbscholtus/chippie/app/input"
	"github.com/rbscholtus/chippie/app/library"
	"github.com/rbscholtus/chippie/chip8"
)

// Colours of the room and the overlays.
var (
	roomColour     = color.RGBA{R: 0x16, G: 0x13, B: 0x0F, A: 0xFF} //nolint:gochecknoglobals // read-only colour
	panelColour    = color.RGBA{R: 0x10, G: 0x0E, B: 0x0B, A: 0xF0} //nolint:gochecknoglobals // read-only colour
	selectedColour = color.RGBA{R: 0x2C, G: 0x26, B: 0x1E, A: 0xFF} //nolint:gochecknoglobals // read-only colour
	inkColour      = color.RGBA{R: 0xEF, G: 0xE3, B: 0xCC, A: 0xFF} //nolint:gochecknoglobals // read-only colour
	mutedColour    = color.RGBA{R: 0x8A, G: 0x7C, B: 0x66, A: 0xFF} //nolint:gochecknoglobals // read-only colour
	accentColour   = color.RGBA{R: 0xFF, G: 0xB4, B: 0x3C, A: 0xFF} //nolint:gochecknoglobals // read-only colour
	errorColour    = color.RGBA{R: 0xFF, G: 0x7A, B: 0x6B, A: 0xFF} //nolint:gochecknoglobals // read-only colour
)

// Sizes in CSS pixels; multiplied by the device scale.
const (
	textSize       = 16
	lineSpacing    = 1.5
	panelPad       = 16
	touchRowSize   = 44  // Apple's minimum touch target
	dragThreshold  = 8   // pixels of movement before a press scrolls
	detailsMinWide = 720 // the drawer shows a details pane from this width
	detailsShare   = 0.6 // descriptions are often long
	traceLines     = 8
)

// ui draws the overlays (drawer, settings, messages, the debug line) and
// owns the layout that input handling uses for hit testing.
type ui struct {
	source *text.GoTextFaceSource
	face   *text.GoTextFace
	scale  float64
	touch  bool // a touch was seen: bigger rows, touch wording

	layout layout
}

// layout holds the rectangles of the overlays, in real pixels.
type layout struct {
	window      image.Rectangle
	panel       image.Rectangle // the drawer's or settings panel's background
	search      image.Rectangle // the drawer's search field
	list        image.Rectangle // the scrolling rows
	content     image.Rectangle // below the search line, full width (settings)
	details     image.Rectangle // the selected ROM, on wide screens; else empty
	statusLeft  image.Rectangle // below the picture: tap for the ROM list
	statusRight image.Rectangle // below the picture: tap for settings
	rowH        float64
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

// computeLayout places the panel over the whole window, leaving a line at
// the top (debug) and two at the bottom (status). Inside: the search field,
// the list, and on wide screens a details pane beside the list.
func computeLayout(window, screen image.Rectangle, scale, lineH float64, touch bool) layout {
	pad := int(panelPad * scale)
	statusH := int(2 * lineH)

	rowH := lineH
	if touch {
		rowH = max(rowH, touchRowSize*scale)
	}

	panel := image.Rect(window.Min.X+pad, window.Min.Y+int(lineH)+pad, window.Max.X-pad, window.Max.Y-statusH)
	if panel.Dy() < 4*int(rowH) {
		panel = window
	}

	inner := panel.Inset(pad)
	search := image.Rect(inner.Min.X, inner.Min.Y, inner.Max.X, inner.Min.Y+int(rowH))
	list := image.Rect(inner.Min.X, search.Max.Y+pad/2, inner.Max.X, inner.Max.Y)

	content := list

	var details image.Rectangle

	if float64(inner.Dx()) >= detailsMinWide*scale {
		split := inner.Max.X - int(float64(inner.Dx())*detailsShare)
		details = image.Rect(split+pad, list.Min.Y, inner.Max.X, list.Max.Y)
		list.Max.X = split
		search.Max.X = split
	}

	third := window.Dx() / 3
	status := image.Rect(window.Min.X, screen.Max.Y, window.Max.X, window.Max.Y)

	return layout{
		window:      window,
		panel:       panel,
		search:      search,
		list:        list,
		content:     content,
		details:     details,
		statusLeft:  image.Rect(status.Min.X, status.Min.Y, status.Min.X+third, status.Max.Y),
		statusRight: image.Rect(status.Max.X-third, status.Min.Y, status.Max.X, status.Max.Y),
		rowH:        rowH,
	}
}

func (u *ui) draw(dst *ebiten.Image, g *Game) {
	screen := g.renderer.Screen()

	switch g.state {
	case stateOff:
		u.centred(dst, screen, "Tap, click or press a key to switch on", inkColour)
	case statePicker:
		u.drawPicker(dst, g)
	case stateSettings:
		u.drawSettings(dst, g)
	case stateLoading:
		u.centred(dst, screen, "Tuning in "+g.loadingEntry.Title+"…", inkColour)
	case stateRunning:
		u.drawRunningStatus(dst, g, screen)
	}

	if g.state == statePicker || g.state == stateSettings {
		x := float64(u.layout.panel.Min.X)
		y := float64(u.layout.panel.Max.Y) + u.lineHeight()/2
		maxW := float64(u.layout.panel.Dx())

		if g.message != "" {
			u.text(dst, u.fit(g.message, maxW), x, y, errorColour)
		} else {
			u.text(dst, u.fit(u.hint(g), maxW), x, y, mutedColour)
		}
	}

	if g.debug {
		msg := fmt.Sprintf("TPS %.0f  FPS %.0f  scale %.1f×  %d×%d",
			ebiten.ActualTPS(), ebiten.ActualFPS(), g.scale, g.width, g.height)
		u.text(dst, msg, panelPad*u.scale, panelPad*u.scale/2, mutedColour)
	}
}

// drawRunningStatus draws the line under the picture: "ROM list" on the
// left, the title (or a message) in the middle, "Settings" on the right.
func (u *ui) drawRunningStatus(dst *ebiten.Image, g *Game, screen image.Rectangle) {
	y := float64(screen.Max.Y) + u.lineHeight()/2
	left, right := float64(screen.Min.X), float64(screen.Max.X)

	u.text(dst, "< ROM list", left, y, accentColour)

	w, _ := text.Measure("Settings >", u.face, 0)
	u.text(dst, "Settings >", right-w, y, accentColour)

	mid, col := g.current.Title, mutedColour
	if g.message != "" {
		mid, col = g.message, errorColour
	}

	lw, _ := text.Measure("< ROM list", u.face, 0)
	gap := panelPad * u.scale
	space := right - left - lw - w - 2*gap

	if !u.touch && g.message == "" {
		hint := "  ·  Esc: ROM list  ·  F2: settings  ·  F1: stats"
		if tw, _ := text.Measure(mid+hint, u.face, 0); tw <= space {
			mid += hint
		}
	}

	mid = u.fit(mid, space)
	mw, _ := text.Measure(mid, u.face, 0)
	u.text(dst, mid, left+lw+gap+(space-mw)/2, y, col)

	// The second line shows the game's keys; a message that didn't fit
	// goes on the third.
	if keys := u.keysLine(g); keys != "" {
		k := u.fit(keys, right-left)
		kw, _ := text.Measure(k, u.face, 0)
		u.text(dst, k, left+(right-left-kw)/2, y+u.lineHeight(), inkColour)
	}

	if g.message != "" && mid != g.message {
		u.text(dst, u.fit(g.message, right-left), left, y+2*u.lineHeight(), errorColour)
	}
}

// keysLine describes the running game's controls: the database's hint if
// it has one, otherwise the keys the game has checked so far.
func (u *ui) keysLine(g *Game) string {
	if len(g.current.Keys) > 0 {
		return "Keys: " + input.DescribeHints(g.current.Keys)
	}

	mask, anyKey := g.machine.KeysRead()

	switch {
	case mask != 0:
		return "Keys it reads: " + input.DescribeMask(mask)
	case anyKey:
		return "Press any key (1–4, Q–R, A–F, Z–V)"
	default:
		return ""
	}
}

// hint is the line under the drawer or settings panel.
func (u *ui) hint(g *Game) string {
	if g.state == stateSettings {
		if u.touch {
			return "Tap a setting to change it  ·  tap outside to close"
		}

		return "↑↓ choose  ·  ←→ or Enter: change  ·  Esc: close"
	}

	hint := "↑↓ choose  ·  Enter: play  ·  type to search"
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
}

func (u *ui) fillPanel(dst *ebiten.Image) {
	p := u.layout.panel
	vector.FillRect(dst, float32(p.Min.X), float32(p.Min.Y), float32(p.Dx()), float32(p.Dy()), panelColour, false)
}

func (u *ui) drawPicker(dst *ebiten.Image, g *Game) {
	l := u.layout
	u.fillPanel(dst)
	g.search.draw(dst)

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

	if len(p.entries) == 0 {
		// Below the load-file row, where the first result would be.
		y := float64(l.list.Min.Y) + float64(len(p.rows))*l.rowH - p.offset
		u.text(list, u.fit("Nothing matches “"+p.query+"”.", maxW), x, y+textOffset, mutedColour)
	}

	for r, row := range p.rows {
		y := float64(l.list.Min.Y) + float64(r)*l.rowH - p.offset
		if y+l.rowH < float64(l.list.Min.Y) || y > float64(l.list.Max.Y) {
			continue
		}

		switch row.kind {
		case rowHeading:
			u.text(list, u.fit(row.heading, maxW), x, y+textOffset, mutedColour)
		case rowLoadFile:
			u.text(list, u.fit("+ Load a ROM from this device…", maxW), x, y+textOffset, accentColour)
		case rowEntry:
			u.drawEntryRow(list, p, row.entry, x, y, maxW, textOffset)
		}
	}

	if !l.details.Empty() {
		if e, ok := p.current(); ok {
			u.drawDetails(dst, e)
		}
	}
}

func (u *ui) drawEntryRow(list *ebiten.Image, p *picker, i int, x, y, maxW, textOffset float64) {
	e := p.entries[i]
	col := inkColour

	if !e.Playable() {
		col = mutedColour
	}

	if i == p.selected && !u.touch {
		vector.FillRect(list, float32(x), float32(y), float32(maxW), float32(u.layout.rowH), selectedColour, false)

		col = accentColour
	}

	label := "  " + e.Title
	if !e.Playable() {
		label += "  —  " + e.Platform.Name()
	}

	u.text(list, u.fit(label, maxW), x, y+textOffset, col)
}

// drawDetails shows the selected ROM beside the list on wide screens.
func (u *ui) drawDetails(dst *ebiten.Image, e library.Entry) {
	d := u.layout.details
	x, y := float64(d.Min.X), float64(d.Min.Y)
	maxW := float64(d.Dx())
	lh := u.lineHeight()

	u.text(dst, u.fit(e.Title, maxW), x, y, accentColour)
	y += lh
	u.text(dst, u.fit(e.Subtitle(), maxW), x, y, mutedColour)
	y += lh

	keys := "Keys: shown under the picture once the game reads them."
	if len(e.Keys) > 0 {
		keys = "Keys: " + input.DescribeHints(e.Keys)
	}

	u.text(dst, u.fit(keys, maxW), x, y, inkColour)
	y += 2 * lh

	status := "Downloads when you play it, then works offline."

	switch {
	case !e.Playable():
		status = "Coming soon: needs " + e.Platform.Name() + "."
	case e.Offline:
		status = "Plays offline."
	}

	lines := u.wrap(e.Description, maxW)
	room := int((float64(d.Max.Y)-y)/lh) - 2

	for i, line := range lines {
		if i >= room {
			break
		}

		if i == room-1 && len(lines) > room {
			line = u.fit(line+" …", maxW)
		}

		u.text(dst, line, x, y, inkColour)
		y += lh
	}

	u.text(dst, u.fit(status, maxW), x, float64(d.Max.Y)-lh, mutedColour)
}

func (u *ui) drawSettings(dst *ebiten.Image, g *Game) {
	l := u.layout
	u.fillPanel(dst)

	s := g.settings
	u.text(dst, u.fit("Settings: "+g.current.Title, float64(l.content.Dx())), float64(l.search.Min.X),
		float64(l.search.Min.Y)+(l.rowH-u.lineHeight())/2, accentColour)

	textOffset := (l.rowH - u.lineHeight()) / 2
	x, right := float64(l.content.Min.X), float64(l.content.Max.X)

	for i := range s.rows {
		y := float64(l.content.Min.Y) + float64(i)*l.rowH
		if y+l.rowH > float64(l.content.Max.Y) {
			break
		}

		lbl := s.label(i)
		name, value, changed := lbl.name, lbl.value, lbl.changed
		col := inkColour

		if i == s.selected && !u.touch {
			vector.FillRect(dst, float32(x), float32(y), float32(right-x), float32(l.rowH), selectedColour, false)

			col = accentColour
		}

		if s.steps(i) {
			value = "‹  " + value + "  ›"
		}

		if changed {
			value += " *"
		}

		vw, _ := text.Measure(value, u.face, 0)
		u.text(dst, u.fit(name, right-x-vw-panelPad*u.scale), x, y+textOffset, col)

		vc := mutedColour
		if changed {
			vc = accentColour
		}

		u.text(dst, value, right-vw, y+textOffset, vc)
	}

	u.drawTrace(dst, g, float64(l.content.Min.Y)+float64(len(s.rows))*l.rowH+u.lineHeight()/2)
}

// drawTrace shows the last instructions when tracing is on, below the
// settings rows, if there is room.
func (u *ui) drawTrace(dst *ebiten.Image, g *Game, y float64) {
	tr := g.machine.Trace()
	if tr == nil {
		return
	}

	entries := tr.Entries()
	room := min(int((float64(u.layout.content.Max.Y)-y)/u.lineHeight()), traceLines, len(entries))

	if room <= 0 {
		return
	}

	trace := chip8.FormatTrace(g.machine.Profile().Platform, entries[len(entries)-room:])

	lines := strings.SplitSeq(strings.TrimRight(trace, "\n"), "\n")
	for line := range lines {
		u.text(dst, u.fit(line, float64(u.layout.content.Dx())), float64(u.layout.content.Min.X), y, mutedColour)
		y += u.lineHeight()
	}
}

// wrap splits s into lines no wider than maxW, at spaces.
func (u *ui) wrap(s string, maxW float64) []string {
	var lines []string

	for para := range strings.SplitSeq(s, "\n") {
		line := ""

		for word := range strings.FieldsSeq(para) {
			candidate := word
			if line != "" {
				candidate = line + " " + word
			}

			if w, _ := text.Measure(candidate, u.face, 0); w > maxW && line != "" {
				lines = append(lines, line)
				candidate = word
			}

			line = candidate
		}

		if line != "" {
			lines = append(lines, u.fit(line, maxW))
		}
	}

	return lines
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
