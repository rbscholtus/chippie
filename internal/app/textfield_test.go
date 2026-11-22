package app

import (
	"image"
	"testing"
)

func TestTextFieldEditing(t *testing.T) {
	t.Parallel()

	f := newTextField("search")
	f.insert("héllo")

	if f.text != "héllo" || f.caret != len("héllo") || !f.changed {
		t.Fatalf("insert: %q caret %d changed %v", f.text, f.caret, f.changed)
	}

	f.changed = false
	f.moveCaret(-1)
	f.moveCaret(-1)
	f.moveCaret(-1)
	f.moveCaret(-1) // before "é"
	f.backspace()   // deletes "h"

	if f.text != "éllo" || f.caret != 0 || !f.changed {
		t.Errorf("backspace: %q caret %d", f.text, f.caret)
	}

	f.moveCaret(1) // over the two-byte "é"

	if f.caret != len("é") {
		t.Errorf("caret %d, want %d", f.caret, len("é"))
	}

	f.backspace()
	f.backspace() // nothing left before the caret

	if f.text != "llo" || f.caret != 0 {
		t.Errorf("after backspaces: %q caret %d", f.text, f.caret)
	}

	f.moveCaret(-1) // already at the start
	f.setText("abc", 99)

	if f.caret != 3 {
		t.Errorf("caret clamped to %d, want 3", f.caret)
	}
}

func TestTextFieldValueIncludesComposition(t *testing.T) {
	t.Parallel()

	f := newTextField("search")
	f.insert("ab")
	f.moveCaret(-1)
	f.composition = "XY"

	if got := f.value(); got != "aXYb" {
		t.Errorf("value %q, want aXYb", got)
	}

	f.focus()

	if !f.focused {
		t.Error("focus")
	}

	f.blur()

	if f.focused {
		t.Error("blur")
	}
}

// fakeBrowser stands in for the browser's focus, visible area and keyboard
// box.
type fakeBrowser struct {
	dom, known   bool
	height       float64 // the visible area's height, upright
	box          image.Rectangle
	shows, hides int
}

func (b *fakeBrowser) attach(f *textField) {
	f.textInputFocused = func() (bool, bool) { return b.dom, b.known }
	f.visibleHeight = func() (float64, bool, bool) { return b.height, true, b.known }
	f.showKeyboardBox = func(x, y, w, h float64) {
		b.shows++
		b.box = image.Rect(int(x), int(y), int(x+w), int(y+h))
	}
	f.hideKeyboardBox = func() { b.hides++; b.box = image.Rectangle{} }
}

// frames runs n frames of the field's keyboard watch.
func frames(f *textField, n int) {
	for range n {
		f.watchKeyboard()
	}
}

func TestTextFieldNoticesTheKeyboardClosing(t *testing.T) {
	t.Parallel()

	f := newTextField("search")
	b := &fakeBrowser{known: true}
	b.attach(f)
	f.ime = true
	f.focus()

	if f.keyboardClosed() {
		t.Fatal("closed before the textarea ever had the focus (the start is still deferred)")
	}

	b.dom = true // this tap's touchend focused the textarea

	if f.keyboardClosed() {
		t.Fatal("closed while the textarea has the focus")
	}

	b.dom = false // Safari's done button

	if !f.keyboardClosed() {
		t.Error("the textarea lost the focus: want closed")
	}

	f.blur()
	f.focus() // tapped again: a new start, nothing seen yet

	if f.keyboardClosed() {
		t.Error("closed right after focusing again")
	}
}

// The heights are Chrome's on an iPhone 16 Plus (DESIGN §13): its keyboard
// shrinks the whole window, and its keyboard-down key hides the keyboard
// while the textarea keeps the DOM focus.
func TestTextFieldOffersTheKeyboardBox(t *testing.T) {
	t.Parallel()

	f := newTextField("search")
	b := &fakeBrowser{known: true, height: 742}
	b.attach(f)
	f.ime = true
	f.bounds = image.Rect(96, 168, 1200, 300) // device pixels
	f.cssScale = 3

	f.focus()
	frames(f, 1) // the press's frame, before its touchend

	b.dom = true // the touchend focused the textarea; the keyboard is opening

	frames(f, stuckFrames-1)

	if b.shows != 0 {
		t.Fatal("box shown while the keyboard was still opening")
	}

	b.height = 401 // the keyboard is open

	frames(f, 2*stuckFrames)

	if b.shows != 0 {
		t.Fatal("box shown with the keyboard open")
	}

	b.height = 742 // Chrome's keyboard-down key: the textarea keeps the focus

	frames(f, stuckFrames)

	if want := image.Rect(32, 56, 400, 100); b.shows != 1 || b.box != want {
		t.Fatalf("after keyboard-down: box %v shown %d times, want %v once", b.box, b.shows, want)
	}

	frames(f, stuckFrames)

	if b.shows != 1 {
		t.Errorf("box shown %d times; it was already there", b.shows)
	}

	b.height = 401 // the finger tapped the box; it handed the focus back and the keyboard opened

	frames(f, 1)

	if b.hides != 1 || !b.box.Empty() {
		t.Errorf("keyboard open again: box %v, hidden %d times; want it hidden once", b.box, b.hides)
	}

	b.height = 742 // keyboard-down again

	frames(f, stuckFrames)
	f.blur() // e.g. a game starts

	if b.shows != 2 || b.hides != 2 {
		t.Errorf("blur: shown %d, hidden %d times; want 2 and 2", b.shows, b.hides)
	}
}

func TestKeyboardWatch(t *testing.T) {
	t.Parallel()

	steps := []struct {
		what     string
		height   float64
		portrait bool
		want     bool
	}{
		{"Safari, closed", 775, true, false},
		{"Safari, open (only the visual viewport shrinks)", 426, true, true},
		{"Safari, closed again", 775, true, false},
		{"Safari's toolbars", 742, true, false},
		{"Chrome-like, open (the window shrinks too)", 401, true, true},
		{"turned: its own tallest height", 320, false, false},
		{"turned, open", 150, false, true},
		{"upright again, closed", 775, true, false},
	}

	var w keyboardWatch

	for _, s := range steps {
		if got := w.see(s.height, s.portrait); got != s.want {
			t.Errorf("%s (%.0f): shown %v, want %v", s.what, s.height, got, s.want)
		}
	}
}

func TestTextFieldNativeIgnoresTheDOM(t *testing.T) {
	t.Parallel()

	f := newTextField("search")
	b := &fakeBrowser{known: false}
	b.attach(f)
	f.ime = true
	f.focus()
	frames(f, 2*stuckFrames)

	if f.keyboardClosed() || b.shows != 0 {
		t.Error("natively the field can't see the DOM and must not act on it")
	}
}
