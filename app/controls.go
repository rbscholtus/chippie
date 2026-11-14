package app

import (
	"fmt"
	"image"
	"math"
	"slices"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/rbscholtus/chippie/app/crt"
	"github.com/rbscholtus/chippie/app/input"
	"github.com/rbscholtus/chippie/app/library"
	"github.com/rbscholtus/chippie/app/skin"
)

// This file holds what the TV's controls do: the channel dial, the speed
// (UHF fine-tuning), volume and tint knobs and the power switch, by click,
// drag, wheel and keyboard (README §11).

// Control ids in skin.json.
const (
	controlChannel = "channel"
	controlSpeed   = "speed"
	controlVolume  = "volume"
	controlTint    = "tint"
	controlPower   = "power"
)

// Control feel, in CSS pixels or fractions of a knob's range.
const (
	knobDragRange  = 160 // vertical drag for a knob's full range
	tintDragStep   = 40  // vertical drag per tint preset
	volumeStep     = 0.1 // one click or wheel notch
	defaultVolume  = 0.7 // before the user turns it
	minTouchTarget = 22  // radius: Apple's 44 pt minimum touch target
	volumePrefKey  = "volume"
	tintPrefPrefix = "tint/"
)

// tintPreset is a picture colour scheme on the tint knob; a nil palette
// means the ROM's own colours.
type tintPreset struct {
	name    string
	palette *crt.Palette
}

// rgb splits a 0xRRGGBB colour into opaque RGBA bytes.
func rgb(v uint32) [4]uint8 {
	return [4]uint8{uint8(v >> 16), uint8(v >> 8), uint8(v), 0xFF} //nolint:gosec // G115: 24-bit colour
}

func paletteOf(off, one, two, both uint32) *crt.Palette {
	p := crt.Palette{}

	for i, v := range []uint32{off, one, two, both} {
		c := rgb(v)
		p[i].R, p[i].G, p[i].B, p[i].A = c[0], c[1], c[2], c[3]
	}

	return &p
}

// tints are the tint knob's positions, left to right.
var tints = []tintPreset{ //nolint:gochecknoglobals // read-only table
	{name: "ROM colours"},
	{name: "White", palette: &crt.DefaultPalette},
	{name: "Amber", palette: paletteOf(0x000000, 0xFFB000, 0x9A6400, 0xFFD88A)},
	{name: "Green", palette: paletteOf(0x000000, 0x40FF70, 0x1E8C3C, 0xB4FFC6)},
	{name: "Octo", palette: paletteOf(0x996600, 0xFFCC00, 0xFF6600, 0x662200)},
}

// controlState is where the controls are.
type controlState struct {
	channel int     // click-stops the channel dial has turned
	volume  float64 // 0 to 1
	speed   float64 // the speed knob's position, 0 to 1
	tint    int     // index into tints
	on      bool    // the power switch
}

func defaultControls() controlState { return controlState{volume: defaultVolume} }

// angle is a round control's rotation, in degrees, for its state.
func (s controlState) angle(c skin.Control) float64 {
	switch c.ID {
	case controlChannel:
		return skin.DialAngle(c, s.channel)
	case controlVolume:
		return skin.KnobAngle(c, s.volume)
	case controlSpeed:
		return skin.KnobAngle(c, s.speed)
	case controlTint:
		return skin.KnobAngle(c, float64(s.tint)/float64(len(tints)-1))
	default:
		return 0
	}
}

// grab is a press on a control, followed until it is released.
type grab struct {
	control   skin.Control
	lastAngle float64 // the dial: the pointer's last direction from its centre
	turn      skin.Turn
	dragY     float64 // knobs: vertical drag not yet used
}

// minHit is the smallest hit radius, in cabinet pixels.
func (g *Game) minHit() float64 { return minTouchTarget * g.scale / g.tv.lay.k }

// controlsLive reports whether the controls respond: once the art shows,
// while the TV is on and no panel covers it.
func (g *Game) controlsLive() bool {
	return g.tv.art != nil && (g.state == stateRunning || g.state == stateLoading)
}

