//go:build js && wasm

package platform

import (
	"fmt"
	"strings"
	"syscall/js"
)

// Ebitengine's exp/textinput types into a hidden textarea (and parks the
// focus on a second, "dummy" one while it restarts the input method). These
// ids are Ebitengine internals (v2.10.6, exp/textinput/textinput_js.go);
// DESIGN §13 explains why the app looks at them.
const (
	textInputID      = "ebitengine-textinput"
	textInputDummyID = "ebitengine-textinput-dummy"
)

// TextInputFocused reports whether one of the text input's textareas has
// the DOM focus: while it has, the phone's keyboard belongs to the search
// field. The second result is false where this can't be told (natively).
func TextInputFocused() (bool, bool) {
	id := js.Global().Get("document").Get("activeElement")
	if !id.Truthy() {
		return false, true
	}

	s := id.Get("id").String()

	return s == textInputID || s == textInputDummyID, true
}

// VisibleHeight reports the height of the page's visible area (the visual
// viewport) in CSS pixels, and whether the device is held upright. An
// on-screen keyboard takes its space from this area. The third result is
// false without the visualViewport API.
func VisibleHeight() (float64, bool, bool) {
	w := js.Global()

	vv := w.Get("visualViewport")
	if !vv.Truthy() {
		return 0, false, false
	}

	return vv.Get("height").Float(), upright(w), true
}

// upright reports the device's orientation, not the window's shape: Chrome
// on iOS shrinks the window when the keyboard opens (430×742 to 430×401), so
// a window wider than tall doesn't mean landscape.
func upright(w js.Value) bool {
	if o := w.Get("screen").Get("orientation"); o.Truthy() {
		return strings.HasPrefix(o.Get("type").String(), "portrait")
	}

	if a := w.Get("orientation"); a.Type() == js.TypeNumber { // older iOS: degrees
		return a.Int()%180 == 0
	}

	return true
}

// keyboardBoxID is the keyboard box's element id.
const keyboardBoxID = "chippie-keyboard-box"

// ShowKeyboardBox puts the keyboard box, an invisible but real text box,
// over the rectangle x, y, w, h in CSS pixels (DESIGN §13). Chrome on iOS
// ignores focus() from code after its keyboard-down key, but a finger
// tapping a real text box focuses it natively and the keyboard opens. The
// box then hands the focus straight to Ebitengine's textarea. It stays
// until HideKeyboardBox, so a tap that doesn't bring the keyboard can be
// tried again.
func ShowKeyboardBox(x, y, w, h float64) {
	doc := js.Global().Get("document")

	box := doc.Call("getElementById", keyboardBoxID)
	if !box.Truthy() {
		box = newKeyboardBox(doc)
	}

	s := box.Get("style")
	s.Set("left", fmt.Sprintf("%gpx", x))
	s.Set("top", fmt.Sprintf("%gpx", y))
	s.Set("width", fmt.Sprintf("%gpx", w))
	s.Set("height", fmt.Sprintf("%gpx", h))
	s.Set("display", "block")
}

// HideKeyboardBox hides the keyboard box, if there is one.
func HideKeyboardBox() {
	if box := js.Global().Get("document").Call("getElementById", keyboardBoxID); box.Truthy() {
		box.Get("style").Set("display", "none")
	}
}

// newKeyboardBox makes the keyboard box, hidden, once per page. Its focus
// handler lives as long as the page, so it's never released.
func newKeyboardBox(doc js.Value) js.Value {
	box := doc.Call("createElement", "input")
	box.Set("id", keyboardBoxID)
	box.Set("type", "text")
	box.Set("tabIndex", -1)
	box.Set("spellcheck", false)
	box.Call("setAttribute", "autocapitalize", "off")
	box.Call("setAttribute", "autocomplete", "off")
	box.Call("setAttribute", "aria-hidden", "true")
	// 16 px text: iOS zooms into a focused text box with smaller text.
	box.Get("style").Set("cssText", "position:fixed;display:none;z-index:1;margin:0;padding:0;border:0;"+
		"opacity:0;background:transparent;color:transparent;caret-color:transparent;font-size:16px")

	box.Call("addEventListener", "focus", js.FuncOf(func(js.Value, []js.Value) any {
		if t := doc.Call("getElementById", textInputID); t.Truthy() {
			t.Call("focus")
		}

		return nil
	}))
	doc.Get("body").Call("appendChild", box)

	return box
}
