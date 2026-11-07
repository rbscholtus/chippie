// Command chippie-spike is a throwaway prototype answering three questions for
// a pure-Go (Ebitengine) Chippie: can ebitenui host a searchable game library,
// can the CRT effect be done in Kage, and can Go use IndexedDB in the browser.
package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"sort"
	"strings"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/themes"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/goregular"
)

var (
	//go:embed programs.json
	programsJSON []byte
	//go:embed crt.kage
	crtSrc []byte
	//go:embed phosphor.kage
	phosphorSrc []byte
)

const fbW, fbH = 64, 32

type program struct {
	Title    string
	Authors  []string
	Release  string
	Platform string
	search   string
}

func loadPrograms() []*program {
	var raw []struct {
		Title   string                       `json:"title"`
		Authors []string                     `json:"authors"`
		Release string                       `json:"release"`
		ROMs    map[string]struct{ Platforms []string } `json:"roms"`
	}
	if err := json.Unmarshal(programsJSON, &raw); err != nil {
		log.Fatal(err)
	}
	out := make([]*program, 0, len(raw))
	for _, r := range raw {
		p := &program{Title: r.Title, Authors: r.Authors, Release: r.Release}
		for _, rom := range r.ROMs {
			if len(rom.Platforms) > 0 {
				p.Platform = rom.Platforms[0]
				break
			}
		}
		p.search = strings.ToLower(p.Title + " " + strings.Join(p.Authors, " "))
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Title) < strings.ToLower(out[j].Title) })
	return out
}

// 3x5 glyphs for the test pattern, one row per string.
var glyphs = map[rune][5]string{
	'C': {"###", "#..", "#..", "#..", "###"}, 'H': {"#.#", "#.#", "###", "#.#", "#.#"},
	'I': {"###", ".#.", ".#.", ".#.", "###"}, 'P': {"###", "#.#", "###", "#..", "#.."},
	'E': {"###", "#..", "##.", "#..", "###"}, '0': {"###", "#.#", "#.#", "#.#", "###"},
	'1': {".#.", "##.", ".#.", ".#.", "###"}, '2': {"###", "..#", "###", "#..", "###"},
	'3': {"###", "..#", ".##", "..#", "###"}, '4': {"#.#", "#.#", "###", "..#", "..#"},
	'5': {"###", "#..", "###", "..#", "###"}, '6': {"###", "#..", "###", "#.#", "###"},
	'7': {"###", "..#", ".#.", ".#.", ".#."}, '8': {"###", "#.#", "###", "#.#", "###"},
	'9': {"###", "#.#", "###", "..#", "###"},
}

type game struct {
	programs []*program
	visible  []any
	current  *program
	channel  int

	ui       *ebitenui.UI
	list     *widget.List
	count    *widget.Text
	slot     *widget.Container
	ime      *imeField
	query    string
	portrait bool
	built    bool

	face, small text.Face

	fb       [fbH][fbW]bool
	fbImg    *ebiten.Image
	upImg    *ebiten.Image
	phos     [2]*ebiten.Image
	crt      *ebiten.Shader
	phosSh   *ebiten.Shader
	screen   image.Rectangle
	power    float64
	poweredOn bool
	tint     int
	osdUntil int

	ballX, ballY, ballDX, ballDY int

	storeMsg  string
	restored  chan string
	idbStatus chan string
}

var tints = []struct {
	name string
	rgb  [3]float32
}{
	{"Amber", [3]float32{1.0, 0.68, 0.2}},
	{"Green", [3]float32{0.35, 1.0, 0.45}},
	{"White", [3]float32{0.95, 0.95, 1.0}},
	{"Timendus", [3]float32{1.0, 0.8, 0.05}},
}

func (g *game) logical() (int, int) {
	if g.portrait {
		return 540, 960
	}
	return 960, 540
}

func (g *game) Layout(ow, oh int) (int, int) {
	p := oh > ow
	if p != g.portrait || !g.built {
		g.portrait = p
		g.build()
	}
	return g.logical()
}

func (g *game) filter(q string) {
	g.query = strings.ToLower(strings.TrimSpace(q))
	g.visible = g.visible[:0]
	for _, p := range g.programs {
		if g.query == "" || strings.Contains(p.search, g.query) {
			g.visible = append(g.visible, p)
		}
	}
	if g.list != nil {
		g.list.SetEntries(g.visible)
		g.count.Label = fmt.Sprintf("%d of %d programs", len(g.visible), len(g.programs))
	}
}

