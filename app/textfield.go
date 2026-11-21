package app

import (
	"image"
	"image/color"
	"math"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/exp/textinput"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/rbscholtus/chippie/platform"
)

// Text field colours.
var (
	fieldBackground = color.RGBA{R: 0x26, G: 0x22, B: 0x1C, A: 0xFF} //nolint:gochecknoglobals // read-only colour
	fieldBorder     = color.RGBA{R: 0x77, G: 0x6A, B: 0x55, A: 0xFF} //nolint:gochecknoglobals // read-only colour
)

// caretBlinkTicks is half the caret's blink period.
const caretBlinkTicks = 30

// Key repeat while a key is held: after repeatDelay ticks, every
// repeatInterval ticks (like a keyboard's own repeat).
const (
	repeatDelay    = 30
	repeatInterval = 3
)

// pressedOrRepeat reports a key press this tick, or a repeat while it's
// held.
func pressedOrRepeat(k ebiten.Key) bool {
	if inpututil.IsKeyJustPressed(k) {
		return true
	}

	d := inpututil.KeyPressDuration(k)

	return d > repeatDelay && (d-repeatDelay)%repeatInterval == 0
}

// textField is a one-line text field. On touch screens it uses Ebitengine's
// exp/textinput (from the prototype): the platform's input method, so phones
// show their keyboard and IME composition works. With a physical keyboard it
// reads typed characters directly: the input method's asynchronous session
// can lose the first keys typed after focusing.
type textField struct {
	bounds      image.Rectangle
	face        *text.GoTextFace
	placeholder string
	pad         int

	text        string
	caret       int // byte offset into text
	composition string
	focused     bool
	ime         bool // use the input method (touch screens)
	changed     bool // the text changed; the owner reads and clears it

	composer textinput.Composer
	chars    []rune

	// domSeen: the input method's textarea has had the DOM focus since the
	// field was focused, so the phone's keyboard opened for it. When the
	// textarea loses the DOM focus after that, the user closed the
	// keyboard (DESIGN §13).
	domSeen bool

	// The phone's keyboard as watchKeyboard last saw it: shown, and whether
	// that could be told.
	keyboard         keyboardWatch
	kbShown, kbKnown bool

	// The keyboard box (DESIGN §13): frames the keyboard has stayed hidden
	// while the textarea has the DOM focus, and where the box is in CSS
	// pixels (empty while hidden).
	stuck int
	box   image.Rectangle

	cssScale float64 // device pixels per CSS pixel; the owner sets it

	// The browser, from platform; tests replace them.
	textInputFocused func() (bool, bool)          // focused, known
	visibleHeight    func() (float64, bool, bool) // height, portrait, known
	showKeyboardBox  func(x, y, w, h float64)
	hideKeyboardBox  func()
}

// stuckFrames is how long the keyboard may stay hidden while the textarea
// has the DOM focus before the field offers its keyboard box: well over the
// time the keyboard takes to open (about 0.3 s).
const stuckFrames = 30

// keyboardShare: the phone's keyboard counts as shown while the visible
// height is below this share of the tallest seen. The keyboard takes about
// half of it; the toolbars that come and go in Safari take far less.
const keyboardShare = 0.85

// keyboardWatch tells whether a phone's on-screen keyboard is showing, from
// the visible area's height (DESIGN §13). It compares with the tallest
// height seen in the same orientation, because the browsers shrink
// different things: Safari on iOS only the visual viewport (775 to 426 CSS
// pixels on an iPhone 16 Plus), Chrome on iOS the whole window (742 to 401),
// so comparing with the window's height never sees Chrome's keyboard.
type keyboardWatch struct {
	tallest [2]float64 // landscape, portrait
}

// see notes a visible height and reports whether the keyboard seems shown.
func (w *keyboardWatch) see(height float64, portrait bool) bool {
	i := 0
	if portrait {
		i = 1
	}

	w.tallest[i] = max(w.tallest[i], height)

	return height < keyboardShare*w.tallest[i]
}

