//go:build js && wasm

package platform

import "syscall/js"

// PublishLayout puts a JSON description of the app's layout on the page as
// window.chippieLayout, so the browser tests can find the picture and the
// controls instead of repeating the layout maths.
func PublishLayout(jsonText string) {
	js.Global().Set("chippieLayout", jsonText)
}

// TouchScreen reports whether the browser says it has a touch screen
// (navigator.maxTouchPoints), so the remote can show before the first touch.
func TouchScreen() bool {
	n := js.Global().Get("navigator").Get("maxTouchPoints")

	return n.Type() == js.TypeNumber && n.Int() > 0
}
