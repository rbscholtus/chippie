//go:build !(js && wasm)

package platform

// PublishLayout does nothing natively: only the browser tests read it.
func PublishLayout(string) {}

// TouchScreen is false natively: a touch, when one comes, turns on the
// touch layout.
func TouchScreen() bool { return false }