func (g *game) build() {
	g.built = true
	w, _ := g.logical()
	panelW, listH := 320, 330
	if g.portrait {
		g.screen = image.Rect(40, 40, w-40, 40+(w-80)/2)
		panelW, listH = w, 120
	} else {
		g.screen = image.Rect(40, 40, 600, 320)
	}
	g.fbImg = ebiten.NewImage(fbW, fbH)
	g.upImg = ebiten.NewImage(g.screen.Dx(), g.screen.Dy())
	g.phos[0] = ebiten.NewImage(g.screen.Dx(), g.screen.Dy())
	g.phos[1] = ebiten.NewImage(g.screen.Dx(), g.screen.Dy())

	label := func(s string) *widget.Text {
		return widget.NewText(widget.TextOpts.Text(s, &g.small, color.RGBA{0xd8, 0xcc, 0xb4, 0xff}),
			widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})))
	}

	panel := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(6),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(16)),
		)),
	)
	if g.portrait {
		panel.GetWidget().LayoutData = widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionStart, VerticalPosition: widget.AnchorLayoutPositionStart,
			StretchHorizontal: true, StretchVertical: true,
			Padding: &widget.Insets{Top: g.screen.Max.Y + 150},
		}
	} else {
		panel.GetWidget().LayoutData = widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionStart, VerticalPosition: widget.AnchorLayoutPositionStart,
			StretchHorizontal: true, StretchVertical: true,
			Padding: &widget.Insets{Left: w - panelW},
		}
	}

	panel.AddChild(label("A · ebitenui TextInput"))
	panel.AddChild(widget.NewTextInput(
		widget.TextInputOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
		widget.TextInputOpts.Placeholder("Search title or author"),
		widget.TextInputOpts.ChangedHandler(func(a *widget.TextInputChangedEventArgs) { g.filter(a.InputText) }),
	))
	panel.AddChild(label("B · Ebitengine textinput (IME)"))
	g.slot = widget.NewContainer(widget.ContainerOpts.WidgetOpts(
		widget.WidgetOpts.MinSize(0, 38),
		widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true}),
	))
	panel.AddChild(g.slot)
	g.count = label("")
	panel.AddChild(g.count)

	g.list = widget.NewList(
		widget.ListOpts.ContainerOpts(widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(0, listH),
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true}),
		)),
		widget.ListOpts.HideHorizontalSlider(),
		widget.ListOpts.EntryFontFace(&g.small),
		widget.ListOpts.EntryLabelFunc(func(e any) string {
			p := e.(*program)
			if p.Release != "" {
				return p.Title + "  (" + p.Release + ")"
			}
			return p.Title
		}),
		widget.ListOpts.EntryTextPadding(widget.NewInsetsSimple(5)),
		widget.ListOpts.EntrySelectedHandler(func(a *widget.ListEntrySelectedEventArgs) {
			g.selectProgram(a.Entry.(*program), true)
		}),
	)
	panel.AddChild(g.list)

	root := widget.NewContainer(widget.ContainerOpts.Layout(widget.NewAnchorLayout()))
	root.AddChild(panel)
	g.ui = &ebitenui.UI{Container: root, PrimaryTheme: themes.GetBasicDarkTheme()}

	q := ""
	if g.ime != nil {
		q = g.ime.text
	}
	g.ime = newIMEField(g.small, "Search title or author", g.filter)
	g.ime.text, g.ime.caret = q, len(q)
	g.filter(q)
}

func (g *game) selectProgram(p *program, save bool) {
	g.current = p
	for i, q := range g.programs {
		if q == p {
			g.channel = i + 1
		}
	}
	g.osdUntil = int(ebiten.Tick()) + 180
	if save {
		go func(title string) {
			if err := storePut("last", title); err != nil {
				g.idbStatus <- "IndexedDB write failed: " + err.Error()
				return
			}
			g.idbStatus <- "IndexedDB: saved “" + title + "”"
		}(p.Title)
	}
}

func (g *game) knobs() (power, tint image.Point, r int) {
	s := g.screen
	if g.portrait {
		return image.Pt(s.Min.X+40, s.Max.Y+70), image.Pt(s.Min.X+110, s.Max.Y+70), 22
	}
	return image.Pt(s.Max.X-110, s.Max.Y+70), image.Pt(s.Max.X-40, s.Max.Y+70), 22
}

func pressed() (image.Point, bool) {
	if ids := inpututil.AppendJustPressedTouchIDs(nil); len(ids) > 0 {
		x, y := ebiten.TouchPosition(ids[0])
		return image.Pt(x, y), true
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		return image.Pt(x, y), true
	}
	return image.Point{}, false
}

