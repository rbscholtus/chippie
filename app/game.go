// Package app is the Ebitengine game: it runs the emulator once per Update,
// draws it through the CRT renderer, plays the buzzer, and shows the
// library drawer and the settings panel. Game logic lives in chip8; this
// package connects it to the screen, keyboard, touch and speakers.
package app

import (
	"errors"
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/rbscholtus/chippie/app/crt"
	"github.com/rbscholtus/chippie/app/library"
	"github.com/rbscholtus/chippie/app/roms"
	"github.com/rbscholtus/chippie/app/skin"
	"github.com/rbscholtus/chippie/app/sound"
	"github.com/rbscholtus/chippie/assets"
	"github.com/rbscholtus/chippie/catalogue"
	"github.com/rbscholtus/chippie/chip8"
	"github.com/rbscholtus/chippie/platform"
	"github.com/rbscholtus/chippie/romstore"
)

// state is what the TV is doing.
type state uint8

const (
	stateOff      state = iota // waiting for the first click, tap or key
	statePicker                // the library drawer is open; emulation paused
	stateSettings              // the settings panel is open; emulation paused
	stateLoading               // a ROM is tuning in (downloading)
	stateRunning               // emulating
)

// Power animation speed: the full on/off animation takes 40 updates.
const powerStep = 1.0 / 40

// Errors shown to the user.
var (
	errUnknownROM      = errors.New("unknown ROM")
	errUnknownPlatform = errors.New("unknown platform")
	errNotPlayable     = errors.New("not playable yet")
)

// Deps are what the game is built from; cmd/chippie wires them up.
type Deps struct {
	Params    platform.Params
	Catalogue *catalogue.Catalogue
	Builtin   *roms.Library // ROMs that need no download
	Store     *romstore.Store
	Prefs     *platform.Prefs
	// CachedROMs, if set, lists the SHA-1s of downloaded ROMs; it may block.
	CachedROMs func() ([]string, error)
	// Refresh, if set, delivers a newer catalogue in the background.
	Refresh <-chan catalogue.RefreshResult
	// Art, if set, reads the TV's images; without it only the picture shows.
	Art ArtFetcher
	// SkinFile, if set, rereads skin.json (natively, for F5 while
	// calibrating).
	SkinFile func() ([]byte, error)
}

// Game is the ebiten.Game.
type Game struct {
	Deps

	machine    *chip8.Machine
	current    library.Entry
	currentROM []byte
	hasROM     bool

	renderer *crt.Renderer
	sound    sound.Output
	picker   *picker
	search   *textField
	settings *settings
	ui       *ui
	tv       *tv
	osd      *osd
	remote   remote
	grab     *grab // a control being turned

	state        state
	loading      <-chan romstore.Result
	loadingEntry library.Entry
	picked       <-chan platform.PickedFile
	cached       chan []string
	dropped      chan platform.PickedFile
	power        float32
	message      string
	debug        bool
	calibrate    bool
	published    pageStatus // the status last given to the page

	width, height int // layout size in real pixels
	scale         float64

	ptr      pointer
	touchID  ebiten.TouchID
	touching bool
	touchBuf []ebiten.TouchID
}

