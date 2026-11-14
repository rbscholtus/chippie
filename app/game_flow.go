package app

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"io/fs"
	"regexp"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/rbscholtus/chippie/app/crt"
	"github.com/rbscholtus/chippie/app/input"
	"github.com/rbscholtus/chippie/app/library"
	"github.com/rbscholtus/chippie/app/roms"
	"github.com/rbscholtus/chippie/app/skin"
	"github.com/rbscholtus/chippie/catalogue"
	"github.com/rbscholtus/chippie/chip8"
	"github.com/rbscholtus/chippie/platform"
	"github.com/rbscholtus/chippie/romstore"
)

// This file holds the game's flow: finding and starting ROMs, running
// frames, the drawer and settings, background results and input. game.go
// has the ebiten.Game methods.

// sha1Pattern matches a ROM hash given as the rom parameter.
var sha1Pattern = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)

// maxLocalROM bounds a dropped file.
const maxLocalROM = romstore.MaxROMSize

// startROM starts a ROM given at start-up: a SHA-1, a built-in ID
// ("game/…", "test/…") or, natively, a file path.
func (g *Game) startROM(id string) error {
	switch {
	case sha1Pattern.MatchString(id):
		return g.playHash(strings.ToLower(id))
	case strings.HasPrefix(id, "game/") || strings.HasPrefix(id, "test/"):
		sha1, ok := g.Builtin.HashFor(id)
		if !ok {
			return fmt.Errorf("%w: %s", errUnknownROM, id)
		}

		return g.playHash(sha1)
	}

	_, rom, err := roms.ReadFile(id)
	if err != nil {
		return fmt.Errorf("open %s: %w", id, err)
	}

	return g.runLocal(id, rom)
}

// playHash plays a ROM from the library by its SHA-1.
func (g *Game) playHash(sha1 string) error {
	e, ok := library.Find(g.picker.all, sha1)
	if !ok {
		e = library.LocalEntry(g.Catalogue, sha1, sha1)
	}

	g.picker.selectSHA1(sha1)

	return g.play(e)
}

// play starts fetching a library entry; the TV shows static meanwhile.
func (g *Game) play(e library.Entry) error {
	if !e.Playable() {
		return fmt.Errorf("%s: %w: it needs %s, coming in a later version", e.Title, errNotPlayable, e.Platform.Name())
	}

	g.message = ""
	g.state = stateLoading
	g.loadingEntry = e
	g.loading = g.Store.Get(context.Background(), e.SHA1)

	return nil
}

// runLocal plays a ROM from the user's device (file picker, drag and drop,
// -rom), resolved against the catalogue when it's a known ROM.
func (g *Game) runLocal(name string, rom []byte) error {
	e := library.LocalEntry(g.Catalogue, romstore.Hash(rom), baseName(name))
	if !e.Playable() {
		return fmt.Errorf("%s: %w: it needs %s, coming in a later version", e.Title, errNotPlayable, e.Platform.Name())
	}

	if g.Params.Platform != "" {
		if _, ok := chip8.ParsePlatform(g.Params.Platform); !ok {
			return fmt.Errorf("%w %q", errUnknownPlatform, g.Params.Platform)
		}
	}

	return g.run(e, rom)
}