func within(p, c image.Point, r int) bool {
	dx, dy := p.X-c.X, p.Y-c.Y
	return dx*dx+dy*dy <= r*r
}

func (g *game) Update() error {
	if !g.built {
		return nil
	}
	select {
	case t := <-g.restored:
		for _, p := range g.programs {
			if p.Title == t {
				g.selectProgram(p, false)
				g.storeMsg = "IndexedDB: restored “" + t + "”"
			}
		}
	case s := <-g.idbStatus:
		g.storeMsg = s
	default:
	}

	g.ime.bounds = g.slot.GetWidget().Rect
	if pt, ok := pressed(); ok {
		pw, tn, r := g.knobs()
		switch {
		case pt.In(g.ime.bounds):
			g.ui.ClearFocus()
			g.ime.focus()
		case within(pt, pw, r):
			g.poweredOn = !g.poweredOn
			g.ime.blur()
		case within(pt, tn, r):
			g.tint = (g.tint + 1) % len(tints)
			g.ime.blur()
		default:
			g.ime.blur()
		}
	}
	if err := g.ime.update(); err != nil {
		return err
	}
	g.ui.Update()

	// Power animation and a stand-in for the emulator: a bouncing XOR sprite.
	target := 0.0
	if g.poweredOn {
		target = 1
	}
	g.power += math.Copysign(math.Min(math.Abs(target-g.power), 1.0/40), target-g.power)
	g.stepPattern()
	return nil
}

func (g *game) stepPattern() {
	for y := range g.fb {
		for x := range g.fb[y] {
			g.fb[y][x] = x == 0 || y == 0 || x == fbW-1 || y == fbH-1
		}
	}
	g.drawString(4, 3, "CHIPPIE")
	g.drawString(4, 24, fmt.Sprintf("%03d", g.channel))
	g.ballX += g.ballDX
	g.ballY += g.ballDY
	if g.ballX <= 1 || g.ballX >= fbW-7 {
		g.ballDX = -g.ballDX
	}
	if g.ballY <= 1 || g.ballY >= fbH-7 {
		g.ballDY = -g.ballDY
	}
	ball := [6]byte{0x78, 0xFC, 0xFC, 0xFC, 0xFC, 0x78}
	for r, row := range ball {
		for c := 0; c < 6; c++ {
			if row&(0x80>>c) != 0 {
				g.fb[g.ballY+r][g.ballX+c] = !g.fb[g.ballY+r][g.ballX+c]
			}
		}
	}
}

func (g *game) drawString(x, y int, s string) {
	for _, ch := range s {
		gl, ok := glyphs[ch]
		if ok {
			for r, row := range gl {
				for c, v := range row {
					if v == '#' {
						g.fb[y+r][x+c] = true
					}
				}
			}
		}
		x += 4
	}
}

var pix = make([]byte, fbW*fbH*4)

