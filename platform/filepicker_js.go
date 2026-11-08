//go:build js && wasm

package platform

import (
	"context"
	"fmt"
	"syscall/js"
)

// maxPickSize bounds a picked file: no CHIP-8 ROM comes close.
const maxPickSize = 1 << 20

// PickFile opens a hidden <input type="file"> and reads the chosen file
// with arrayBuffer(). See filepicker.go.
func PickFile() <-chan PickedFile {
	ch := make(chan PickedFile, 1)
	doc := js.Global().Get("document")

	input := doc.Call("createElement", "input")
	input.Set("type", "file")
	input.Set("accept", ".ch8,.c8,.sc8,.xo8,.8o,.bin,application/octet-stream")
	input.Get("style").Set("display", "none")
	doc.Get("body").Call("appendChild", input)

	var onChange js.Func

	onChange = js.FuncOf(func(js.Value, []js.Value) any {
		defer onChange.Release()
		defer input.Call("remove")

		files := input.Get("files")
		if files.Get("length").Int() == 0 {
			return nil
		}

		file := files.Index(0)
		name := file.Get("name").String()

		if size := file.Get("size").Int(); size > maxPickSize {
			ch <- PickedFile{Name: name, Err: fmt.Errorf("%s: %w", name, ErrTooLarge)}

			return nil
		}

		// arrayBuffer() is a promise: wait for it in a goroutine, so this
		// event handler returns at once.
		go func() {
			buf, err := await(context.Background(), js.Global().Get("AbortController").New(), file.Call("arrayBuffer"))
			if err != nil {
				ch <- PickedFile{Name: name, Err: err}

				return
			}

			data := make([]byte, buf.Get("byteLength").Int())
			js.CopyBytesToGo(data, js.Global().Get("Uint8Array").New(buf))

			ch <- PickedFile{Name: name, Data: data}
		}()

		return nil
	})

	input.Call("addEventListener", "change", onChange)
	input.Call("click")

	return ch
}

// CanPickFiles reports whether PickFile works in this build.
func CanPickFiles() bool { return true }