func baseName(path string) string {
	if i := strings.LastIndexAny(path, `/\`); i >= 0 {
		return path[i+1:]
	}

	return path
}

// run starts emulating a ROM with its profile: recommended, plus the
// user's saved overrides.
func (g *Game) run(e library.Entry, rom []byte) error {
	prof := g.profileFor(e.SHA1)
	g.machine.Reset(prof)

	if err := g.machine.LoadROM(rom); err != nil {
		return fmt.Errorf("%s: %w", e.Title, err)
	}

	g.loadTint(e.SHA1)
	g.renderer.SetPalette(g.paletteWithTint(e.SHA1))
	g.renderer.SetRotation(e.Rotation)
	g.renderer.SetPicture(pictureIn(g.tv.lay.tube, e.Rotation))

	g.current, g.currentROM, g.hasROM = e, rom, true
	g.syncSpeedKnob()
	g.message = ""
	g.state = stateRunning
	g.renderer.Update(g.machine.Display())

	return nil
}

// paletteFor returns a ROM's colours from the catalogue, or the default.
func (g *Game) paletteFor(sha1 string) crt.Palette {
	if rom, ok := g.Catalogue.DB.ROM(sha1); ok && rom.Colors != nil {
		if p, ok := crt.ParsePalette(rom.Colors.Pixels); ok {
			return p
		}
	}

	return crt.DefaultPalette
}

// overridesKey is where a ROM's settings are saved.
func overridesKey(sha1 string) string { return "overrides/" + sha1 }

// overrides returns the user's saved settings for a ROM.
func (g *Game) overrides(sha1 string) catalogue.Overrides {
	var o catalogue.Overrides

	if s, ok := g.Prefs.Get(overridesKey(sha1)); ok {
		_ = json.Unmarshal([]byte(s), &o) // bad saved settings: use none
	}

	if p := g.Params.Platform; p != "" && o.Platform == "" {
		o.Platform = p // -platform / ?platform= applies until the user picks one
	}

	return o
}

// profileFor resolves the profile a ROM runs with. The random seed comes
// from the clock here, not in the core (README §6.1).
func (g *Game) profileFor(sha1 string) chip8.Profile {
	p := g.Catalogue.DB.Resolve(sha1, g.overrides(sha1)).Profile
	p.Seed = uint64(time.Now().UnixNano())

	return p
}

// saveOverrides stores what the settings panel changed, relative to the
// recommendation; "reset to recommended" leaves nothing to store.
func (g *Game) saveOverrides(p chip8.Profile) {
	o := g.Catalogue.DB.OverridesFor(g.current.SHA1, p)
	if o.Empty() {
		g.Prefs.Set(overridesKey(g.current.SHA1), "")

		return
	}

	if data, err := json.Marshal(o); err == nil {
		g.Prefs.Set(overridesKey(g.current.SHA1), string(data))
	}
}

// runFrame runs one emulated frame, unless the game should pause.
func (g *Game) runFrame() {
	if !shouldRun(g.state, ebiten.IsFocused(), g.machine.Halted()) {
		input.Release(g.machine)
		g.sound.SetBuzzer(false)

		return
	}

	input.Poll(g.machine, g.remote.held)
	g.machine.RunFrame()

	snd := g.machine.Sound()
	g.sound.SetPattern(snd.HasPattern, snd.Pattern, snd.Pitch)
	g.sound.SetBuzzer(snd.Buzzer)
	g.renderer.Update(g.machine.Display())

	if err := g.machine.State().Err; err != nil {
		g.message = "Halted: " + err.Error() + ". Open the ROM list or settings below."
	}
}

func (g *Game) updateRunning(in pointerInput) {
	l := g.ui.layout

	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape) || (in.tapped && in.tapAt.In(l.statusLeft)):
		g.openPicker()
	case inpututil.IsKeyJustPressed(ebiten.KeyF2) || inpututil.IsKeyJustPressed(ebiten.KeyTab) ||
		(in.tapped && in.tapAt.In(l.statusRight)):
		g.openSettings()
	default:
		g.runFrame()
	}
}

// pollBackground collects results from goroutines without waiting.
func (g *Game) pollBackground() {
	select {
	case res := <-g.loading:
		g.loading = nil

		if res.Err != nil {
			g.message = g.loadingEntry.Title + ": " + friendlyError(res.Err)
			g.openPicker()

			break
		}

		g.picker.markOffline(map[string]bool{res.SHA1: true})
		g.loadingEntry.Offline = true

		if err := g.run(g.loadingEntry, res.ROM); err != nil {
			g.message = err.Error()
			g.openPicker()
		}
	case keys := <-g.cached:
		set := make(map[string]bool, len(keys))
		for _, k := range keys {
			set[k] = true
		}

		g.picker.markOffline(set)
	case res := <-g.Refresh:
		g.Refresh = nil

		if res.Err == nil {
			g.Catalogue = res.Catalogue

			offline := g.Builtin.Offline()
			for _, e := range g.picker.all {
				if e.Offline {
					offline[e.SHA1] = true // keep what's cached
				}
			}

			g.picker.setAll(library.Build(res.Catalogue, offline))
		}
	case f := <-g.picked:
		g.picked = nil
		g.openLocal(f)
	case f := <-g.dropped:
		g.openLocal(f)
	default:
	}
}

// friendlyError shortens download errors for the status line.
func friendlyError(err error) string {
	if strings.Contains(err.Error(), "no download location") {
		return "no download location"
	}

	return "couldn't download it (offline?)"
}

func (g *Game) openLocal(f platform.PickedFile) {
	if f.Err != nil {
		g.message = f.Err.Error()

		return
	}

	if err := g.runLocal(f.Name, f.Data); err != nil {
		g.message = err.Error()
	}
}

// checkDrops starts reading a file dropped on the window (in a goroutine:
// reading may block in the browser).
func (g *Game) checkDrops() {
	files := ebiten.DroppedFiles()
	if files == nil {
		return
	}

	go func() {
		g.dropped <- readDropped(files)
	}()
}

// readDropped reads the first regular file of a drop.
func readDropped(fsys fs.FS) platform.PickedFile {
	var found platform.PickedFile

	_ = fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		f, err := fsys.Open(path)
		if err != nil {
			found = platform.PickedFile{Name: path, Err: err}

			return fs.SkipAll
		}
		defer f.Close()

		data, err := io.ReadAll(io.LimitReader(f, maxLocalROM+1))

		switch {
		case err != nil:
			found = platform.PickedFile{Name: path, Err: err}
		case len(data) > maxLocalROM:
			found = platform.PickedFile{Name: path, Err: fmt.Errorf("%s: %w", path, romstore.ErrUnavailable)}
		default:
			found = platform.PickedFile{Name: path, Data: data}
		}

		return fs.SkipAll
	})

	return found
}

// openPicker opens the library drawer. On a keyboard, typing searches
// straight away; on touch, the keyboard opens only when the field is
// tapped.
func (g *Game) openPicker() {
	g.state = statePicker
	g.sound.SetBuzzer(false)
	input.Release(g.machine)

	if g.ui.touch {
		g.search.blur()
	} else {
		g.search.focus()
	}

	if g.hasROM {
		g.picker.selectSHA1(g.current.SHA1)
	}
}

func (g *Game) closePicker() {
	g.search.blur()
	g.state = stateRunning
}

func (g *Game) updatePicker(in pointerInput) error {
	escape := inpututil.IsKeyJustPressed(ebiten.KeyEscape)

	switch {
	case escape && g.search.text != "":
		g.search.setText("", 0)
	case escape && g.hasROM:
		g.closePicker()

		return nil
	case pressedOrRepeat(ebiten.KeyArrowDown):
		g.picker.move(1)
	case pressedOrRepeat(ebiten.KeyArrowUp):
		g.picker.move(-1)
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
		g.pick()

		return nil
	}

	g.search.ime = g.ui.touch

	if err := g.search.update(); err != nil {
		return err
	}

	if g.search.changed {
		g.search.changed = false
		g.picker.setQuery(g.search.value())
	}

	// Drag and wheel scroll the list.
	g.picker.scroll(-float64(in.dy))

	if _, wy := ebiten.Wheel(); wy != 0 {
		g.picker.scroll(-wy * g.ui.layout.rowH)
	}

	if in.tapped {
		g.tapPicker(in.tapAt)
	}

	return nil
}

func (g *Game) tapPicker(at image.Point) {
	l := g.ui.layout

	switch {
	case at.In(l.search):
		g.search.focus()
	case at.In(l.list):
		g.search.blur()

		row, ok := g.picker.rowAt(float64(at.Y - l.list.Min.Y))
		if !ok {
			return
		}

		switch row.kind {
		case rowEntry:
			g.picker.selected = row.entry
			g.pick()
		case rowLoadFile:
			g.picked = platform.PickFile()
		case rowHeading:
		}
	case at.Y >= l.panel.Max.Y && g.hasROM:
		g.closePicker() // tapped the status line: back to the ROM
	}
}

func (g *Game) pick() {
	e, ok := g.picker.current()
	if !ok {
		return
	}

	g.search.blur()

	if err := g.play(e); err != nil {
		g.message = err.Error()
	}
}

// openSettings opens the settings panel for the running ROM.
func (g *Game) openSettings() {
	sha1 := g.current.SHA1
	own := g.Catalogue.DB.Resolve(sha1, catalogue.Overrides{}).Profile
	forPlatform := func(p chip8.Platform) chip8.Profile {
		return g.Catalogue.DB.Resolve(sha1, catalogue.Overrides{Platform: p.String()}).Profile
	}

	cur := g.machine.Profile()
	cur.Seed, own.Seed = 0, 0

	g.settings = newSettings(cur, own, forPlatform, g.machine.Trace() != nil)

	names := make([]string, len(g.skins))
	for i, s := range g.skins {
		names[i] = s.skin.Name
	}

	g.settings.setSkins(names, g.skin)
	g.state = stateSettings
	g.sound.SetBuzzer(false)
	input.Release(g.machine)
}

func (g *Game) updateSettings(in pointerInput) {
	s := g.settings

	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyF2) ||
		inpututil.IsKeyJustPressed(ebiten.KeyTab):
		g.state = stateRunning
	case pressedOrRepeat(ebiten.KeyArrowDown):
		s.move(1)
	case pressedOrRepeat(ebiten.KeyArrowUp):
		s.move(-1)
	case pressedOrRepeat(ebiten.KeyArrowLeft):
		g.applySetting(s.change(s.selected, -1))
	case pressedOrRepeat(ebiten.KeyArrowRight):
		g.applySetting(s.change(s.selected, 1))
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace):
		g.applySetting(s.change(s.selected, 0))
	}

	if !in.tapped {
		return
	}

	l := g.ui.layout
	if !in.tapAt.In(l.content) {
		g.state = stateRunning // tapped outside the panel: close it

		return
	}

	i := int(float64(in.tapAt.Y-l.content.Min.Y) / l.rowH)
	if i < 0 || i >= len(s.rows) {
		return
	}

	s.selected = i
	dir := 0

	if s.steps(i) {
		// Left half steps down, right half steps up.
		dir = 1
		if in.tapAt.X < l.content.Min.X+l.content.Dx()/2 {
			dir = -1
		}
	}

	g.applySetting(s.change(i, dir))
}

// applySetting carries out what a settings change needs, and saves it.
func (g *Game) applySetting(eff settingEffect) {
	s := g.settings

	switch eff {
	case effectLive:
		g.machine.SetQuirks(s.profile.Quirks)
		g.machine.SetTicksPerFrame(s.profile.TicksPerFrame)
		g.saveOverrides(s.profile)
	case effectRestart:
		g.saveOverrides(s.profile)

		if err := g.run(g.current, g.currentROM); err != nil {
			g.message = err.Error()
		}

		g.state = stateSettings
	case effectTrace:
		if s.trace {
			g.machine.EnableTrace(traceLength)
		} else {
			g.machine.EnableTrace(0)
		}
	case effectCopy:
		if tr := g.machine.Trace(); tr != nil {
			if err := platform.CopyText(chip8.FormatTrace(g.machine.Profile().Platform, tr.Entries())); err != nil {
				g.message = "Couldn't copy the trace: " + err.Error()
			} else {
				g.message = "Trace copied."
			}
		}
	case effectClose:
		g.state = stateRunning
	case effectSkin:
		g.setSkin(s.skin)
	case effectNone:
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

// functionKeys: F1 statistics, F3 calibration outlines, F5 (natively)
// reload skin.json.
func (g *Game) functionKeys() {
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyF1):
		g.debug = !g.debug
	case inpututil.IsKeyJustPressed(ebiten.KeyF3):
		g.calibrate = !g.calibrate
	case inpututil.IsKeyJustPressed(ebiten.KeyF5) && g.SkinFile != nil:
		g.reloadSkin()
	}
}

// reloadSkin rereads skin.json, keeping the art and controls, so its
// numbers can be tuned while looking at the outlines.
func (g *Game) reloadSkin() {
	data, err := g.SkinFile(g.tv.dir)
	if err == nil {
		var sk *skin.Skin
		if sk, err = skin.Parse(data); err == nil {
			g.tv.skin = sk
			g.relayout()
			g.message = "skin.json reloaded."

			return
		}
	}

	g.message = err.Error()
}

// switchOn switches the TV on: back to the game if there is one, else the
// library.
func (g *Game) switchOn() {
	if g.hasROM {
		g.state = stateRunning

		return
	}

	g.openPicker()
}

// setSkin switches the TV to skin i and saves the choice.
func (g *Game) setSkin(i int) {
	if i < 0 || i >= len(g.skins) || i == g.skin {
		return
	}

	g.skin = i
	g.tv.setSkin(g.skins[i].skin, g.skins[i].dir)
	g.Prefs.Set(skinPrefKey, g.skins[i].dir)
	g.relayout()
}

// relayout places the TV, the picture, the remote and the overlays for the
// current size.
func (g *Game) relayout() {
	window := image.Rect(0, 0, g.width, g.height)
	g.tv.relayout(window, g.scale, textSize*lineSpacing*g.scale, g.ui.touch)

	bg := g.tv.skin.Colour()
	g.ui.light = int(bg[0])*3+int(bg[1])*6+int(bg[2]) > 0xFF*10/2

	l := g.tv.lay
	g.renderer.SetScreen(l.tube)
	g.renderer.SetPicture(pictureIn(l.tube, g.current.Rotation))
	g.osd.resize(l.tube)
	g.remote.layout(l.remote)
	g.ui.resize(window, l.below, g.scale)
	g.picker.setView(float64(g.ui.layout.list.Dy()), g.ui.layout.rowH)
	g.search.bounds, g.search.face, g.search.pad = g.ui.layout.search, g.ui.face, int(panelPad*g.scale/2)
	g.publishLayout()
}

// pageLayout is what the browser tests read (window.chippieLayout), in
// real pixels: [x, y, w, h] rectangles and [x, y] centres.
type pageLayout struct {
	Scale    float64           `json:"scale"`
	Art      bool              `json:"art"` // the cabinet is drawn and the controls work
	Cabinet  [4]int            `json:"cabinet"`
	Tube     [4]int            `json:"tube"`
	Picture  [4]int            `json:"picture"`
	Library  [4]int            `json:"library"`  // tap: the ROM list
	Settings [4]int            `json:"settings"` // tap: the settings panel
	Controls map[string][2]int `json:"controls"`
	Status   pageStatus        `json:"status"`
	Remote   map[string][2]int `json:"remote,omitempty"` // hex key → centre
}

// pageStatus is what the TV is doing, for the browser tests. It is
// comparable, so Update can tell cheaply whether to publish again.
type pageStatus struct {
	State   int     `json:"state"` // the state constant
	Title   string  `json:"title"` // the game playing or tuning in
	Channel int     `json:"channel"`
	Volume  float64 `json:"volume"`
	Tint    int     `json:"tint"`
	Speed   int     `json:"speed"` // instructions per frame
	Skin    string  `json:"skin"`  // the TV skin's folder
}

// status is the current pageStatus.
func (g *Game) status() pageStatus {
	title := g.current.Title
	if g.state == stateLoading {
		title = g.loadingEntry.Title
	}

	c := g.tv.controls

	return pageStatus{
		State: int(g.state), Title: title, Channel: c.channel, Volume: c.volume, Tint: c.tint,
		Speed: g.machine.Profile().TicksPerFrame, Skin: g.tv.dir,
	}
}

func box(r image.Rectangle) [4]int { return [4]int{r.Min.X, r.Min.Y, r.Dx(), r.Dy()} }

// publishLayout tells the page where things are (platform.PublishLayout).
func (g *Game) publishLayout() {
	l := g.tv.lay
	pl := pageLayout{
		Scale: g.scale, Art: g.tv.art != nil,
		Cabinet: box(l.cabinet), Tube: box(l.tube), Picture: box(g.renderer.Picture()),
		Library: box(g.ui.layout.statusLeft), Settings: box(g.ui.layout.statusRight),
		Controls: map[string][2]int{},
		Status:   g.status(),
	}

	g.published = pl.Status

	for _, c := range g.tv.skin.Controls {
		cx, cy := c.Centre()
		x, y := l.fromCabinet(cx, cy)
		pl.Controls[c.ID] = [2]int{int(x), int(y)}
	}

	if !g.remote.area.Empty() {
		pl.Remote = map[string][2]int{}

		for hex, k := range g.remote.keys {
			c := k.Min.Add(k.Max).Div(2)
			pl.Remote[fmt.Sprintf("%X", hex)] = [2]int{c.X, c.Y}
		}
	}

	if data, err := json.Marshal(pl); err == nil {
		platform.PublishLayout(string(data))
	}
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