func (g *game) Draw(dst *ebiten.Image) {
	w, h := g.logical()
	dst.Fill(color.RGBA{0x1b, 0x18, 0x14, 0xff})

	// Framebuffer → texture → nearest-neighbour upscale → phosphor → CRT.
	for y := range g.fb {
		for x := range g.fb[y] {
			v := byte(0)
			if g.fb[y][x] {
				v = 0xff
			}
			i := (y*fbW + x) * 4
			pix[i], pix[i+1], pix[i+2], pix[i+3] = v, v, v, 0xff
		}
	}
	g.fbImg.WritePixels(pix)
	g.upImg.Clear()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(g.screen.Dx())/fbW, float64(g.screen.Dy())/fbH)
	g.upImg.DrawImage(g.fbImg, op)

	if g.current != nil && int(ebiten.Tick()) < g.osdUntil {
		to := &text.DrawOptions{}
		to.GeoM.Translate(16, 12)
		to.ColorScale.ScaleWithColor(color.RGBA{0x40, 0xff, 0x60, 0xff})
		text.Draw(g.upImg, fmt.Sprintf("CH %03d  %s", g.channel, strings.ToUpper(g.current.Title)), g.face, to)
	}

	cur, prev := g.phos[ebiten.Tick()%2], g.phos[(ebiten.Tick()+1)%2]
	so := &ebiten.DrawRectShaderOptions{}
	so.Images[0], so.Images[1] = g.upImg, prev
	so.Uniforms = map[string]any{"Decay": float32(0.82)}
	cur.DrawRectShader(g.screen.Dx(), g.screen.Dy(), g.phosSh, so)

	// The TV: bezel, screen, knobs.
	s := g.screen
	vector.FillRect(dst, float32(s.Min.X-28), float32(s.Min.Y-28), float32(s.Dx()+56), float32(s.Dy()+130), color.RGBA{0x5a, 0x3b, 0x22, 0xff}, true)
	vector.FillRect(dst, float32(s.Min.X-14), float32(s.Min.Y-14), float32(s.Dx()+28), float32(s.Dy()+28), color.RGBA{0x14, 0x12, 0x10, 0xff}, true)
	t := tints[g.tint].rgb
	co := &ebiten.DrawRectShaderOptions{}
	co.GeoM.Translate(float64(s.Min.X), float64(s.Min.Y))
	co.Images[0] = cur
	co.Uniforms = map[string]any{
		"Time":  float32(ebiten.Tick()) / 60,
		"Power": float32(g.power),
		"Tint":  []float32{t[0], t[1], t[2]},
	}
	dst.DrawRectShader(s.Dx(), s.Dy(), g.crt, co)

	pw, tn, r := g.knobs()
	for i, k := range []image.Point{pw, tn} {
		vector.FillCircle(dst, float32(k.X), float32(k.Y), float32(r), color.RGBA{0x2a, 0x26, 0x22, 0xff}, true)
		vector.StrokeCircle(dst, float32(k.X), float32(k.Y), float32(r), 2, color.RGBA{0xb8, 0x9a, 0x6a, 0xff}, true)
		a := float64(g.tint)*math.Pi/2 - math.Pi/2
		if i == 0 {
			a = -math.Pi / 2
			if !g.poweredOn {
				a = math.Pi
			}
		}
		vector.StrokeLine(dst, float32(k.X), float32(k.Y), float32(k.X)+float32(math.Cos(a))*float32(r-5), float32(k.Y)+float32(math.Sin(a))*float32(r-5), 3, color.RGBA{0xf4, 0xea, 0xd8, 0xff}, true)
		lo := &text.DrawOptions{}
		lo.GeoM.Translate(float64(k.X-r), float64(k.Y+r+4))
		lo.ColorScale.ScaleWithColor(color.RGBA{0xd8, 0xcc, 0xb4, 0xff})
		name := "POWER"
		if i == 1 {
			name = "TINT"
		}
		text.Draw(dst, name, g.small, lo)
	}

	// Info + diagnostics.
	info := "Pick a game from the library."
	if g.current != nil {
		info = fmt.Sprintf("%s · %s · %s", g.current.Title, strings.Join(g.current.Authors, ", "), g.current.Platform)
	}
	diag := fmt.Sprintf("TPS %.0f  FPS %.0f  scale %.1fx  %dx%d logical  tint %s\n%s",
		ebiten.ActualTPS(), ebiten.ActualFPS(), ebiten.Monitor().DeviceScaleFactor(), w, h, tints[g.tint].name, g.storeMsg)
	io := &text.DrawOptions{}
	io.LineSpacing = 18
	if g.portrait {
		io.GeoM.Translate(float64(s.Min.X+170), float64(s.Max.Y+48))
	} else {
		io.GeoM.Translate(float64(s.Min.X-20), float64(s.Max.Y+120))
	}
	io.ColorScale.ScaleWithColor(color.RGBA{0xd8, 0xcc, 0xb4, 0xff})
	text.Draw(dst, info+"\n"+diag, g.small, io)

	g.ui.Draw(dst)
	g.ime.draw(dst)
}

func main() {
	src, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		log.Fatal(err)
	}
	crt, err := ebiten.NewShader(crtSrc)
	if err != nil {
		log.Fatal(err)
	}
	phos, err := ebiten.NewShader(phosphorSrc)
	if err != nil {
		log.Fatal(err)
	}
	g := &game{
		programs:  loadPrograms(),
		face:      &text.GoTextFace{Source: src, Size: 20},
		small:     &text.GoTextFace{Source: src, Size: 14},
		crt:       crt,
		phosSh:    phos,
		poweredOn: true,
		ballX:     30, ballY: 12, ballDX: 1, ballDY: 1,
		restored:  make(chan string, 1),
		idbStatus: make(chan string, 4),
		storeMsg:  "IndexedDB: checking…",
	}
	g.visible = make([]any, 0, len(g.programs))
	go func() {
		t, err := storeGet("last")
		switch {
		case err != nil:
			g.idbStatus <- "IndexedDB unavailable: " + err.Error()
		case t == "":
			g.idbStatus <- "IndexedDB: ok, nothing saved yet"
		default:
			g.restored <- t
		}
	}()

	ebiten.SetWindowSize(960, 540)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Chippie spike")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
