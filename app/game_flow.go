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
		g.message = "Halted: " + err.Error() + ". Press Esc to pick another ROM."
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

func (g *Game) updatePicker() {
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

	if pt, ok := g.clicked(); ok {
		if i, hit := g.ui.entryAt(pt); hit && g.picker.selectIndex(i) {
			g.pick()
		}
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

// gesture reports a click, tap or key press this tick: browsers allow audio
// only after one.
func (g *Game) gesture() bool {
	if _, ok := g.clicked(); ok {
		return true
	}

	return len(inpututil.AppendJustPressedKeys(nil)) > 0
}

// clicked returns where the mouse or a finger was pressed this tick.
func (g *Game) clicked() (image.Point, bool) {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()

		return image.Pt(x, y), true
	}

	if ids := inpututil.AppendJustPressedTouchIDs(nil); len(ids) > 0 {
		x, y := ebiten.TouchPosition(ids[0])

		return image.Pt(x, y), true
	}

	return image.Point{}, false
}
