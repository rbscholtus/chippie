package app

import (
	"context"
	"fmt"
	"image"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/rbscholtus/chippie/app/input"
	"github.com/rbscholtus/chippie/app/roms"
	"github.com/rbscholtus/chippie/chip8"
)

// This file holds the game's flow: loading ROMs, running frames, the
// picker, power and input helpers. game.go has the ebiten.Game methods.

// startROM starts loading a library entry by ID or, natively, a ROM file.
func (g *Game) startROM(id string) error {
	if strings.HasPrefix(id, "game/") || strings.HasPrefix(id, "test/") {
		e, ok := g.lib.Find(id)
		if !ok {
			return fmt.Errorf("%w: %s", errUnknownROM, id)
		}

		g.picker.selectID(id)

		return g.load(e)
	}

	e, rom, err := roms.ReadFile(id)
	if err != nil {
		return fmt.Errorf("open %s: %w", id, err)
	}

	if g.params.Platform != "" {
		p, ok := chip8.ParsePlatform(g.params.Platform)
		if !ok {
			return fmt.Errorf("%w %q", errUnknownPlatform, g.params.Platform)
		}

		e.Platform = p
	}

	return g.run(e, rom)
}

// load starts fetching an entry in the background.
func (g *Game) load(e roms.Entry) error {
	if !e.Playable() {
		return fmt.Errorf("%s: %w: it needs %s, coming in a later version", e.Title, errNotPlayable, e.Platform.Name())
	}

	g.message = ""
	g.state = stateLoading
	g.loading = g.lib.LoadAsync(context.Background(), e)

	return nil
}

// run starts emulating a loaded ROM.
func (g *Game) run(e roms.Entry, rom []byte) error {
	g.machine.Reset(e.Profile())

	if err := g.machine.LoadROM(rom); err != nil {
		return fmt.Errorf("%s: %w", e.Title, err)
	}

	g.current, g.hasROM = e, true
	g.message = ""
	g.state = stateRunning
	g.renderer.Update(g.machine.Display())

	return nil
}

// runFrame runs one emulated frame, unless the game should pause.
func (g *Game) runFrame() {
	if !shouldRun(g.state, ebiten.IsFocused(), g.machine.Halted()) {
		input.Release(g.machine)
		g.sound.SetBuzzer(false)

		return
	}

	input.Poll(g.machine)
	g.machine.RunFrame()
	g.sound.SetBuzzer(g.machine.Sound().Buzzer)
	g.renderer.Update(g.machine.Display())

	if err := g.machine.State().Err; err != nil {
		g.message = "Halted: " + err.Error() + ". Esc or tap here: ROM list."
	}
}

func (g *Game) pollLoading() {
	if g.loading == nil {
		return
	}

	select {
	case res := <-g.loading:
		g.loading = nil

		if res.Err != nil {
			g.message = res.Err.Error()
			g.state = statePicker

			return
		}

		if err := g.run(res.Entry, res.ROM); err != nil {
			g.message = err.Error()
			g.state = statePicker
		}
	default:
	}
}

func (g *Game) openPicker() {
	g.state = statePicker
	g.sound.SetBuzzer(false)
	input.Release(g.machine)
}

// relayout places the picture and the overlays for the current size.
func (g *Game) relayout() {
	screen := screenRect(g.width, g.height)
	g.renderer.SetScreen(screen)
	g.ui.resize(image.Rect(0, 0, g.width, g.height), screen, g.scale)
	g.picker.setView(float64(g.ui.layout.list.Dy()), g.ui.layout.rowH)
}

func (g *Game) updatePicker(in pointerInput) {
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyArrowDown):
		g.picker.move(1)
	case inpututil.IsKeyJustPressed(ebiten.KeyArrowUp):
		g.picker.move(-1)
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
		g.pick()
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape) && g.hasROM:
		g.state = stateRunning
	}

	// Drag and wheel scroll the list.
	g.picker.scroll(-float64(in.dy))

	if _, wy := ebiten.Wheel(); wy != 0 {
		g.picker.scroll(-wy * g.ui.layout.rowH)
	}

	if !in.tapped {
		return
	}

	l := g.ui.layout

	switch {
	case in.tapAt.In(l.list):
		if i, ok := g.picker.entryAt(float64(in.tapAt.Y - l.list.Min.Y)); ok && g.picker.selectIndex(i) {
			g.pick()
		}
	case in.tapAt.Y >= l.panel.Max.Y && g.hasROM:
		g.state = stateRunning // tapped the status line: back to the ROM
	}
}

func (g *Game) pick() {
	if e, ok := g.picker.current(); ok {
		if err := g.load(e); err != nil {
			g.message = err.Error()
		}
	}
}

func (g *Game) animatePower() {
	target := float32(0)
	if g.state != stateOff {
		target = 1
	}

	switch {
	case g.power < target:
		g.power = min(target, g.power+powerStep)
	case g.power > target:
		g.power = max(target, g.power-powerStep)
	}

	g.renderer.SetPower(g.power)
}

// pointerInput is this tick's mouse or touch input, already turned into
// taps and drags by the game's pointer tracker.
type pointerInput struct {
	pressed bool        // a press started this tick
	touch   bool        // the input came from a touch screen
	pos     image.Point // where the pointer is (or was released)
	dy      int         // vertical drag movement this tick
	tapped  bool        // a tap ended this tick
	tapAt   image.Point // where the tap started
}

// readPointer reads one finger or the left mouse button and feeds the
// pointer tracker.
func (g *Game) readPointer() pointerInput {
	var (
		in             pointerInput
		held, released bool
	)

	switch {
	case g.touching:
		in.touch = true

		if inpututil.IsTouchJustReleased(g.touchID) {
			released, g.touching = true, false
			in.pos = image.Pt(inpututil.TouchPositionInPreviousTick(g.touchID))
		} else {
			held = true
			in.pos = image.Pt(ebiten.TouchPosition(g.touchID))
		}
	default:
		g.touchBuf = inpututil.AppendJustPressedTouchIDs(g.touchBuf[:0])
		if len(g.touchBuf) > 0 {
			g.touchID, g.touching = g.touchBuf[0], true
			in.pressed, in.touch = true, true
			in.pos = image.Pt(ebiten.TouchPosition(g.touchID))

			break
		}

		in.pos = image.Pt(ebiten.CursorPosition())
		in.pressed = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
		released = inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)
		held = !in.pressed && !released && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	}

	switch {
	case in.pressed:
		g.ptr.press(in.pos)
	case held:
		in.dy = g.ptr.move(in.pos)
	case released:
		in.tapAt, in.tapped = g.ptr.release(in.pos)
	}

	return in
}
