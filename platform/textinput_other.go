//go:build !(js && wasm)

package platform

// TextInputFocused can't tell natively, where no DOM holds the focus.
func TextInputFocused() (bool, bool) { return false, false }

// VisibleHeight can't tell natively; there is no on-screen keyboard.
func VisibleHeight() (float64, bool, bool) { return 0, false, false }

// ShowKeyboardBox does nothing natively.
func ShowKeyboardBox(_, _, _, _ float64) {}

// HideKeyboardBox does nothing natively.
func HideKeyboardBox() {}