func newTextField(placeholder string) *textField {
	f := &textField{
		placeholder:      placeholder,
		textInputFocused: platform.TextInputFocused,
		visibleHeight:    platform.VisibleHeight,
		showKeyboardBox:  platform.ShowKeyboardBox,
		hideKeyboardBox:  platform.HideKeyboardBox,
	}
	f.composer.OnNewSession = func() *textinput.SessionOptions {
		return &textinput.SessionOptions{
			CaretBounds:     f.caretBounds(),
			TextBeforeCaret: f.text[:f.caret],
			TextAfterCaret:  f.text[f.caret:],
		}
	}
	f.composer.OnComposition = func(c *textinput.Composition) {
		if c.Text() != f.composition {
			f.composition = c.Text()
			f.changed = true // phone keyboards keep words in composition while typing
		}
	}
	f.composer.OnCommit = func(c *textinput.Commit) {
		if before, after := c.IsSurroundingTextReplaced(); before || after {
			b, a := c.SurroundingText()
			f.setText(b+c.Text()+a, len(b)+len(c.Text()))
		} else {
			f.insert(c.Text())
		}

		f.composition = ""
	}
	f.composer.OnEndByUser = f.blur

	return f
}

// value is the text as shown, including any word still being composed.
func (f *textField) value() string { return f.text[:f.caret] + f.composition + f.text[f.caret:] }

// focus gives the field the focus. On touch screens the keyboard opens in
// the next touchend, so the owner focuses the field on the press that
// starts a tap, not when the tap ends (DESIGN §13).
func (f *textField) focus() {
	if !f.focused {
		f.domSeen = false
	}

	f.focused = true
}

// blur takes the focus away and ends the input method's session at once.
// Ebitengine then moves the DOM focus to the canvas, which closes the
// phone's keyboard, even when update stops running because the drawer
// closed.
func (f *textField) blur() {
	f.focused = false
	f.stuck = 0
	f.hideBox()
	f.composer.Confirm()
}

// watchKeyboard looks at the phone's keyboard; it runs every frame while
// the field is focused on a touch screen. Seeing the visible height from
// the press that focuses the field, before that tap's touchend opens the
// keyboard, gives keyboardWatch its tallest height.
//
// While the keyboard stays hidden though the textarea has the DOM focus, it
// puts the keyboard box over the field: Chrome on iOS hides the keyboard so
// with its keyboard-down key, and then ignores every focus() from code,
// even during a tap, until a finger taps a real text box. The next tap on
// the field lands on the box, which hands the focus to the textarea, and
// the keyboard opens (DESIGN §13).
func (f *textField) watchKeyboard() {
	h, portrait, known := f.visibleHeight()

	f.kbKnown = known
	f.kbShown = known && f.keyboard.see(h, portrait)

	if dom, domKnown := f.textInputFocused(); domKnown && dom && f.kbKnown && !f.kbShown {
		f.stuck++
	} else {
		f.stuck = 0
	}

	if f.stuck < stuckFrames {
		f.hideBox()

		return
	}

	if r := f.cssBounds(); r != f.box {
		f.box = r
		f.showKeyboardBox(float64(r.Min.X), float64(r.Min.Y), float64(r.Dx()), float64(r.Dy()))
	}
}

// hideBox hides the keyboard box if it shows.
func (f *textField) hideBox() {
	if f.box.Empty() {
		return
	}

	f.box = image.Rectangle{}
	f.hideKeyboardBox()
}

// cssBounds is the field's rectangle in CSS pixels.
func (f *textField) cssBounds() image.Rectangle {
	s := f.cssScale
	if s <= 0 {
		s = 1
	}

	css := func(v int) int { return int(math.Round(float64(v) / s)) }

	return image.Rect(css(f.bounds.Min.X), css(f.bounds.Min.Y), css(f.bounds.Max.X), css(f.bounds.Max.Y))
}

// keyboardClosed reports whether the user closed the phone's keyboard: the
// textarea had the DOM focus for this field and lost it (Safari's done
// button does this). It notes when the textarea has the focus.
func (f *textField) keyboardClosed() bool {
	dom, known := f.textInputFocused()

	switch {
	case !known:
		return false
	case dom:
		f.domSeen = true

		return false
	default:
		return f.domSeen
	}
}