// updateControls handles the pointer, the wheel and the keyboard for the
// controls. It reports whether the pointer input was used, so nothing else
// acts on it.
func (g *Game) updateControls(in pointerInput) bool {
	if !g.controlsLive() {
		g.grab = nil

		return false
	}

	g.controlKeys()
	g.controlWheel(in.pos)

	switch {
	case in.pressed:
		x, y := g.tv.lay.toCabinet(in.pos)
		c, ok := g.tv.skin.HitTest(x, y, g.minHit())

		if !ok {
			return false
		}

		g.grab = &grab{control: c, lastAngle: skin.Angle(c, x, y)}

		return true
	case g.grab == nil:
		return false
	case in.tapped:
		x, _ := g.tv.lay.toCabinet(in.tapAt)
		g.clickControl(g.grab.control, skin.ClickDirection(g.grab.control, x))
		g.grab = nil

		return true
	case !g.ptr.down:
		g.grab = nil // released after a drag

		return true
	case g.ptr.dragging:
		g.dragControl(in)
	}

	return true
}

// dragControl turns the grabbed control: the dial follows the pointer
// around its centre, knobs follow vertical movement (up is more).
func (g *Game) dragControl(in pointerInput) {
	gr := g.grab
	c := gr.control

	if c.Kind == skin.Dial {
		x, y := g.tv.lay.toCabinet(in.pos)
		a := skin.Angle(c, x, y)
		steps := gr.turn.Add(skin.AngleDelta(gr.lastAngle, a), c.Steps)
		gr.lastAngle = a

		g.stepChannel(steps)

		return
	}

	gr.dragY -= float64(in.dy) / g.scale

	switch c.ID {
	case controlVolume:
		g.setVolume(g.tv.controls.volume + gr.dragY/knobDragRange)
		gr.dragY = 0
	case controlSpeed:
		per := knobDragRange / float64(len(speedSteps)-1)
		for ; gr.dragY >= per; gr.dragY -= per {
			g.stepSpeed(1)
		}

		for ; gr.dragY <= -per; gr.dragY += per {
			g.stepSpeed(-1)
		}
	case controlTint:
		for ; gr.dragY >= tintDragStep; gr.dragY -= tintDragStep {
			g.stepTint(1)
		}

		for ; gr.dragY <= -tintDragStep; gr.dragY += tintDragStep {
			g.stepTint(-1)
		}
	}
}

// clickControl is a click or tap: one step (left half down, right half
// up), or the power switch flipping.
func (g *Game) clickControl(c skin.Control, dir int) {
	switch c.ID {
	case controlChannel:
		g.stepChannel(dir)
	case controlVolume:
		g.setVolume(g.tv.controls.volume + float64(dir)*volumeStep)
	case controlSpeed:
		g.stepSpeed(dir)
	case controlTint:
		g.stepTint(1) // tint cycles; a click always moves on
	case controlPower:
		g.powerOff()
	}
}

// controlWheel turns the control under the pointer one step per notch.
func (g *Game) controlWheel(pos image.Point) {
	_, wy := ebiten.Wheel()
	if wy == 0 {
		return
	}

	x, y := g.tv.lay.toCabinet(pos)

	c, ok := g.tv.skin.HitTest(x, y, 0)
	if !ok {
		return
	}

	dir := 1
	if wy < 0 {
		dir = -1
	}

	if c.ID == controlTint {
		g.stepTint(dir)
	} else if c.ID != controlPower {
		g.clickControl(c, dir)
	}
}

// controlKeys: PageUp/PageDown channel, -/= volume, [/] speed, T tint,
// P power. None of these is on the CHIP-8 keypad.
func (g *Game) controlKeys() {
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyPageUp):
		g.stepChannel(1)
	case inpututil.IsKeyJustPressed(ebiten.KeyPageDown):
		g.stepChannel(-1)
	case pressedOrRepeat(ebiten.KeyEqual):
		g.setVolume(g.tv.controls.volume + volumeStep)
	case pressedOrRepeat(ebiten.KeyMinus):
		g.setVolume(g.tv.controls.volume - volumeStep)
	case pressedOrRepeat(ebiten.KeyBracketRight):
		g.stepSpeed(1)
	case pressedOrRepeat(ebiten.KeyBracketLeft):
		g.stepSpeed(-1)
	case inpututil.IsKeyJustPressed(ebiten.KeyT):
		g.stepTint(1)
	case inpututil.IsKeyJustPressed(ebiten.KeyP):
		g.powerOff()
	}
}

