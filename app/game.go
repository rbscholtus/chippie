// Package app is the Ebitengine game: it runs the emulator once per Update,
// draws it through the CRT renderer, plays the buzzer and shows the ROM
// picker. Game logic lives in chip8; this package only connects it to the
// screen, keyboard and speakers.
package app

import (
	"errors"
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/rbscholtus/chippie/app/crt"
	"github.com/rbscholtus/chippie/app/roms"
	"github.com/rbscholtus/chippie/app/sound"
	"github.com/rbscholtus/chippie/chip8"
	"github.com/rbscholtus/chippie/platform"
)

// state is what the TV is doing.
type state uint8

const (
	stateOff     state = iota // waiting for the first click or key
	statePicker               // the ROM list is open; emulation paused
	stateLoading              // a ROM is being fetched
	stateRunning              // emulating
)

// Power animation speed: the full on/off animation takes 40 updates.
const powerStep = 1.0 / 40

// Game is the ebiten.Game.
type Game struct {
	params   platform.Params
	lib      *roms.Library
	machine  *chip8.Machine
	current  roms.Entry
	hasROM   bool
	renderer *crt.Renderer
	sound    sound.Output
	picker   *picker
	ui       *ui

	state   state
	loading <-chan roms.Result
	power   float32
	message string
	debug   bool

	width, height int // layout size in real pixels
	scale         float64
}

// New creates the game. A ROM given in the parameters starts loading
// straight away; otherwise the TV waits to be switched on.
func New(p platform.Params, lib *roms.Library) (*Game, error) {
	mode := crt.ModeCRT
	if p.Shader == platform.ShaderClean {
		mode = crt.ModeClean
	}

	r, err := crt.New(mode)
	if err != nil {
		return nil, fmt.Errorf("create renderer: %w", err)
	}

	u, err := newUI()
	if err != nil {
		return nil, err
	}

	g := &Game{
		params:   p,
		lib:      lib,
		machine:  chip8.New(chip8.DefaultProfile(chip8.ModernChip8)),
		renderer: r,
		picker:   newPicker(lib.Entries()),
		ui:       u,
		debug:    p.Debug,
		scale:    1,
	}
	g.renderer.Update(g.machine.Display())

	if p.ROM != "" {
		g.power = 1
		g.renderer.SetPower(1)

		if err := g.startROM(p.ROM); err != nil {
			g.message = err.Error()
			g.state = statePicker
		}
	}

	return g, nil
}

// Errors shown to the user.
var (
	errUnknownROM      = errors.New("unknown ROM")
	errUnknownPlatform = errors.New("unknown platform")
	errNotPlayable     = errors.New("not playable yet")
)

// Update runs one 60 Hz tick: input, then at most one emulated frame.
func (g *Game) Update() error {
	if g.gesture() && !g.sound.Started() {
		if err := g.sound.Start(); err != nil {
			g.message = err.Error()
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		g.debug = !g.debug
	}

	g.pollLoading()

	switch g.state {
	case stateOff:
		if g.gesture() {
			g.state = statePicker
		}
	case statePicker:
		g.updatePicker()
	case stateLoading:
	case stateRunning:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.openPicker()

			break
		}

		g.runFrame()
	}

	g.animatePower()

	return nil
}

// shouldRun reports whether the emulator runs this tick: only while a ROM
// runs, the window has focus and the machine hasn't halted.
func shouldRun(s state, focused, halted bool) bool {
	return s == stateRunning && focused && !halted
}

// Layout returns the real pixel size (outside size × device scale), so the
// picture is sharp on high-density screens.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	g.scale = ebiten.Monitor().DeviceScaleFactor()
	w := int(float64(outsideWidth) * g.scale)
	h := int(float64(outsideHeight) * g.scale)

	if w != g.width || h != g.height {
		g.width, g.height = w, h
		g.renderer.SetScreen(screenRect(w, h))
		g.ui.resize(g.scale)
	}

	return w, h
}

// screenRect is the largest 2:1 rectangle (the CHIP-8 picture's shape)
// centred in the window, with a margin and room for a status line below.
func screenRect(w, h int) image.Rectangle {
	const (
		margin    = 0.06
		statusGap = 0.08
		aspect    = 2
	)

	availW := float64(w) * (1 - 2*margin)
	availH := float64(h) * (1 - 2*margin - statusGap)

	sw := min(availW, availH*aspect)
	sh := sw / aspect
	x := (float64(w) - sw) / 2
	y := (float64(h) - sh) / 2 * (1 - statusGap)

	return image.Rect(int(x), int(y), int(x+sw), int(y+sh))
}

// Draw draws the picture, then any overlay.
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(roomColour)
	g.renderer.Draw(screen, uint64(ebiten.Tick())) //nolint:gosec // G115: Tick is never negative
	g.ui.draw(screen, g)
}
