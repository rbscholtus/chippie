package platform

import "errors"

// ErrUnsupported reports a feature this build doesn't have (the file
// picker natively: use -rom or drag and drop instead).
var ErrUnsupported = errors.New("platform: not supported here")

// PickedFile is the result of the file picker.
type PickedFile struct {
	Name string
	Data []byte
	Err  error
}

// PickFile opens the browser's file picker and delivers the chosen file on
// the channel (which has room for it). Call it straight after a click or
// tap: browsers only open the picker during a user gesture. If the user
// cancels, nothing is ever sent.
//
// func PickFile() <-chan PickedFile is in filepicker_js.go and
// filepicker_other.go.