// New creates the game. A ROM given in the parameters starts loading
// straight away; otherwise the TV waits to be switched on.
func New(d Deps) (*Game, error) {
	mode := crt.ModeCRT
	if d.Params.Shader == platform.ShaderClean {
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

	t, o, err := newTVParts(d.Art, r)
	if err != nil {
		return nil, err
	}

	g := &Game{
		Deps:      d,
		machine:   chip8.New(chip8.DefaultProfile(chip8.ModernChip8)),
		renderer:  r,
		picker:    newPicker(library.Build(d.Catalogue, d.Builtin.Offline()), platform.CanPickFiles()),
		search:    newTextField("Search title, author, year or platform"),
		ui:        u,
		tv:        t,
		osd:       o,
		debug:     d.Params.Debug,
		calibrate: d.Params.Calibrate,
		scale:     1,
		ptr:       pointer{threshold: dragThreshold},
		dropped:   make(chan platform.PickedFile, 1),
		cached:    make(chan []string, 1),
	}
	g.renderer.Update(g.machine.Display())
	g.loadVolume()
	g.ui.touch = platform.TouchScreen() // the remote shows from the start on phones

	if d.CachedROMs != nil {
		go func() {
			keys, _ := d.CachedROMs() // no list just means no "offline" marks yet
			g.cached <- keys
		}()
	}

	if d.Params.ROM != "" {
		g.power = 1
		g.renderer.SetPower(1)

		if err := g.startROM(d.Params.ROM); err != nil {
			g.message = err.Error()
			g.state = statePicker
		}
	}

	return g, nil
}

// newTVParts builds the TV from the embedded skin, and its on-screen
// display.
func newTVParts(art ArtFetcher, r *crt.Renderer) (*tv, *osd, error) {
	sk, err := skin.Parse(assets.Skin)
	if err != nil {
		return nil, nil, fmt.Errorf("TV skin: %w", err)
	}

	o, err := newOSD(r)
	if err != nil {
		return nil, nil, fmt.Errorf("OSD font: %w", err)
	}

	return newTV(sk, art), o, nil
}

// Update runs one 60 Hz tick: input, background results, then at most one
// emulated frame.
func (g *Game) Update() error {
	in := g.readPointer()
	if in.touch && !g.ui.touch {
		g.ui.touch = true // bigger rows and touch wording from now on
		g.search.blur()   // on touch, the keyboard opens only when asked
		g.relayout()
	}

	gesture := in.pressed || len(inpututil.AppendJustPressedKeys(nil)) > 0
	if gesture && !g.sound.Started() {
		if err := g.sound.Start(); err != nil {
			g.message = err.Error()
		}
	}

	g.functionKeys()
	g.pollBackground()
	g.checkDrops()

	if g.tv.update() {
		g.publishLayout() // the controls work now
	}

	g.remote.update()

	wasOff := g.state == stateOff
	if g.updateControls(in) {
		in = pointerInput{touch: in.touch, pos: in.pos} // a control used it
	}

	var err error

	switch g.state {
	case stateOff:
		if gesture && wasOff {
			g.switchOn()
			g.ptr.release(in.pos) // this press switched on; it doesn't pick
		}
	case statePicker:
		err = g.updatePicker(in)
	case stateSettings:
		g.updateSettings(in)
	case stateLoading:
		g.renderer.Static(uint64(ebiten.Tick())) //nolint:gosec // G115: Tick is never negative
	case stateRunning:
		g.updateRunning(in)
	}

	g.animatePower()
	g.tv.controls.on = g.state != stateOff
	g.osd.update()

	if g.status() != g.published {
		g.publishLayout()
	}

	return err
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
		g.relayout()
	}

	return w, h
}

// Draw draws the picture, the TV around it, the remote, then any overlay.
func (g *Game) Draw(screen *ebiten.Image) {
	c := g.tv.skin.Colour()
	screen.Fill(color.RGBA{R: c[0], G: c[1], B: c[2], A: 0xFF})
	g.tv.drawBackdrop(screen)
	g.renderer.Draw(screen, uint64(ebiten.Tick())) //nolint:gosec // G115: Tick is never negative
	g.tv.draw(screen)
	g.remote.draw(screen, g.ui.face, g.keysHint())
	g.ui.draw(screen, g)

	if g.calibrate {
		g.tv.drawCalibration(screen, g.minHit())
	}
}

// keysHint is the running game's keys from the database, as a bit per hex
// key, for the remote to light up.
func (g *Game) keysHint() uint16 {
	var mask uint16

	for _, hex := range g.current.Keys {
		if hex >= 0 && hex < 16 {
			mask |= 1 << hex
		}
	}

	return mask
}