// setText replaces the text and puts the caret at byte offset caret.
func (f *textField) setText(s string, caret int) {
	if s != f.text {
		f.changed = true
	}

	f.text, f.caret = s, min(max(caret, 0), len(s))
}

// insert types s at the caret.
func (f *textField) insert(s string) {
	f.setText(f.text[:f.caret]+s+f.text[f.caret:], f.caret+len(s))
}

// backspace deletes the character before the caret.
func (f *textField) backspace() {
	if f.caret == 0 {
		return
	}

	_, n := utf8.DecodeLastRuneInString(f.text[:f.caret])
	f.setText(f.text[:f.caret-n]+f.text[f.caret:], f.caret-n)
}

// moveCaret moves the caret one character left (-1) or right (+1).
func (f *textField) moveCaret(dir int) {
	switch {
	case dir < 0 && f.caret > 0:
		_, n := utf8.DecodeLastRuneInString(f.text[:f.caret])
		f.caret -= n
	case dir > 0 && f.caret < len(f.text):
		_, n := utf8.DecodeRuneInString(f.text[f.caret:])
		f.caret += n
	}
}

// update handles typing while focused.
func (f *textField) update() error {
	if !f.focused || !f.ime {
		f.composer.Confirm()
	}

	if !f.focused {
		return nil
	}

	if f.ime {
		f.watchKeyboard()
	}

	// Without this, the field would ask for a new session at once, and the
	// next tap anywhere would bring the keyboard back.
	if f.ime && f.keyboardClosed() {
		f.blur()

		return nil
	}

	if f.ime {
		handled, err := f.composer.Update()
		if err != nil || handled {
			return err //nolint:wrapcheck // Ebitengine's input error, passed up unchanged
		}
	} else {
		f.chars = ebiten.AppendInputChars(f.chars[:0])
		if len(f.chars) > 0 {
			f.insert(string(f.chars))
		}
	}

	switch {
	case pressedOrRepeat(ebiten.KeyBackspace):
		f.composer.Confirm()
		f.backspace()
	case pressedOrRepeat(ebiten.KeyArrowLeft):
		f.composer.Confirm()
		f.moveCaret(-1)
	case pressedOrRepeat(ebiten.KeyArrowRight):
		f.composer.Confirm()
		f.moveCaret(1)
	}

	return nil
}

func (f *textField) lineHeight() int {
	m := f.face.Metrics()

	return int(m.HAscent + m.HDescent)
}

func (f *textField) caretBounds() image.Rectangle {
	if f.face == nil {
		return image.Rectangle{}
	}

	line := f.text[:f.caret] + f.composition + f.text[f.caret:]
	x := f.bounds.Min.X + f.pad + int(text.AdvanceAt(line, f.caret+len(f.composition), f.face))
	y := f.bounds.Min.Y + (f.bounds.Dy()-f.lineHeight())/2

	return image.Rect(x, y, x+1, y+f.lineHeight())
}

func (f *textField) draw(dst *ebiten.Image) {
	r := f.bounds
	border := fieldBorder

	if f.focused {
		border = accentColour
	}

	vector.FillRect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), fieldBackground, false)
	vector.StrokeRect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), 2, border, false)

	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(r.Min.X+f.pad), float64(r.Min.Y+(r.Dy()-f.lineHeight())/2))

	s := f.value()
	if s == "" && !f.focused {
		s = f.placeholder

		op.ColorScale.ScaleWithColor(mutedColour)
	} else {
		op.ColorScale.ScaleWithColor(inkColour)
	}

	text.Draw(dst, s, f.face, op)

	if f.focused && (ebiten.Tick()/caretBlinkTicks)%2 == 0 {
		c := f.caretBounds()
		vector.StrokeLine(dst, float32(c.Min.X), float32(c.Min.Y), float32(c.Min.X), float32(c.Max.Y), 2, border, false)
	}
}
