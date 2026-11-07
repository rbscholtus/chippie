package main

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

// imeField is a single-line text field built on Ebitengine's exp/textinput.
// Unlike ebitenui's TextInput, it uses the platform IME, which in browsers
// is backed by a hidden textarea, so phones should show their keyboard.
type imeField struct {
	bounds      image.Rectangle
	face        text.Face
	placeholder string
	text        string
	caret       int // byte offset
	focused     bool
	composer    textinput.Composer
	composition string
	onChange    func(string)
}

func newIMEField(face text.Face, placeholder string, onChange func(string)) *imeField {
	f := &imeField{face: face, placeholder: placeholder, onChange: onChange}
	f.composer.OnNewSession = func() *textinput.SessionOptions {
		return &textinput.SessionOptions{
			CaretBounds:     f.caretBounds(),
			TextBeforeCaret: f.text[:f.caret],
			TextAfterCaret:  f.text[f.caret:],
		}
	}
	f.composer.OnComposition = func(c *textinput.Composition) { f.composition = c.Text() }
	f.composer.OnCommit = func(c *textinput.Commit) {
		if before, after := c.IsSurroundingTextReplaced(); before || after {
			b, a := c.SurroundingText()
			f.text = b + c.Text() + a
			f.caret = len(b) + len(c.Text())
		} else {
			f.text = f.text[:f.caret] + c.Text() + f.text[f.caret:]
			f.caret += len(c.Text())
		}
		f.composition = ""
		f.changed()
	}
	f.composer.OnEndByUser = f.blur
	return f
}

func (f *imeField) changed() {
	if f.onChange != nil {
		f.onChange(f.text)
	}
}

func (f *imeField) focus() { f.focused = true }
func (f *imeField) blur()  { f.focused = false }

func (f *imeField) lineHeight() int {
	m := f.face.Metrics()
	return int(m.HAscent + m.HDescent)
}

func (f *imeField) caretBounds() image.Rectangle {
	x := f.bounds.Min.X + 8 + int(text.Advance(f.text[:f.caret]+f.composition, f.face))
	y := f.bounds.Min.Y + (f.bounds.Dy()-f.lineHeight())/2
	return image.Rect(x, y, x+1, y+f.lineHeight())
}

func (f *imeField) update() error {
	if !f.focused {
		f.composer.Confirm()
		return nil
	}
	handled, err := f.composer.Update()
	if err != nil || handled {
		return err
	}
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && f.caret > 0:
		f.composer.Confirm()
		_, l := utf8.DecodeLastRuneInString(f.text[:f.caret])
		f.text = f.text[:f.caret-l] + f.text[f.caret:]
		f.caret -= l
		f.changed()
	case inpututil.IsKeyJustPressed(ebiten.KeyLeft) && f.caret > 0:
		f.composer.Confirm()
		_, l := utf8.DecodeLastRuneInString(f.text[:f.caret])
		f.caret -= l
	case inpututil.IsKeyJustPressed(ebiten.KeyRight) && f.caret < len(f.text):
		f.composer.Confirm()
		_, l := utf8.DecodeRuneInString(f.text[f.caret:])
		f.caret += l
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		f.composer.Cancel()
		f.blur()
	}
	return nil
}

func (f *imeField) draw(dst *ebiten.Image) {
	r := f.bounds
	bg := color.RGBA{0x26, 0x22, 0x1c, 0xff}
	border := color.RGBA{0x77, 0x6a, 0x55, 0xff}
	if f.focused {
		border = color.RGBA{0xff, 0xb0, 0x30, 0xff}
	}
	vector.FillRect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), bg, false)
	vector.StrokeRect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), 2, border, false)

	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(r.Min.X+8), float64(r.Min.Y+(r.Dy()-f.lineHeight())/2))
	s := f.text[:f.caret] + f.composition + f.text[f.caret:]
	if s == "" && !f.focused {
		s = f.placeholder
		op.ColorScale.ScaleWithColor(color.RGBA{0x88, 0x80, 0x70, 0xff})
	} else {
		op.ColorScale.ScaleWithColor(color.RGBA{0xf4, 0xea, 0xd8, 0xff})
	}
	text.Draw(dst, s, f.face, op)

	if f.focused && (ebiten.Tick()/30)%2 == 0 {
		c := f.caretBounds()
		vector.StrokeLine(dst, float32(c.Min.X), float32(c.Min.Y), float32(c.Min.X), float32(c.Max.Y), 2, border, false)
	}
}
