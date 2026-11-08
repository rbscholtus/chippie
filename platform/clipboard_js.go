//go:build js && wasm

package platform

import (
	"fmt"
	"syscall/js"
)

// CopyText puts s on the clipboard. Browsers allow it during a user gesture
// (a click or tap); the copy finishes in the background.
func CopyText(s string) error {
	clip := js.Global().Get("navigator").Get("clipboard")
	if clip.IsUndefined() || clip.IsNull() {
		return fmt.Errorf("%w: no clipboard", ErrUnsupported)
	}

	clip.Call("writeText", s)

	return nil
}
