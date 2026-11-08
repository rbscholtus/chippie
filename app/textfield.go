package app

import (
	"image"
	"image/color"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/exp/textinput"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
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
}

func newTextField(placeholder string) *textField {
	f := &textField{placeholder: placeholder}
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

func (f *textField) focus() { f.focused = true }
func (f *textField) blur()  { f.focused = false }

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