// stepChannel moves n games through the library's current list (its
// search), skipping games that can't play yet, and tunes in.
func (g *Game) stepChannel(n int) {
	if n == 0 {
		return
	}

	from := g.current.SHA1
	if g.state == stateLoading {
		from = g.loadingEntry.SHA1
	}

	e, ch, ok := g.picker.neighbour(from, n)
	if !ok {
		return
	}

	g.tv.controls.channel += n
	g.picker.selectSHA1(e.SHA1)

	if err := g.play(e); err != nil {
		g.message = err.Error()
	}

	g.osd.show(fmt.Sprintf("CH %d", ch), e.Title)
}

// setVolume sets, saves and shows the volume.
func (g *Game) setVolume(v float64) {
	v = math.Round(min(max(v, 0), 1)*100) / 100
	g.tv.controls.volume = v
	g.sound.SetVolume(v)
	g.Prefs.Set(volumePrefKey, strconv.FormatFloat(v, 'f', -1, 64))
	g.osd.showBar("VOLUME", v)
}

// loadVolume restores the saved volume.
func (g *Game) loadVolume() {
	v := defaultVolume

	if s, ok := g.Prefs.Get(volumePrefKey); ok {
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			v = min(max(f, 0), 1)
		}
	}

	g.tv.controls.volume = v
	g.sound.SetVolume(v)
}

// stepSpeed moves the running game's speed one step, like the settings
// panel's speed row, and saves it with the game's settings.
func (g *Game) stepSpeed(dir int) {
	if !g.hasROM {
		return
	}

	p := g.machine.Profile()
	p.TicksPerFrame = stepSpeed(p.TicksPerFrame, dir)
	g.machine.SetTicksPerFrame(p.TicksPerFrame)
	g.saveOverrides(p)
	g.syncSpeedKnob()
	g.osd.show(fmt.Sprintf("SPEED %d", p.TicksPerFrame))
}

// syncSpeedKnob turns the speed knob to the running game's speed.
func (g *Game) syncSpeedKnob() {
	n := g.machine.Profile().TicksPerFrame
	i := slices.IndexFunc(speedSteps, func(s int) bool { return s >= n })

	if i < 0 {
		i = len(speedSteps) - 1
	}

	g.tv.controls.speed = float64(i) / float64(len(speedSteps)-1)
}

// stepTint moves the tint knob, wrapping, and applies and saves it for the
// running game.
func (g *Game) stepTint(dir int) {
	t := ((g.tv.controls.tint+dir)%len(tints) + len(tints)) % len(tints)
	g.tv.controls.tint = t

	if g.hasROM {
		g.Prefs.Set(tintPrefPrefix+g.current.SHA1, strconv.Itoa(t))
		g.renderer.SetPalette(g.paletteWithTint(g.current.SHA1))
		g.renderer.Update(g.machine.Display())
	}

	g.osd.show("TINT " + tints[t].name)
}

// loadTint restores a game's saved tint.
func (g *Game) loadTint(sha1 string) {
	g.tv.controls.tint = 0

	if s, ok := g.Prefs.Get(tintPrefPrefix + sha1); ok {
		if t, err := strconv.Atoi(s); err == nil && t >= 0 && t < len(tints) {
			g.tv.controls.tint = t
		}
	}
}

// paletteWithTint is the game's palette through the tint knob.
func (g *Game) paletteWithTint(sha1 string) crt.Palette {
	if p := tints[g.tv.controls.tint].palette; p != nil {
		return *p
	}

	return g.paletteFor(sha1)
}

// powerOff switches the TV off: the picture collapses and the game pauses.
// Any click, tap or key switches it back on (Update, stateOff).
func (g *Game) powerOff() {
	g.state = stateOff
	g.sound.SetBuzzer(false)
	input.Release(g.machine)
	g.osd.clear()
}

// neighbour returns the playable entry n steps from sha1 in the current
// list, wrapping, and its channel number: its place among the playable
// entries, from 1. From an unknown sha1 it counts from before the first.
func (p *picker) neighbour(sha1 string, n int) (library.Entry, int, bool) {
	var playable []library.Entry

	at := -1

	for _, e := range p.entries {
		if !e.Playable() {
			continue
		}

		if e.SHA1 == sha1 {
			at = len(playable)
		}

		playable = append(playable, e)
	}

	if len(playable) == 0 {
		return library.Entry{}, 0, false
	}

	if at < 0 && n > 0 {
		n-- // the first step lands on the first entry
	}

	i := ((max(at, 0)+n)%len(playable) + len(playable)) % len(playable)

	return playable[i], i + 1, true
}
