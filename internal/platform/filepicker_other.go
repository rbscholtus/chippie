//go:build !(js && wasm)

package platform

// PickFile reports ErrUnsupported natively. See filepicker.go.
func PickFile() <-chan PickedFile {
	ch := make(chan PickedFile, 1)
	ch <- PickedFile{Err: ErrUnsupported}

	return ch
}

// CanPickFiles reports whether PickFile works in this build.
func CanPickFiles() bool { return false }
