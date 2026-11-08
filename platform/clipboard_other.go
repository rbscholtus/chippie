//go:build !(js && wasm)

package platform

// CopyText isn't supported natively.
func CopyText(string) error { return ErrUnsupported }
